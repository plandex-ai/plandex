package handlers

import (
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/types"
)

type initClientsParams struct {
	w    http.ResponseWriter
	auth *types.ServerAuth

	apiKeys     map[string]string // deprecated
	openAIOrgId string            // deprecated

	authVars map[string]string

	plan *db.Plan
}

func initClients(params initClientsParams) map[string]model.ClientInfo {
	w := params.w

	var authVars map[string]string
	if params.authVars != nil {
		authVars = params.authVars
	} else if params.apiKeys != nil {
		authVars = map[string]string{}
		for envVar, apiKey := range params.apiKeys {
			authVars[envVar] = apiKey
		}
		if params.openAIOrgId != "" {
			authVars["OPENAI_ORG_ID"] = params.openAIOrgId
		}
	}

	hookResult, apiErr := hooks.ExecHook(hooks.GetIntegratedModels, hooks.HookParams{
		Auth: params.auth,
		Plan: params.plan,
	})

	if apiErr != nil {
		log.Printf("Error getting integrated models: %v\n", apiErr)
		http.Error(w, "Error getting integrated models", http.StatusInternalServerError)
		return nil
	}

	if hookResult.GetIntegratedModelsResult != nil && hookResult.GetIntegratedModelsResult.IntegratedModelsMode {
		authVars = hookResult.GetIntegratedModelsResult.AuthVars
	}
	if len(authVars) == 0 {
		log.Println("No api keys/credentials provided for models")
		http.Error(w, "No api keys/credentials provided for models", http.StatusBadRequest)
		return nil
	}

	clients := model.InitClients(authVars)

	return clients
}
