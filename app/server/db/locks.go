package db

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

const lockHeartbeatInterval = 3 * time.Second
const lockHeartbeatTimeout = 60 * time.Second
const maxRetries = 10
const initialRetryDelay = 100 * time.Millisecond
const backoffFactor = 2.0  // Exponential base
const jitterFraction = 0.3 // e.g. ±30% of the backoff
const localLockTimeout = 30 * time.Second

// LockRepoParams holds the data needed for your lock calls
type LockRepoParams struct {
	OrgId       string
	UserId      string
	PlanId      string
	Branch      string
	Scope       LockScope
	PlanBuildId string
	Ctx         context.Context
	CancelFn    context.CancelFunc
}

// -----------------------------------------------------------------------------
// Local "Branch‐Aware" RW Lock
// -----------------------------------------------------------------------------

// planLocksMu protects the global map of planId -> *PlanLockState
var planLocksMu sync.Mutex

// planLocks tracks all plan concurrency states in this process
var planLocks = make(map[string]*PlanLockState)

// PlanLockState tracks concurrency for one plan, with its own mutex and cond.
type PlanLockState struct {
	mu            sync.Mutex // Protects the fields below
	cond          *sync.Cond
	currentBranch string
	readCount     int
	hasWriter     bool
}

// getOrCreatePlanLockState fetches or creates a PlanLockState for a planId.
func getOrCreatePlanLockState(planId string) *PlanLockState {
	planLocksMu.Lock()
	defer planLocksMu.Unlock()

	pls := planLocks[planId]
	if pls == nil {
		pls = &PlanLockState{}
		pls.cond = sync.NewCond(&pls.mu)
		planLocks[planId] = pls
	}
	return pls
}

// acquireLocalLock attempts to acquire local concurrency for (planId, branch, scope).
// Blocks until successful or context canceled.
func acquireLocalLock(ctx context.Context, planId, branch string, scope LockScope) error {
	pls := getOrCreatePlanLockState(planId)

	pls.mu.Lock()
	defer pls.mu.Unlock()

	for {
		// If there's an active writer, no read or write can proceed
		if pls.hasWriter {
			// Wait, but also check context
			if err := waitOrCtx(ctx, pls); err != nil {
				return err
			}
			continue
		}

		if scope == LockScopeRead {
			// If we have existing readers, must match branch
			if pls.readCount > 0 {
				if pls.currentBranch != branch {
					if err := waitOrCtx(ctx, pls); err != nil {
						return err
					}
					continue
				}
			} else {
				// No readers yet => set the branch
				pls.currentBranch = branch
			}
			// Acquire a read slot
			pls.readCount++
			return nil

		} else {
			// scope == LockScopeWrite
			// Write can only proceed if no readers and no writer
			if pls.readCount > 0 {
				if err := waitOrCtx(ctx, pls); err != nil {
					return err
				}
				continue
			}
			if pls.hasWriter {
				if err := waitOrCtx(ctx, pls); err != nil {
					return err
				}
				continue
			}
			// Acquire the writer
			pls.hasWriter = true
			pls.currentBranch = branch
			return nil
		}
	}
}

// releaseLocalLock frees local concurrency for (planId, branch, scope).
func releaseLocalLock(planId, branch string, scope LockScope) {
	planLocksMu.Lock()
	pls, ok := planLocks[planId]
	if !ok {
		planLocksMu.Unlock()
		return
	}
	planLocksMu.Unlock()

	pls.mu.Lock()
	defer pls.mu.Unlock()

	if pls.currentBranch != branch {
		log.Printf("WARN: Releasing lock for branch=%s, but planLockState has branch=%s",
			branch, pls.currentBranch)
	}

	if scope == LockScopeRead {
		pls.readCount--
		if pls.readCount < 0 {
			log.Printf("BUG: readCount < 0 for plan %s", planId)
			pls.readCount = 0
		}
	} else {
		pls.hasWriter = false
	}

	if pls.readCount == 0 && !pls.hasWriter {
		pls.currentBranch = ""
		// Optionally remove from the map entirely
		planLocksMu.Lock()
		delete(planLocks, planId)
		planLocksMu.Unlock()
	}

	pls.cond.Broadcast()
}

// waitOrCtx calls pls.cond.Wait() but also checks if ctx is canceled.
func waitOrCtx(ctx context.Context, pls *PlanLockState) error {
	// Since we already hold pls.mu, we do:
	if ctx.Err() != nil {
		// context canceled?
		return ctx.Err()
	}

	// Then do the standard cond.Wait() call
	pls.cond.Wait()

	// Once we return from Wait(), we hold pls.mu again
	// The caller can re-check conditions in its loop
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func LockRepo(params LockRepoParams) (string, error) {
	var attempt int
	for {
		// Step 1: Local concurrency
		ctx, cancel := context.WithTimeout(params.Ctx, localLockTimeout)
		defer cancel()

		err := acquireLocalLock(ctx, params.PlanId, params.Branch, params.Scope)
		if err == context.DeadlineExceeded {
			return "", fmt.Errorf("local lock acquisition timed out after 30s: %w", err)
		}

		// Step 2: Attempt DB lock
		id, err := lockRepoDB(params, 0)
		if err != nil {
			// Release local if DB fails
			releaseLocalLock(params.PlanId, params.Branch, params.Scope)

			if ctxErr := params.Ctx.Err(); ctxErr != nil {
				return "", ctxErr
			}
			attempt++
			if attempt > maxRetries {
				return "", fmt.Errorf("DB lock conflict too many times: %v", err)
			}
			// Exponential backoff
			backoffSleep := time.Duration(float64(initialRetryDelay) * math.Pow(backoffFactor, float64(attempt)))
			time.Sleep(backoffSleep)
			continue
		}
		// success
		return id, nil
	}
}

func DeleteRepoLock(id, planId string) error {
	// Try to fetch scope + branch from DB
	var scope LockScope
	var branch *string
	err := Conn.QueryRow("SELECT scope, branch FROM repo_locks WHERE id = $1", id).Scan(&scope, &branch)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("error getting lock info: %v", err)
	}

	// Do the DB remove first
	dbErr := deleteRepoLockDB(id, planId, 0)

	// Then free local concurrency
	if branch != nil && scope != "" {
		releaseLocalLock(planId, *branch, scope)
	}

	return dbErr
}

func lockRepoDB(params LockRepoParams, numRetry int) (string, error) {
	start := time.Now()
	goroutineID := getGoroutineID() // You'll need to implement this

	log.Printf("[Lock][%d] START lock attempt for plan %s scope %s (retry %d) at %v",
		goroutineID, params.PlanId, params.Scope, numRetry, start)

	defer func() {
		elapsed := time.Since(start)
		log.Printf("[Lock][%d] END lock attempt took %v", goroutineID, elapsed)
	}()

	log.Printf("locking repo. orgId: %s | planId: %s | scope: %s | branch %s | numRetry %d \n", params.OrgId, params.PlanId, params.Scope, params.Branch, numRetry)
	// spew.Dump(params)

	stack := debug.Stack()
	// Format truncated stack excluding runtime frames
	stackTrace := formatStackTrace(stack)

	log.Println()
	log.Printf("LOCK_ATTEMPT | params: %+v | retry: %d | stack:\n%s", params, numRetry, stackTrace)

	// ensure context did not cancel
	if params.Ctx.Err() != nil {
		log.Printf("[Lock][%d] Context canceled, returning error: %v", goroutineID, params.Ctx.Err())
		return "", params.Ctx.Err()
	}

	initialJitter := time.Duration(rand.Int63n(int64(5000 * time.Microsecond)))
	time.Sleep(initialJitter)

	orgId := params.OrgId
	userId := params.UserId
	planId := params.PlanId
	branch := params.Branch
	scope := params.Scope
	planBuildId := params.PlanBuildId
	ctx := params.Ctx
	cancelFn := params.CancelFn

	if orgId == "" {
		return "", fmt.Errorf("orgId is required")
	}
	if planId == "" {
		return "", fmt.Errorf("planId is required")
	}
	if scope != LockScopeRead && scope != LockScopeWrite {
		return "", fmt.Errorf("invalid lock scope: %s", scope)
	}

	tx, err := Conn.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		log.Printf("[Lock][%d] Error starting transaction %v",
			goroutineID, err)
		return "", fmt.Errorf("error starting transaction: %v", err)
	}

	var committed bool

	// Ensure that rollback is attempted in case of failure
	defer func() {
		if committed {
			return
		}

		if rbErr := tx.Rollback(); rbErr != nil {
			if rbErr == sql.ErrTxDone {
				log.Println("attempted to roll back transaction, but it was already committed")
			} else {
				log.Printf("transaction rollback error: %v\n", rbErr)
			}
		} else {
			log.Println("transaction rolled back")
		}
	}()

	forUpdate := params.Scope == LockScopeWrite

	selectStart := time.Now()
	if forUpdate {
		log.Printf("[Lock][%d] Starting SELECT FOR UPDATE at %v", goroutineID, selectStart)
	} else {
		log.Printf("[Lock][%d] Starting SELECT FOR SHARE at %v", goroutineID, selectStart)
	}

	lockablePlanIdQuery := "SELECT * FROM lockable_plan_ids WHERE plan_id = $1"
	if forUpdate {
		lockablePlanIdQuery += " FOR UPDATE"
	} else {
		lockablePlanIdQuery += " FOR SHARE"
	}

	_, err = tx.Exec(lockablePlanIdQuery, planId)
	if err != nil {
		return retryWithExponentialBackoff(params.Ctx, err, numRetry, func(nextAttempt int) (string, error) {
			return lockRepoDB(params, nextAttempt)
		})
	}

	query := "SELECT id, org_id, user_id, plan_id, plan_build_id, scope, branch, last_heartbeat_at, created_at FROM repo_locks WHERE plan_id = $1"
	if forUpdate {
		query += " FOR UPDATE"
	} else {
		query += " FOR SHARE"
	}
	queryArgs := []interface{}{planId}

	var locks []*repoLock
	log.Println("obtaining repo lock with query")
	repoLockRows, err := tx.Query(query, queryArgs...)
	if err != nil {
		return retryWithExponentialBackoff(params.Ctx, err, numRetry, func(nextAttempt int) (string, error) {
			return lockRepoDB(params, nextAttempt)
		})
	}
	log.Println("repo lock query executed")

	if forUpdate {
		log.Printf("[Lock][%d] SELECT FOR UPDATE took %v",
			goroutineID, time.Since(selectStart))
	} else {
		log.Printf("[Lock][%d] SELECT FOR SHARE took %v",
			goroutineID, time.Since(selectStart))
	}

	defer repoLockRows.Close()

	var expiredLockIds []string

	now := time.Now()
	for repoLockRows.Next() {
		var lock repoLock
		if err := repoLockRows.Scan(&lock.Id, &lock.OrgId, &lock.UserId, &lock.PlanId, &lock.PlanBuildId, &lock.Scope, &lock.Branch, &lock.LastHeartbeatAt, &lock.CreatedAt); err != nil {
			return "", fmt.Errorf("error scanning repo lock: %v", err)
		}

		// ensure heartbeat hasn't timed out
		if now.Sub(lock.LastHeartbeatAt) < lockHeartbeatTimeout {
			locks = append(locks, &lock)
		} else {
			expiredLockIds = append(expiredLockIds, lock.Id)
		}
	}

	if err := repoLockRows.Err(); err != nil {
		return "", fmt.Errorf("error iterating over repo locks: %v", err)
	}

	if len(expiredLockIds) > 0 {
		log.Printf("deleting expired locks: %v", expiredLockIds)

		query := "DELETE FROM repo_locks WHERE id = ANY($1)"
		_, err := tx.Exec(query, pq.Array(expiredLockIds))
		if err != nil {
			if isDeadlockError(err) {
				log.Println("deadlock clearing expired locks, won't do anything")
			} else {
				return "", fmt.Errorf("error removing expired locks: %v", err)
			}
		}
	}

	canAcquire := true

	for _, lock := range locks {
		lockBranch := ""
		if lock.Branch != nil {
			lockBranch = *lock.Branch
		}

		if scope == LockScopeRead {
			// if we're trying to acquire a read lock, we can do so unless there's a conflicting lock
			// a write lock always conflicts with a read lock (regardless of branch)
			// a read lock conflicts if it's for a different branch (since it would need to checkout a different branch in the middle of an already-running read)
			if lock.Scope == LockScopeWrite {
				canAcquire = false
				break
			} else if lock.Scope == LockScopeRead {
				if lockBranch != branch {
					canAcquire = false
					break
				}
			}
		} else if scope == LockScopeWrite {
			// if we're trying to acquire a write lock, we can only do so if there's no other lock (read or write)
			canAcquire = false
			break
		} else {
			err = fmt.Errorf("invalid lock scope: %v", scope)
			return "", err
		}
	}

	if !canAcquire {
		log.Println("can't acquire lock.", "numRetry:", numRetry)
		conflictErr := errors.New("lock conflict: cannot acquire read/write lock")
		return retryWithExponentialBackoff(params.Ctx, conflictErr, numRetry, func(nextAttempt int) (string, error) {
			return lockRepoDB(params, nextAttempt)
		})
	}

	log.Println("can acquire lock - inserting new lock")

	insertStart := time.Now()
	log.Printf("[Lock][%d] Starting INSERT at %v", goroutineID, insertStart)

	// Insert the new lock
	var lockPlanBuildId *string
	if planBuildId != "" {
		lockPlanBuildId = &planBuildId
	}

	var lockBranch *string
	if branch != "" {
		lockBranch = &branch
	}

	newLock := &repoLock{
		PlanId:      planId,
		OrgId:       orgId,
		PlanBuildId: lockPlanBuildId,
		Scope:       scope,
		Branch:      lockBranch,
	}

	if userId != "" {
		newLock.UserId = &userId
	}

	// log.Println("newLock:")
	// spew.Dump(newLock)

	var insertedId sql.NullString

	insertQuery := "INSERT INTO repo_locks (org_id, user_id, plan_id, plan_build_id, scope, branch) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (plan_id) WHERE scope = 'w' DO NOTHING RETURNING id"

	log.Printf("Insert query: %s", insertQuery)

	err = tx.QueryRow(
		insertQuery,
		newLock.OrgId,
		newLock.UserId,
		newLock.PlanId,
		newLock.PlanBuildId,
		newLock.Scope,
		newLock.Branch,
	).Scan(&insertedId)
	if err != nil {
		if err == sql.ErrNoRows {
			// Means ON CONFLICT DO NOTHING prevented insertion
			// => concurrency conflict => backoff & retry
			return retryWithExponentialBackoff(params.Ctx,
				errors.New("lock conflict: row not inserted"),
				numRetry,
				func(nextAttempt int) (string, error) {
					return lockRepoDB(params, nextAttempt)
				},
			)
		}

		return "", fmt.Errorf("error inserting new lock: %v", err)
	}

	if insertedId.Valid {
		newLock.Id = insertedId.String
	} else {
		log.Printf("no rows returned from insert query, means there was a conflict")
		return retryWithExponentialBackoff(params.Ctx, err, numRetry, func(nextAttempt int) (string, error) {
			return lockRepoDB(params, nextAttempt)
		})
	}

	log.Printf("[Lock][%d] INSERT took %v",
		goroutineID, time.Since(insertStart))

	// check if git lock file exists
	// remove it if so
	err = gitRemoveIndexLockFileIfExists(getPlanDir(orgId, planId))
	if err != nil {
		return "", fmt.Errorf("error removing lock file: %v", err)
	}

	// branches, err := GitListBranches(orgId, planId)
	// if err != nil {
	// 	return "", fmt.Errorf("error getting branches: %v", err)
	// }

	// log.Println("branches:", branches)

	if branch != "" {
		// checkout the branch
		err = gitCheckoutBranch(getPlanDir(orgId, planId), branch)
		if err != nil {
			return "", fmt.Errorf("error checking out branch: %v", err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("error committing transaction: %v", err)
	}

	committed = true

	log.Println()
	log.Printf("LOCK_ACQUIRED | id: %s | params: %+v | stack:\n%s", newLock.Id, params, stackTrace)
	log.Println()

	// Start a goroutine to keep the lock alive
	go func() {
		numErrors := 0
		for {
			select {
			case <-ctx.Done():
				// log.Printf("case <-stream.Ctx.Done(): %s\n", newLock.Id)
				err := DeleteRepoLock(newLock.Id, planId)
				if err != nil {
					log.Printf("Error unlocking repo: %v\n", err)
				}
				return

			default:
				jitter := time.Duration(rand.Int63n(int64(float64(lockHeartbeatInterval) * 0.1)))
				time.Sleep(lockHeartbeatInterval + jitter)

				res, err := Conn.Exec("UPDATE repo_locks SET last_heartbeat_at = NOW() WHERE id = $1", newLock.Id)

				if err != nil {
					log.Printf("Error updating repo lock last heartbeat: %v\n", err)

					if isDeadlockError(err) {
						log.Println("Deadlock error, keep retrying")
					} else {
						numErrors++
					}

					if numErrors > 5 {
						log.Printf("Too many errors updating repo lock last heartbeat: %v\n", err)
						cancelFn()
						return
					}
				} else {
					// check if 0 rows were updated
					rowsAffected, err := res.RowsAffected()
					if err != nil {
						log.Printf("Error getting rows affected: %v\n", err)
						cancelFn()
						return
					}

					if rowsAffected == 0 {
						log.Printf("Lock not found: %s | stopping heartbeat loop\n", newLock.Id)
						return
					}
				}
			}

		}
	}()

	log.Println("repo locked. id:", newLock.Id)

	return newLock.Id, nil
}

func deleteRepoLockDB(id, planId string, numRetry int) error {
	start := time.Now()
	goroutineID := getGoroutineID()

	log.Printf("[Delete][%d] START delete lock %s at %v", goroutineID, id, start)
	stack := debug.Stack()
	// Format truncated stack excluding runtime frames
	stackTrace := formatStackTrace(stack)
	log.Printf("DELETE_ATTEMPT | id: %s | stack:\n%s", id, stackTrace)

	initialJitter := time.Duration(rand.Int63n(int64(5000 * time.Microsecond)))
	time.Sleep(initialJitter)

	defer func() {
		elapsed := time.Since(start)
		log.Printf("[Delete][%d] END delete lock took %v", goroutineID, elapsed)
	}()

	var committed bool

	// Start a new transaction for the delete
	tx, err := Conn.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		log.Printf("[Delete][%d] Error starting delete transaction: %v", goroutineID, err)
		return fmt.Errorf("error starting delete transaction: %v", err)
	}
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	deleteStart := time.Now()
	log.Printf("[Delete][%d] Starting DELETE at %v", goroutineID, deleteStart)

	lockablePlanIdQuery := "SELECT * FROM lockable_plan_ids WHERE plan_id = $1 FOR UPDATE"
	_, err = tx.Exec(lockablePlanIdQuery, planId)
	if err != nil {
		if isDeadlockError(err) {
			log.Printf("[Delete][%d] Deadlock error, retrying delete %v",
				goroutineID, err)
			_, err := retryWithExponentialBackoff(context.Background(), err, numRetry, func(nextAttempt int) (string, error) {
				return "", deleteRepoLockDB(id, planId, nextAttempt)
			})

			if err != nil {
				log.Printf("[Delete][%d] Error retrying delete: %v",
					goroutineID, err)
				return fmt.Errorf("error retrying delete: %v", err)
			}
		}

		return fmt.Errorf("error getting lockable plan id: %v", err)
	}

	// get lock scope
	query := "SELECT scope, branch FROM repo_locks WHERE id = $1"
	var lockScope LockScope
	var branch *string
	err = tx.QueryRow(query, id).Scan(&lockScope, &branch)
	if err != nil {
		return fmt.Errorf("error getting lock scope: %v", err)
	}

	// var b string
	// if branch != nil {
	// 	b = *branch
	// }

	query = "DELETE FROM repo_locks WHERE id = $1"
	result, err := tx.Exec(query, id)
	if err != nil {
		if isDeadlockError(err) {
			log.Printf("[Delete][%d] Deadlock error, retrying delete %v",
				goroutineID, err)
			_, err := retryWithExponentialBackoff(context.Background(), err, numRetry, func(nextAttempt int) (string, error) {
				return "", deleteRepoLockDB(id, planId, nextAttempt)
			})

			if err != nil {
				log.Printf("[Delete][%d] Error retrying delete: %v",
					goroutineID, err)
				return fmt.Errorf("error retrying delete: %v", err)
			}
		}

		log.Printf("[Delete][%d] Error executing delete %v",
			goroutineID, err)
		return fmt.Errorf("error removing lock: %v", err)
	}

	// defer releaseMemLock(planId, b, lockScope)

	// Check if we actually deleted anything
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[Delete][%d] Error checking rows affected: %v", goroutineID, err)
	} else {
		log.Printf("[Delete][%d] Deleted %d rows", goroutineID, rowsAffected)
	}

	if err = tx.Commit(); err != nil {
		log.Printf("[Delete][%d] Error committing delete: %v", goroutineID, err)
		return fmt.Errorf("error committing delete: %v", err)
	}

	committed = true

	log.Printf("[Delete][%d] DELETE completed in %v",
		goroutineID, time.Since(deleteStart))

	return nil
}

func formatStackTrace(stack []byte) string {
	lines := strings.Split(string(stack), "\n")
	// Take first 10 meaningful lines of stack trace
	// Skip runtime frames (first 7 lines) and limit to next 10 lines
	relevantLines := lines[7:min(len(lines), 17)]
	return strings.Join(relevantLines, "\n")
}

func getGoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func isDeadlockError(err error) bool {
	if err == nil {
		return false
	}

	if pqErr, ok := err.(*pq.Error); ok && (pqErr.Code == "40001" || pqErr.Code == "40P01") {
		return true
	}

	return false
}

func retryWithExponentialBackoff(
	ctx context.Context,
	cause error,
	attempt int,
	nextCall func(int) (string, error),
) (string, error) {
	// If we have retried enough times, bail out.
	if attempt >= maxRetries {
		return "", fmt.Errorf("failed to acquire lock after %d attempts: %w", attempt, cause)
	}

	// Exponential delay: initialRetryDelay * 2^(attempt)
	backoff := time.Duration(float64(initialRetryDelay) * math.Pow(backoffFactor, float64(attempt)))
	// Add jitter: ± jitterFraction
	jitterRange := time.Duration(float64(backoff) * jitterFraction)
	jitter := time.Duration(rand.Int63n(int64(jitterRange)*2)) - jitterRange

	wait := backoff + jitter
	if wait < 0 {
		wait = 0
	}

	log.Printf("Lock/transaction conflict (attempt #%d). Retrying in %s... (cause: %v)", attempt, wait, cause)

	select {
	case <-ctx.Done():
		return "", fmt.Errorf("context canceled while waiting to retry: %w", ctx.Err())
	case <-time.After(wait):
		// Proceed with the next attempt.
	}

	return nextCall(attempt + 1)
}

func CleanupAllLocks(ctx context.Context) error {
	log.Println("Cleaning up all repo locks...")

	// Start a transaction with repeatable read isolation level
	tx, err := Conn.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}

	// Ensure rollback is attempted in case of failure
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("transaction rollback error: %v\n", rbErr)
			} else {
				log.Println("transaction rolled back")
			}
		}
	}()

	// Delete all locks
	query := "DELETE FROM repo_locks"
	_, err = tx.Exec(query)
	if err != nil {
		return fmt.Errorf("error removing all locks: %v", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	log.Println("Successfully cleaned up all repo locks")
	return nil
}
