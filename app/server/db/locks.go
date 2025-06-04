package db

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"plandex-server/notify"
	"plandex-server/shutdown"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

const locksVerboseLogging = false

const lockHeartbeatInterval = 3 * time.Second
const lockHeartbeatTimeout = 60 * time.Second
const maxLockRetries = 6
const initialLockRetryDelay = 300 * time.Millisecond
const backoffFactor = 2    // Exponential base
const jitterFraction = 0.3 // e.g. ±30% of the backoff

// We want deletes to win quickly vs. lock reads, so we retry them more aggressively with no backoff
const maxDeleteRetries = 60
const deleteRetryDelay = 50 * time.Millisecond

var activeLockIds = make(map[string]bool)
var activeLockIdsMu sync.Mutex

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
	Reason      string
}

func lockRepoDB(params LockRepoParams, numRetry int) (string, error) {
	start := time.Now()
	goroutineID := getGoroutineID()

	if locksVerboseLogging {
		log.Printf("[Lock][%d] START lock attempt for plan %s scope %s (retry %d) at %v | reason: %s",
			goroutineID, params.PlanId, params.Scope, numRetry, start, params.Reason)
	}

	defer func() {
		if locksVerboseLogging {
			elapsed := time.Since(start)
			log.Printf("[Lock][%d] END lock attempt took %v | reason: %s", goroutineID, elapsed, params.Reason)
		}
	}()

	// ensure context did not cancel
	if params.Ctx.Err() != nil {
		if locksVerboseLogging {
			log.Printf("[Lock][%d] Context canceled, returning error: %v | reason: %s", goroutineID, params.Ctx.Err(), params.Reason)
		}
		return "", params.Ctx.Err()
	}

	initialJitter := time.Duration(rand.Int63n(int64(5000 * time.Microsecond)))

	select {
	case <-params.Ctx.Done():
		return "", params.Ctx.Err()
	case <-time.After(initialJitter):
	}

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
		if locksVerboseLogging {
			log.Printf("[Lock][%d] Error starting transaction %v | reason: %s",
				goroutineID, err, params.Reason)
		}
		return "", fmt.Errorf("error starting transaction: %v", err)
	}

	var committed bool

	// Ensure that rollback is attempted in case of failure
	defer func() {
		if committed {
			return
		}

		panicErr := recover()
		if panicErr != nil {
			log.Printf("panic in lock repo: %v", panicErr)
		}

		if rbErr := tx.Rollback(); rbErr != nil {
			if rbErr == sql.ErrTxDone {
				if locksVerboseLogging {
					// log.Println("attempted to roll back transaction, but it was already committed")
				}
			} else {
				log.Printf("transaction rollback error: %v\n", rbErr)
			}
		} else {
			if locksVerboseLogging {
				log.Println("transaction rolled back")
			}
		}
	}()

	forUpdate := params.Scope == LockScopeWrite

	selectStart := time.Now()
	if locksVerboseLogging {
		if forUpdate {
			log.Printf("[Lock][%d] Starting SELECT FOR UPDATE at %v | reason: %s", goroutineID, selectStart, params.Reason)
		} else {
			log.Printf("[Lock][%d] Starting SELECT FOR SHARE at %v | reason: %s", goroutineID, selectStart, params.Reason)
		}
	}

	lockablePlanIdQuery := "SELECT * FROM lockable_plan_ids WHERE plan_id = $1"
	if forUpdate {
		lockablePlanIdQuery += " FOR UPDATE"
	} else {
		lockablePlanIdQuery += " FOR SHARE"
	}

	_, err = tx.Exec(lockablePlanIdQuery, planId)
	if err != nil {
		log.Printf("[Lock][%d] getting lockable plan id for %s: %v | reason: %s", goroutineID, planId, err, params.Reason)
		return retryWithExponentialBackoff(params.Ctx, err, numRetry, func(nextAttempt int) (string, error) {
			return lockRepoDB(params, nextAttempt)
		})
	}

	log.Printf("[Lock][%d] got lockable plan id for %s | reason: %s", goroutineID, planId, params.Reason)

	query := "SELECT id, org_id, user_id, plan_id, plan_build_id, scope, branch, last_heartbeat_at, created_at FROM repo_locks WHERE plan_id = $1"
	if forUpdate {
		query += " FOR UPDATE"
	} else {
		query += " FOR SHARE"
	}
	queryArgs := []interface{}{planId}

	var locks []*repoLock
	if locksVerboseLogging {
		log.Println("obtaining repo lock with query")
	}
	repoLockRows, err := tx.Query(query, queryArgs...)
	if err != nil {
		log.Printf("[Lock][%d] error obtaining repo lock with query: %v | reason: %s", goroutineID, err, params.Reason)
		return retryWithExponentialBackoff(params.Ctx, err, numRetry, func(nextAttempt int) (string, error) {
			return lockRepoDB(params, nextAttempt)
		})
	}
	if locksVerboseLogging {
		log.Println("repo lock query executed")
	}

	if locksVerboseLogging {
		if forUpdate {
			log.Printf("[Lock][%d] SELECT FOR UPDATE took %v | reason: %s",
				goroutineID, time.Since(selectStart), params.Reason)
		} else {
			log.Printf("[Lock][%d] SELECT FOR SHARE took %v | reason: %s",
				goroutineID, time.Since(selectStart), params.Reason)
		}
	}

	defer repoLockRows.Close()

	var expiredLockIds []string
	expiredLockIdsSet := make(map[string]bool)

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
			expiredLockIdsSet[lock.Id] = true
		}
	}

	if err := repoLockRows.Err(); err != nil {
		log.Printf("[Lock][%d] error iterating over repo locks: %v | reason: %s", goroutineID, err, params.Reason)
		return "", fmt.Errorf("error iterating over repo locks: %v", err)
	}

	log.Printf("[Lock][%d] %d locks found, %d expired | reason: %s", goroutineID, len(locks), len(expiredLockIds), params.Reason)

	if len(expiredLockIds) > 0 {
		log.Printf("[Lock][%d] %d expired locks found, deleting | reason: %s", goroutineID, len(expiredLockIds), params.Reason)
		if locksVerboseLogging {
			log.Printf("deleting expired locks: %v", expiredLockIds)
		}

		query := "DELETE FROM repo_locks WHERE id = ANY($1)"
		_, err := tx.Exec(query, pq.Array(expiredLockIds))
		if err != nil {
			if isDeadlockError(err) {
				log.Println("deadlock clearing expired locks, won't do anything")
			} else {
				log.Printf("[Lock][%d] error removing expired locks: %v | reason: %s", goroutineID, err, params.Reason)
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
		if locksVerboseLogging {
			log.Println("can't acquire lock.", "numRetry:", numRetry)
		}
		conflictErr := errors.New("lock conflict: cannot acquire read/write lock")
		log.Printf("[Lock][%d] can't acquire lock, retrying: %v | reason: %s | now: %s | locks:\n%s\n", goroutineID, conflictErr, params.Reason, now, spew.Sdump(locks))

		return retryWithExponentialBackoff(params.Ctx, conflictErr, numRetry, func(nextAttempt int) (string, error) {
			return lockRepoDB(params, nextAttempt)
		})
	}

	if locksVerboseLogging {
		log.Println("can acquire lock - inserting new lock")
	}

	insertStart := time.Now()
	if locksVerboseLogging {
		log.Printf("[Lock][%d] Starting INSERT at %v | reason: %s", goroutineID, insertStart, params.Reason)
	}

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

	var insertedId sql.NullString

	insertQuery := "INSERT INTO repo_locks (org_id, user_id, plan_id, plan_build_id, scope, branch) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (plan_id) WHERE scope = 'w' DO NOTHING RETURNING id"

	if locksVerboseLogging {
		log.Printf("Insert query: %s", insertQuery)
	}

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

		log.Printf("[Lock][%d] error inserting new lock: %v | reason: %s", goroutineID, err, params.Reason)
		return "", fmt.Errorf("error inserting new lock: %v", err)
	}

	if insertedId.Valid {
		newLock.Id = insertedId.String
	} else {
		if locksVerboseLogging {
			log.Printf("no rows returned from insert query, means there was a conflict")
		}
		return retryWithExponentialBackoff(params.Ctx, err, numRetry, func(nextAttempt int) (string, error) {
			return lockRepoDB(params, nextAttempt)
		})
	}

	if locksVerboseLogging {
		log.Printf("[Lock][%d] INSERT took %v | reason: %s",
			goroutineID, time.Since(insertStart), params.Reason)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("error committing transaction: %v", err)
	}

	committed = true

	activeLockIdsMu.Lock()
	activeLockIds[newLock.Id] = true
	activeLockIdsMu.Unlock()

	log.Printf("Lock acquired: %s for plan %s with scope %s | reason: %s", newLock.Id, planId, scope, params.Reason)

	// Start a goroutine to keep the lock alive
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in heartbeat goroutine: %v\n%s", r, debug.Stack())
				cancelFn()
				go notify.NotifyErr(notify.SeverityError, fmt.Errorf("panic in lock heartbeat goroutine: %v\n%s", r, debug.Stack()))
			}
		}()

		onCancel := func() {
			log.Printf("[Lock][Heartbeat] Timeout or context canceled during heartbeat loop for lock %s for plan %s | reason: %s", newLock.Id, planId, params.Reason)
		}

		numErrors := 0
		for {
			select {
			case <-ctx.Done():
				onCancel()
				return

			default:
				jitter := time.Duration(rand.Int63n(int64(float64(lockHeartbeatInterval)*0.1)) * int64(numErrors+1))

				log.Printf("[Lock][Heartbeat] Will update heartbeat for %s | %s | Heartbeat interval: %s, jitter: %s", planId, params.Reason, lockHeartbeatInterval, jitter)

				select {
				case <-ctx.Done():
					onCancel()
					return
				case <-time.After(lockHeartbeatInterval + jitter):
				}

				log.Printf("[Lock][Heartbeat] %s | %s | Updating repo lock last heartbeat\n", planId, params.Reason)

				res, err := Conn.Exec("UPDATE repo_locks SET last_heartbeat_at = NOW() WHERE id = $1", newLock.Id)

				if err != nil {
					log.Printf("[Lock][Heartbeat] %s | %s | Error updating repo lock last heartbeat: %v\n", planId, params.Reason, err)

					if isDeadlockError(err) {
						log.Printf("[Lock][Heartbeat] %s | %s | Heartbeat deadlock error, keep retrying\n", planId, params.Reason)
					}

					numErrors++

					if numErrors > 5 {
						log.Printf("[Lock][Heartbeat] %s | %s | Too many errors updating repo lock last heartbeat: %v\n", planId, params.Reason, err)
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
						log.Printf("[Lock][Heartbeat] %s | %s | Lock not found: %s | stopping heartbeat loop\n", planId, params.Reason, newLock.Id)
						return
					}

					log.Printf("[Lock][Heartbeat] %s | %s | Lock found: %s | continuing heartbeat loop\n", planId, params.Reason, newLock.Id)
				}
			}

		}
	}()

	// check if git lock file exists
	// remove it if so
	err = gitRemoveIndexLockFileIfExists(getPlanDir(orgId, planId))
	if err != nil {
		log.Printf("[Lock] %s | %s | Error removing lock file: %v", planId, params.Reason, err)
		return newLock.Id, fmt.Errorf("error removing lock file: %v", err)
	}

	if branch != "" {
		// checkout the branch
		err = gitCheckoutBranch(getPlanDir(orgId, planId), branch)
		if err != nil {
			log.Printf("[Lock] %s | %s | Error checking out branch: %v", planId, params.Reason, err)
			return newLock.Id, fmt.Errorf("error checking out branch: %v", err)
		}
		log.Printf("[Lock] %s | %s | Checked out branch", planId, params.Reason)
	}

	return newLock.Id, nil
}

func deleteRepoLockDB(id, planId, reason string, numRetry int) error {
	start := time.Now()
	goroutineID := getGoroutineID()

	if locksVerboseLogging {
		log.Printf("[Lock][Delete][%d] START delete lock %s at %v | reason: %s", goroutineID, id, start, reason)

		defer func() {
			log.Printf("[Lock][Delete][%d] END delete lock took %v | reason: %s", goroutineID, time.Since(start), reason)
		}()
	}

	result, err := Conn.Exec("DELETE FROM repo_locks WHERE id = $1", id)
	if err != nil {
		log.Printf("[Lock][Delete][%d] Error deleting lock: %v | reason: %s", goroutineID, err, reason)

		err := retryDeleteLock(shutdown.ShutdownCtx, err, numRetry, func(nextAttempt int) error {
			return deleteRepoLockDB(id, planId, reason, nextAttempt)
		})

		if err != nil {
			log.Printf("[Lock][Delete][%d] Error deleting lock after retries: %v | %s | %s | %s", goroutineID, err, id, planId, reason)
			return err
		}

		// retries succeeded, stop
		return nil
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("[Lock][Delete][%d] Lock released: %s for plan %s | %s", goroutineID, id, planId, reason)
	} else {
		log.Printf("[Lock][Delete][%d] Lock not found: %s | %s | %s", goroutineID, id, planId, reason)
	}

	activeLockIdsMu.Lock()
	delete(activeLockIds, id)
	activeLockIdsMu.Unlock()

	return nil
}

func formatStackTrace(stack []byte) string {
	numLines := 10
	if !locksVerboseLogging {
		numLines = 5
	}
	return formatStackTraceWithNumLines(stack, numLines)
}

func formatStackTraceLong(stack []byte) string {
	return formatStackTraceWithNumLines(stack, 20)
}

func formatStackTraceWithNumLines(stack []byte, numLines int) string {
	lines := strings.Split(string(stack), "\n")
	// Take first 10 meaningful lines of stack trace
	// Skip runtime frames (first 7 lines) and limit to next 10 lines
	relevantLines := lines[7:min(len(lines), 7+numLines)]
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
	if attempt >= maxLockRetries {
		log.Printf("[Lock][Retry][%d] Failed to acquire lock after %d attempts: %v", getGoroutineID(), attempt, cause)
		return "", fmt.Errorf("failed to acquire lock after %d attempts: %w", attempt, cause)
	}

	// Exponential delay: initialRetryDelay * 2^(attempt)
	backoff := time.Duration(float64(initialLockRetryDelay) * math.Pow(backoffFactor, float64(attempt)))
	// Add jitter: ± jitterFraction
	jitterRange := time.Duration(float64(backoff) * jitterFraction)
	jitter := time.Duration(rand.Int63n(int64(jitterRange)*2)) - jitterRange

	wait := backoff + jitter
	if wait < 0 {
		wait = 0
	}

	log.Printf("[Lock][Retry][%d] Lock/transaction conflict (attempt #%d). Retrying in %s... (cause: %v)", getGoroutineID(), attempt, wait, cause)

	select {
	case <-ctx.Done():
		log.Printf("[Lock][Retry][%d] Context canceled while waiting to retry: %v", getGoroutineID(), ctx.Err())
		return "", fmt.Errorf("context canceled while waiting to retry: %w", ctx.Err())
	case <-time.After(wait):
		// Proceed with the next attempt.
	}

	return nextCall(attempt + 1)
}

func retryDeleteLock(ctx context.Context, cause error, attempt int, nextCall func(int) error) error {
	if attempt >= maxDeleteRetries {
		return fmt.Errorf("delete lock failed after 10 attempts: %w", cause)
	}
	// retry 10 times, no backoff or maybe a tiny 50ms
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(deleteRetryDelay):
	}
	return nextCall(attempt + 1)
}

func CleanupActiveLocks(ctx context.Context) error {
	log.Println("Cleaning up any active repo locks...")

	// Start a transaction with repeatable read isolation level
	tx, err := Conn.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}

	// Ensure rollback is attempted in case of failure
	defer func() {
		panicErr := recover()
		if panicErr != nil {
			log.Printf("panic in cleanup all locks: %v", panicErr)
		}

		if rbErr := tx.Rollback(); rbErr != nil {
			if rbErr == sql.ErrTxDone {
				// log.Println("attempted to roll back transaction, but it was already committed")
			} else {
				log.Printf("transaction rollback error: %v\n", rbErr)
			}
		} else {
			if locksVerboseLogging {
				log.Println("transaction rolled back")
			}
		}
	}()

	// Delete all active locks
	query := "DELETE FROM repo_locks WHERE id = ANY($1)"
	ids := make([]string, 0, len(activeLockIds))
	for id := range activeLockIds {
		ids = append(ids, id)
	}
	_, err = tx.Exec(query, pq.Array(ids))
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("No active locks to cleanup")
		} else {
			return fmt.Errorf("error removing all locks: %v", err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	activeLockIdsMu.Lock()
	activeLockIds = make(map[string]bool)
	activeLockIdsMu.Unlock()

	log.Println("Successfully cleaned up all repo locks")
	return nil
}
