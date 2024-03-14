package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
)

func loadContexts(w http.ResponseWriter, r *http.Request, auth *types.ServerAuth, loadReq *shared.LoadContextRequest, plan *db.Plan, branchName string) (*shared.LoadContextResponse, []*db.Context) {
	var err error

	ctx, cancel := context.WithCancel(context.Background())
	unlockFn := lockRepo(w, r, auth, db.LockScopeWrite, ctx, cancel, true)
	if unlockFn == nil {
		return nil, nil
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	res, dbContexts, err := db.LoadContexts(db.LoadContextsParams{
		OrgId:      auth.OrgId,
		Plan:       plan,
		BranchName: branchName,
		Req:        loadReq,
		UserId:     auth.User.Id,
	})

	if err != nil {
		log.Printf("Error loading contexts: %v\n", err)
		http.Error(w, "Error loading contexts: "+err.Error(), http.StatusInternalServerError)
		return nil, nil
	}

	if res.MaxTokensExceeded {
		log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d)", res.TotalTokens, res.MaxTokens)
		bytes, err := json.Marshal(res)

		if err != nil {
			log.Printf("Error marshalling response: %v\n", err)
			http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
			return nil, nil
		}

		w.Write(bytes)
		return nil, nil
	}

	err = db.GitAddAndCommit(auth.OrgId, plan.Id, branchName, res.Msg)

	if err != nil {
		log.Printf("Error committing changes: %v\n", err)
		http.Error(w, "Error committing changes: "+err.Error(), http.StatusInternalServerError)
		return nil, nil
	}

	return res, dbContexts
}
