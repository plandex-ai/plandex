package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/types"
	"runtime/debug"

	"github.com/gorilla/mux"
)

const locksVerboseLogging = true

func LockRepo(w http.ResponseWriter, r *http.Request, auth *types.ServerAuth, scope db.LockScope, ctx context.Context, cancelFn context.CancelFunc, requireBranch bool) *func(err error) {
	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]

	if locksVerboseLogging {
		log.Printf("LockRepo: %s, planId: %s, branch: %s", scope, planId, branch)
	}

	if requireBranch && branch == "" {
		if locksVerboseLogging {
			log.Println("Branch not specified")
		}
		http.Error(w, "Branch not specified", http.StatusBadRequest)
		return nil
	}

	return lockRepo(w, r, auth, scope, ctx, cancelFn, planId, branch)
}

func LockRepoForBranch(w http.ResponseWriter, r *http.Request, auth *types.ServerAuth, scope db.LockScope, ctx context.Context, cancelFn context.CancelFunc, branch string) *func(err error) {
	vars := mux.Vars(r)
	planId := vars["planId"]

	if locksVerboseLogging {
		log.Printf("LockRepoForBranch: %s, planId: %s, branch: %s", scope, planId, branch)
	}

	return lockRepo(w, r, auth, scope, ctx, cancelFn, planId, branch)
}

func lockRepo(w http.ResponseWriter, r *http.Request, auth *types.ServerAuth, scope db.LockScope, ctx context.Context, cancelFn context.CancelFunc, planId, branch string) *func(err error) {
	params := db.LockRepoParams{
		PlanId:   planId,
		Branch:   branch,
		Scope:    scope,
		Ctx:      ctx,
		CancelFn: cancelFn,
	}
	if auth == nil {
		// error out
		log.Println("Auth required")
		http.Error(w, "Auth required", http.StatusInternalServerError)
		return nil
	} else {
		params.OrgId = auth.OrgId
		if auth.User != nil {
			params.UserId = auth.User.Id
		}
	}

	if locksVerboseLogging {
		log.Printf("lockRepo: %s, planId: %s, branch: %s", params.Scope, params.PlanId, params.Branch)
	}

	repoLockId, err := db.LockRepo(params)

	if err != nil {
		log.Printf("Error locking repo: %v\n", err)
		http.Error(w, "Error locking repo: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	fn := func(err error) {
		log.Println("Unlocking repo in deferred unlock function")
		log.Printf("err: %v\n", err)

		if r := recover(); r != nil {
			stackTrace := debug.Stack()
			log.Printf("Recovered from panic: %v\n", r)
			log.Printf("Stack trace: %s\n", stackTrace)
			err = fmt.Errorf("server panic: %v", r)
			http.Error(w, "Error locking repo: "+err.Error(), http.StatusInternalServerError)
		}

		// log.Println("Rolling back repo if error")
		err = RollbackRepoIfErr(auth.OrgId, planId, branch, err)
		if err != nil {
			log.Printf("Error rolling back repo: %v\n", err)
		}

		err = db.DeleteRepoLock(repoLockId, planId)
		if err != nil {
			log.Printf("Error unlocking repo: %v\n", err)
		}
	}

	return &fn
}

func RollbackRepoIfErr(orgId, planId, branch string, err error) error {
	// if no error, return nil
	if err == nil {
		log.Println("No error, not rolling back repo")
		return nil
	}

	log.Println("Rolling back repo due to error")

	// if any errors, rollback repo
	err = db.GitClearUncommittedChanges(orgId, planId, branch)

	if err != nil {
		return fmt.Errorf("error clearing uncommitted changes: %v", err)
	}

	return nil
}
