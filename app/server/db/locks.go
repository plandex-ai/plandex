package db

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

const lockHeartbeatInterval = 700 * time.Millisecond
const lockHeartbeatTimeout = 10 * time.Second
const maxRetries = 10
const initialRetryInterval = 100 * time.Millisecond

// distributed locking to ensure only one user can write to a plan repo at a time
// multiple readers are allowed, but read locks block writes
// write lock is exclusive (blocks both reads and writes)

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

func LockRepo(params LockRepoParams) (string, error) {
	return lockRepo(params, 0)
}

func lockRepo(params LockRepoParams, numRetry int) (string, error) {
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
		log.Printf("[Lock][%d] Error starting transaction after %v: %v",
			goroutineID, time.Since(start), err)
		return "", fmt.Errorf("error starting transaction: %v", err)
	}

	// Ensure that rollback is attempted in case of failure
	defer func() {
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

	query := "SELECT id, org_id, user_id, plan_id, plan_build_id, scope, branch, last_heartbeat_at, created_at FROM repo_locks WHERE plan_id = $1"
	if forUpdate {
		query += " FOR UPDATE"
	} else {
		query += " FOR SHARE"
	}
	queryArgs := []interface{}{planId}

	var locks []*repoLock

	fn := func() error {
		log.Println("obtaining repo lock with query")
		rows, err := tx.Query(query, queryArgs...)
		if err != nil {
			if pqErr, ok := err.(*pq.Error); ok && (pqErr.Code == "40001" || pqErr.Code == "40P01") {
				// return concurrency errors directly for retries
				return err
			}

			return fmt.Errorf("error getting repo locks: %v", err)
		}
		log.Println("repo lock query executed")

		if forUpdate {
			log.Printf("[Lock][%d] SELECT FOR UPDATE took %v",
				goroutineID, time.Since(selectStart))
		} else {
			log.Printf("[Lock][%d] SELECT FOR SHARE took %v",
				goroutineID, time.Since(selectStart))
		}

		defer rows.Close()

		var expiredLockIds []string

		now := time.Now()
		for rows.Next() {
			var lock repoLock
			if err := rows.Scan(&lock.Id, &lock.OrgId, &lock.UserId, &lock.PlanId, &lock.PlanBuildId, &lock.Scope, &lock.Branch, &lock.LastHeartbeatAt, &lock.CreatedAt); err != nil {
				return fmt.Errorf("error scanning repo lock: %v", err)
			}

			// ensure heartbeat hasn't timed out
			if now.Sub(lock.LastHeartbeatAt) < lockHeartbeatTimeout {
				locks = append(locks, &lock)
			} else {
				expiredLockIds = append(expiredLockIds, lock.Id)
			}
		}

		if len(expiredLockIds) > 0 {
			log.Printf("deleting expired locks: %v", expiredLockIds)

			query := "DELETE FROM repo_locks WHERE id = ANY($1)"
			_, err := tx.Exec(query, pq.Array(expiredLockIds))
			if err != nil {
				return fmt.Errorf("error removing expired locks: %v", err)
			}
		}

		return nil
	}
	err = fn()
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && (pqErr.Code == "40001" || pqErr.Code == "40P01") {
			if numRetry > maxRetries {
				err = fmt.Errorf("plan is currently being updated by another user")
				log.Println("max retries reached on serialization error, returning error")
				return "", err
			}

			// Calculate exponential backoff with small jitter
			backoff := initialRetryInterval * time.Duration(1<<numRetry)
			jitter := time.Duration(rand.Float64() * float64(backoff) * 0.1) // 10% jitter
			wait := backoff + jitter

			log.Printf("Serialization or deadlock error, retrying transaction after %v: %v\n", wait, err)

			select {
			case <-ctx.Done():
				return "", fmt.Errorf("context finished during retry transaction")
			case <-time.After(wait):
				log.Printf("Retrying transaction after %v\n", wait)
				return lockRepo(params, numRetry+1)
			}
		}

		return "", err
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

		// 10 second timeout
		if numRetry > 20 {
			err = fmt.Errorf("plan is currently being updated by another user or process")
			return "", err
		}
		time.Sleep(500 * time.Millisecond)
		return lockRepo(params, numRetry+1)
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

	insertQuery := "INSERT INTO repo_locks (org_id, user_id, plan_id, plan_build_id, scope, branch) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id"
	err = tx.QueryRow(
		insertQuery,
		newLock.OrgId,
		newLock.UserId,
		newLock.PlanId,
		newLock.PlanBuildId,
		newLock.Scope,
		newLock.Branch,
	).Scan(&newLock.Id)
	if err != nil {
		return "", fmt.Errorf("error inserting new lock: %v", err)
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
				err := DeleteRepoLock(newLock.Id)
				if err != nil {
					log.Printf("Error unlocking repo: %v\n", err)
				}
				return

			default:
				res, err := Conn.Exec("UPDATE repo_locks SET last_heartbeat_at = NOW() WHERE id = $1", newLock.Id)

				if err != nil {
					log.Printf("Error updating repo lock last heartbeat: %v\n", err)
					numErrors++

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

				time.Sleep(lockHeartbeatInterval)
			}

		}
	}()

	log.Println("repo locked. id:", newLock.Id)

	return newLock.Id, nil
}

func DeleteRepoLock(id string) error {
	start := time.Now()
	goroutineID := getGoroutineID()

	log.Printf("[Delete][%d] START delete lock %s at %v", goroutineID, id, start)
	stack := debug.Stack()
	// Format truncated stack excluding runtime frames
	stackTrace := formatStackTrace(stack)
	log.Printf("DELETE_ATTEMPT | id: %s | stack:\n%s", id, stackTrace)

	defer func() {
		elapsed := time.Since(start)
		log.Printf("[Delete][%d] END delete lock took %v", goroutineID, elapsed)
	}()

	// Start a new transaction for the delete
	tx, err := Conn.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		log.Printf("[Delete][%d] Error starting delete transaction: %v", goroutineID, err)
		return fmt.Errorf("error starting delete transaction: %v", err)
	}
	defer tx.Rollback()

	deleteStart := time.Now()
	log.Printf("[Delete][%d] Starting DELETE at %v", goroutineID, deleteStart)

	query := "DELETE FROM repo_locks WHERE id = $1"
	result, err := tx.Exec(query, id)
	if err != nil {
		log.Printf("[Delete][%d] Error executing delete after %v: %v",
			goroutineID, time.Since(deleteStart), err)
		return fmt.Errorf("error removing lock: %v", err)
	}

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
