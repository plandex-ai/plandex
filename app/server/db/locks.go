package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
)

const lockHeartbeatInterval = 700 * time.Millisecond
const lockHeartbeatTimeout = 4 * time.Second

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
	log.Println("locking repo")
	// spew.Dump(params)

	orgId := params.OrgId
	userId := params.UserId
	planId := params.PlanId
	branch := params.Branch
	scope := params.Scope
	planBuildId := params.PlanBuildId
	ctx := params.Ctx
	cancelFn := params.CancelFn

	tx, err := Conn.Begin()
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
		rows, err := tx.Query(query, queryArgs...)
		if err != nil {
			return fmt.Errorf("error getting repo locks: %v", err)
		}

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
	if err := fn(); err != nil {
		return "", err
	}

	canAcquire := true
	canRetry := true

	// log.Println("locks:")
	// spew.Dump(locks)

	for _, lock := range locks {
		lockBranch := ""
		if lock.Branch != nil {
			lockBranch = *lock.Branch
		}

		if scope == LockScopeRead {
			canAcquireThisLock := lock.Scope == LockScopeRead && lockBranch == branch
			if !canAcquireThisLock {
				canAcquire = false
			}
		} else if scope == LockScopeWrite {
			canAcquire = false

			// if lock is for the same plan plan and branch, allow parallel writes
			if planId == lock.PlanId && branch == lockBranch {
				canAcquire = true
			}

			if lock.Scope == LockScopeWrite && lockBranch == branch {
				canRetry = false
			}
		} else {
			err = fmt.Errorf("invalid lock scope: %v", scope)
			return "", err
		}
	}

	if !canAcquire {
		log.Println("can't acquire lock. canRetry:", canRetry, "numRetry:", numRetry)

		if canRetry {
			// 10 second timeout
			if numRetry > 20 {
				err = fmt.Errorf("plan is currently being updated by another user")
				return "", err
			}
			time.Sleep(500 * time.Millisecond)
			return lockRepo(params, numRetry+1)
		}
		err = fmt.Errorf("plan is currently being updated by another user")
		return "", err
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
		OrgId:       orgId,
		UserId:      userId,
		PlanId:      planId,
		PlanBuildId: lockPlanBuildId,
		Scope:       scope,
		Branch:      lockBranch,
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

	branches, err := GitListBranches(orgId, planId)
	if err != nil {
		return "", fmt.Errorf("error getting branches: %v", err)
	}

	log.Println("branches:", branches)

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
				err := UnlockRepo(newLock.Id)
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

func UnlockRepo(id string) error {
	log.Println("unlocking repo:", id)

	query := "DELETE FROM repo_locks WHERE id = $1"
	_, err := Conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("error removing lock: %v", err)
	}

	log.Println("repo unlocked successfully:", id)

	return nil
}
