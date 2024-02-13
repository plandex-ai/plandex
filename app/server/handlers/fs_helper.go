package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/types"

	"github.com/gorilla/mux"
)

func lockRepo(w http.ResponseWriter, r *http.Request, auth *types.ServerAuth, scope db.LockScope, ctx context.Context, cancelFn context.CancelFunc) *func(err error) {
	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]

	repoLockId, err := db.LockRepo(
		db.LockRepoParams{
			OrgId:    auth.OrgId,
			UserId:   auth.User.Id,
			PlanId:   planId,
			Branch:   branch,
			Scope:    scope,
			Ctx:      ctx,
			CancelFn: cancelFn,
		},
	)

	if err != nil {
		log.Printf("Error locking repo: %v\n", err)
		http.Error(w, "Error locking repo: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	fn := func(err error) {

		err = RollbackRepoIfErr(auth.OrgId, planId, err)
		if err != nil {
			log.Printf("Error rolling back repo: %v\n", err)
		}

		err = db.UnlockRepo(repoLockId)
		if err != nil {
			log.Printf("Error unlocking repo: %v\n", err)
		}
	}

	return &fn
}

func RollbackRepoIfErr(orgId, planId string, err error) error {
	// if no error, return nil
	if err == nil {
		return nil
	}

	// if any errors, rollback repo
	err = db.GitClearUncommittedChanges(orgId, planId)

	if err != nil {
		return fmt.Errorf("error clearing uncommitted changes: %v", err)
	}

	return nil
}
