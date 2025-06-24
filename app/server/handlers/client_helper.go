package handlers

import (
	"log"
	"net/http"
	"os"
	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/types"
	shared "plandex-shared"
)

type initClientsParams struct {
	w    http.ResponseWriter
	auth *types.ServerAuth

	apiKeys     map[string]string // deprecated
	openAIOrgId string            // deprecated

	authVars map[string]string

	plan     *db.Plan
	settings *shared.PlanSettings
}

type initClientsResult struct {
	clients  map[string]model.ClientInfo
	authVars map[string]string
}

func initClients(params initClientsParams) initClientsResult {
	w := params.w
	settings := params.settings

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
		return initClientsResult{}
	}

	if hookResult.GetIntegratedModelsResult != nil && hookResult.GetIntegratedModelsResult.IntegratedModelsMode {
		authVars = hookResult.GetIntegratedModelsResult.AuthVars
	}
	if len(authVars) == 0 && os.Getenv("IS_CLOUD") != "" {
		log.Println("No api keys/credentials provided for models")
		http.Error(w, "No api keys/credentials provided for models", http.StatusBadRequest)
		return initClientsResult{}
	}

	clients := model.InitClients(authVars, settings)

	return initClientsResult{
		clients:  clients,
		authVars: authVars,
	}
}
