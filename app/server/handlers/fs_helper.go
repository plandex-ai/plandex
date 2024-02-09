package handlers

import (
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/types"

	"github.com/gorilla/mux"
)

func lockRepo(w http.ResponseWriter, r *http.Request, auth *types.ServerAuth, scope db.LockScope) *func() {
	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]

	repoLockId, err := db.LockRepo(
		db.LockRepoParams{
			OrgId:  auth.OrgId,
			UserId: auth.User.Id,
			PlanId: planId,
			Branch: branch,
			Scope:  scope,
		},
	)

	if err != nil {
		log.Printf("Error locking repo: %v\n", err)
		http.Error(w, "Error locking repo: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	fn := func() {
		err := db.UnlockRepo(repoLockId)
		if err != nil {
			log.Printf("Error unlocking repo: %v\n", err)
		}
	}

	return &fn
}
