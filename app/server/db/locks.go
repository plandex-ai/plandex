package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
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
	log.Printf("locking repo. orgId: %s | planId: %s | scope: %s, \n", params.OrgId, params.PlanId, params.Scope)
	// spew.Dump(params)

	orgId := params.OrgId
	userId := params.UserId
	planId := params.PlanId
	branch := params.Branch
	scope := params.Scope
	planBuildId := params.PlanBuildId
	ctx := params.Ctx
	cancelFn := params.CancelFn

	tx, err := Conn.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return "", fmt.Errorf("error starting transaction: %v", err)
	}

	// Ensure that rollback is attempted in case of failure
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("transaction rollback error: %v\n", rbErr)
			} else {
				log.Println("transaction rolled back")
			}
		}
	}()

	query := "SELECT id, org_id, user_id, plan_id, plan_build_id, scope, branch, created_at FROM repo_locks WHERE plan_id = $1 FOR UPDATE"
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

		defer rows.Close()

		var expiredLockIds []string

		now := time.Now()
		for rows.Next() {
			var lock repoLock
			if err := rows.Scan(&lock.Id, &lock.OrgId, &lock.UserId, &lock.PlanId, &lock.PlanBuildId, &lock.Scope, &lock.Branch, &lock.CreatedAt); err != nil {
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

			log.Printf("Serialization or deadlock error, retrying transaction: %v\n", err)

			wait := initialRetryInterval * time.Duration(1<<numRetry) * time.Duration(rand.Intn(500)*int(time.Millisecond))

			select {
			case <-ctx.Done():
				return "", fmt.Errorf("context finished during retry transaction")
			case <-time.After(wait):
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
		PlanBuildId: lockPlanBuildId,
		Scope:       scope,
		Branch:      lockBranch,
	}

	if orgId != "" {
		newLock.OrgId = &orgId
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
				}

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

				time.Sleep(lockHeartbeatInterval)
			}

		}
	}()

	log.Println("repo locked. id:", newLock.Id)

	return newLock.Id, nil
}

func DeleteRepoLock(id string) error {
	log.Println("deleting repo lock:", id)

	query := "DELETE FROM repo_locks WHERE id = $1"
	_, err := Conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("error removing lock: %v", err)
	}

	log.Println("repo lock deleted successfully:", id)

	return nil
}
