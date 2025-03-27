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

	if locksVerboseLogging {
		log.Printf("[Queue] Getting queue for plan %s", planId)
	}

	q, ok := m[planId]
	if !ok {
		if locksVerboseLogging {
			log.Printf("[Queue] Creating new queue for plan %s", planId)
		}
		q = &repoQueue{}
		m[planId] = q
	}
	return q
}

func (m repoQueueMap) add(op *repoOperation) int {
	if locksVerboseLogging {
		log.Printf("[Queue] Adding operation %s (%s) to queue for plan %s", op.id, op.reason, op.planId)
	}
	q := m.getQueue(op.planId)
	return q.add(op)
}

// Add enqueues an operation, and then kicks off processing if needed.
func (q *repoQueue) add(op *repoOperation) int {
	var numOps int
	q.mu.Lock()
	q.ops = append(q.ops, op)
	numOps = len(q.ops)

	if locksVerboseLogging {
		log.Printf("[Queue] Operation %s (%s) enqueued, queue length now %d", op.id, op.reason, numOps)
	}

	// If nobody else is processing, we'll start
	if !q.isProcessing {
		if locksVerboseLogging {
			log.Printf("[Queue] Starting queue processing for operation %s (%s)", op.id, op.reason)
		}
		q.isProcessing = true
		go q.runQueue() // run in the background
	} else if locksVerboseLogging {
		log.Printf("[Queue] Queue already processing, operation %s (%s) will wait", op.id, op.reason)
	}
	q.mu.Unlock()

	return numOps
}

func (q *repoQueue) nextBatch() []*repoOperation {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.ops) == 0 {
		if locksVerboseLogging {
			log.Printf("[Queue] No operations in queue")
		}
		return nil
	}

	firstOp := q.ops[0]
	res := []*repoOperation{firstOp}

	if locksVerboseLogging {
		log.Printf("[Queue] Processing first operation %s (%s) with scope %s, branch %s",
			firstOp.id, firstOp.reason, firstOp.scope, firstOp.branch)
	}

	q.ops = q.ops[1:]

	// writes always go one at a time, blocking everything else, as do read locks on the root plan (no branch)
	if firstOp.scope == LockScopeWrite || firstOp.branch == "" {
		if locksVerboseLogging {
			log.Printf("[Queue] Operation %s is write or root branch read, processing alone", firstOp.id)
		}
		return res
	}

	// reads go in parallel as long as they are on the same branch
	for len(q.ops) > 0 {
		op := q.ops[0]
		if op.scope == LockScopeRead && op.branch == firstOp.branch {
			if locksVerboseLogging {
				log.Printf("[Queue] Batching compatible read operation %s (%s) with same branch %s",
					op.id, op.reason, op.branch)
			}
			res = append(res, op)
			q.ops = q.ops[1:]
		} else {
			if locksVerboseLogging {
				log.Printf("[Queue] Operation %s (%s) with scope %s, branch %s not compatible with batch, stopping",
					op.id, op.reason, op.scope, op.branch)
			}
			break
		}
	}

	if locksVerboseLogging {
		log.Printf("[Queue] Created batch of %d operations", len(res))
	}

	return res
}

func (q *repoQueue) runQueue() {
	if locksVerboseLogging {
		log.Printf("[Queue] Starting queue processing")
	}

	for {
		// get the next batch
		ops := q.nextBatch()
		if len(ops) == 0 {
			// Nothing left in the queue, so mark not processing and return
			if locksVerboseLogging {
				log.Printf("[Queue] Queue empty, stopping processing")
			}
			q.mu.Lock()
			q.isProcessing = false
			q.mu.Unlock()
			return
		}

		firstOp := ops[0]

		func() {

			if locksVerboseLogging {
				log.Printf("[Queue] Attempting to acquire DB lock for plan %s, branch %s, scope %s",
					firstOp.planId, firstOp.branch, firstOp.scope)
			}

			lockId, err := lockRepoDB(LockRepoParams{
				OrgId:       firstOp.orgId,
				UserId:      firstOp.userId,
				PlanId:      firstOp.planId,
				Branch:      firstOp.branch,
				Scope:       firstOp.scope,
				PlanBuildId: firstOp.planBuildId,
				Reason:      firstOp.reason,
				Ctx:         firstOp.ctx,
				CancelFn:    firstOp.cancelFn,
			}, 0)

			if lockId != "" {
				log.Printf("[Queue] Acquired DB lock %s", lockId)

				defer func() {
					log.Printf("[Queue] Releasing DB lock %s for plan %s", lockId, firstOp.planId)
					releaseErr := deleteRepoLockDB(lockId, firstOp.planId, firstOp.reason, 0)
					if releaseErr != nil {
						log.Printf("[Queue] Failed to release DB lock: %v", releaseErr)
					} else {
						log.Printf("[Queue] DB lock %s released successfully", lockId)
					}
				}()
			}

			if err != nil {
				log.Printf("[Queue] Failed to get DB lock: %v", err)
				for _, op := range ops {
					if locksVerboseLogging {
						log.Printf("[Queue] Notifying operation %s (%s) of lock failure", op.id, op.reason)
					}
					op.done <- fmt.Errorf("failed to get DB lock: %w", err)
				}
				// we still need to process the rest of the queue
				// if the error is critical, caller will handle it
				return
			}

			if locksVerboseLogging {
				log.Printf("[Queue] Acquired DB lock %s, processing batch of %d operations", lockId, len(ops))
			}

			repo := getGitRepo(firstOp.orgId, firstOp.planId)
			var needsRollback bool

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
						if locksVerboseLogging {
							log.Printf("[Queue] Operation %s (%s) context canceled", op.id, op.reason)
						}
						op.done <- op.ctx.Err()
					default:
						if locksVerboseLogging {
							log.Printf("[Queue] Starting operation %s (%s)", op.id, op.reason)
						}
						// actually do the operation

						var opErr error

						func() {
							defer func() {
								panicErr := recover()
								if panicErr != nil {
									log.Printf("[Queue] Panic in operation %s (%s): %v", op.id, op.reason, panicErr)
									opErr = fmt.Errorf("panic in operation: %v", panicErr)
								}

								if opErr != nil && op.scope == LockScopeWrite && op.clearRepoOnErr {
									if locksVerboseLogging {
										log.Printf("[Queue] Operation %s (%s) failed with error, marking for rollback: %v",
											op.id, op.reason, opErr)
									}
									needsRollback = true
								}
							}()

							if locksVerboseLogging {
								log.Printf("[Queue] Executing operation %s (%s)", op.id, op.reason)
							}
							opErr = op.op(repo)
							if locksVerboseLogging {
								if opErr != nil {
									log.Printf("[Queue] Operation %s (%s) failed with error: %v", op.id, op.reason, opErr)
								} else {
									log.Printf("[Queue] Operation %s (%s) completed successfully", op.id, op.reason)
								}
							}
						}()

						// signal to the caller via op.done
						if locksVerboseLogging {
							log.Printf("[Queue] Notifying caller of operation %s (%s) completion", op.id, op.reason)
						}
						op.done <- opErr
					}
				}(op)
			}
			wg.Wait()

			if needsRollback {
				log.Printf("[Queue] Performing rollback for plan %s branch %s", firstOp.planId, firstOp.branch)
				rollbackErr := repo.GitClearUncommittedChanges(firstOp.branch)
				if rollbackErr != nil {
					log.Printf("[Queue] Failed to rollback: %v", rollbackErr)
				} else if locksVerboseLogging {
					log.Printf("[Queue] Rollback completed successfully")
				}
			}
		}()
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

	log.Printf("[Queue] ExecRepoOperation called for plan %s, branch %s, scope %s, reason %s",
		params.PlanId, params.Branch, params.Scope, params.Reason)

	done := make(chan error, 1)
	numOps := repoQueues.add(&repoOperation{
		id:             id,
		orgId:          params.OrgId,
		planId:         params.PlanId,
		branch:         params.Branch,
		scope:          params.Scope,
		reason:         params.Reason,
		planBuildId:    params.PlanBuildId,
		op:             op,
		done:           done,
		ctx:            params.Ctx,
		cancelFn:       params.CancelFn,
		clearRepoOnErr: params.ClearRepoOnErr,
	})

	if numOps > 1 {
		if locksVerboseLogging {
			log.Printf("[Queue] Operation %s (%s) queued behind %d operations", id, params.Reason, numOps-1)
			for i, op := range repoQueues.getQueue(params.PlanId).ops {
				log.Printf("[Queue] Operation %d: %s - %s\n", i, op.id, op.reason)
			}
		}
	}

	select {
	case err := <-done:
		if locksVerboseLogging {
			if err != nil {
				log.Printf("[Queue] Operation %s (%s) completed with error: %v", id, params.Reason, err)
			} else {
				log.Printf("[Queue] Operation %s (%s) completed successfully", id, params.Reason)
			}
		}
		return err
	case <-params.Ctx.Done():
		if locksVerboseLogging {
			log.Printf("[Queue] Operation %s (%s) context canceled while waiting", id, params.Reason)
		}
		return params.Ctx.Err()
	}
}
