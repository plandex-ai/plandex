package db

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
)

type repoOpFn func(repo *GitRepo) error

type repoOperation struct {
	orgId          string
	userId         string
	planId         string
	branch         string
	scope          LockScope
	planBuildId    string
	id             string
	reason         string
	op             repoOpFn
	ctx            context.Context
	cancelFn       context.CancelFunc
	done           chan error
	clearRepoOnErr bool
}

type repoQueue struct {
	ops          []*repoOperation
	mu           sync.Mutex
	isProcessing bool
}

type repoQueueMap map[string]*repoQueue

var queuesMu sync.Mutex
var repoQueues = make(repoQueueMap)

func (m repoQueueMap) getQueue(planId string) *repoQueue {
	queuesMu.Lock()
	defer queuesMu.Unlock()

	q, ok := m[planId]
	if !ok {
		q = &repoQueue{}
		m[planId] = q
	}
	return q
}

func (m repoQueueMap) add(op *repoOperation) int {
	q := m.getQueue(op.planId)
	return q.add(op)
}

// Add enqueues an operation, and then kicks off processing if needed.
func (q *repoQueue) add(op *repoOperation) int {
	var numOps int
	q.mu.Lock()
	q.ops = append(q.ops, op)
	numOps = len(q.ops)

	// If nobody else is processing, we’ll start
	if !q.isProcessing {
		q.isProcessing = true
		go q.runQueue() // run in the background
	}
	q.mu.Unlock()

	return numOps
}

func (q *repoQueue) nextBatch() []*repoOperation {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.ops) == 0 {
		return nil
	}

	firstOp := q.ops[0]
	res := []*repoOperation{firstOp}

	q.ops = q.ops[1:]

	// writes always go one at a time, blocking everything else, as do read locks on the root plan (no branch)
	if firstOp.scope == LockScopeWrite || firstOp.branch == "" {
		return res
	}

	// reads go in parallel as long as they are on the same branch
	for len(q.ops) > 0 {
		op := q.ops[0]
		if op.scope == LockScopeRead && op.branch == firstOp.branch {
			res = append(res, op)
			q.ops = q.ops[1:]
		} else {
			break
		}
	}

	return res
}

func (q *repoQueue) runQueue() {
	for {
		// get the next batch
		ops := q.nextBatch()
		if len(ops) == 0 {
			// Nothing left in the queue, so mark not processing and return
			q.mu.Lock()
			q.isProcessing = false
			q.mu.Unlock()
			return
		}

		log.Printf("processing batch of %d operations", len(ops))

		// Process the batch
		// If it's a writer => single op
		// If multiple same‐branch readers => do them in parallel
		var wg sync.WaitGroup
		for _, op := range ops {
			wg.Add(1)
			go func(op *repoOperation) {
				defer wg.Done()
				select {
				case <-op.ctx.Done():
					op.done <- op.ctx.Err()
				default:
					log.Printf("processing operation %s - (%s)", op.id, op.reason)

					lockId, err := lockRepoDB(LockRepoParams{
						OrgId:       op.orgId,
						UserId:      op.userId,
						PlanId:      op.planId,
						Branch:      op.branch,
						Scope:       op.scope,
						PlanBuildId: op.planBuildId,
						Ctx:         op.ctx,
						CancelFn:    op.cancelFn,
					}, 0)

					if err != nil {
						log.Printf("failed to get DB lock: %v", err)
						op.done <- fmt.Errorf("failed to get DB lock: %w", err)
						return
					}

					// actually do the operation
					repo := getGitRepo(op.orgId, op.planId)
					var opErr error

					func() {
						defer func() {
							panicErr := recover()
							if panicErr != nil {
								log.Printf("panic in operation: %v", panicErr)
								opErr = fmt.Errorf("panic in operation: %v", panicErr)
							}

							if opErr != nil && op.scope == LockScopeWrite && op.clearRepoOnErr {
								err := repo.GitClearUncommittedChanges(op.branch)
								if err != nil {
									log.Printf("failed to clear repo after error: %v", err)
								}
							}
						}()

						opErr = op.op(repo)
					}()

					releaseErr := deleteRepoLockDB(lockId, op.planId, 0)
					if releaseErr != nil {
						log.Printf("failed to release DB lock: %v", releaseErr)
						op.done <- fmt.Errorf("failed to release DB lock: %w", releaseErr)
						return
					}

					// signal to the caller via op.done
					op.done <- opErr
				}
			}(op)
		}
		wg.Wait()
	}
}

type ExecRepoOperationParams struct {
	OrgId          string
	UserId         string
	PlanId         string
	Branch         string
	Scope          LockScope
	PlanBuildId    string
	Reason         string
	Ctx            context.Context
	CancelFn       context.CancelFunc
	ClearRepoOnErr bool
}

func ExecRepoOperation(
	params ExecRepoOperationParams,
	op repoOpFn,
) error {
	id := uuid.New().String()
	done := make(chan error)
	numOps := repoQueues.add(&repoOperation{
		id:             id,
		orgId:          params.OrgId,
		planId:         params.PlanId,
		branch:         params.Branch,
		scope:          params.Scope,
		reason:         params.Reason,
		op:             op,
		done:           done,
		clearRepoOnErr: params.ClearRepoOnErr,
	})

	if numOps > 1 {
		log.Printf("operation %s - (%s) queued behind %d operations", id, params.Reason, numOps-1)
	}

	select {
	case err := <-done:
		return err
	case <-params.Ctx.Done():
		return params.Ctx.Err()
	}
}
