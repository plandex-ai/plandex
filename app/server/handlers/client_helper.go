package handlers

import (
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/types"

	"github.com/sashabaranov/go-openai"
)

type initClientsParams struct {
	w           http.ResponseWriter
	auth        *types.ServerAuth
	apiKey      string
	apiKeys     map[string]string
	endpoint    string
	openAIBase  string
	openAIOrgId string
	plan        *db.Plan
}

func initClients(params initClientsParams) map[string]*openai.Client {
	w := params.w
	apiKey := params.apiKey
	apiKeys := params.apiKeys
	plan := params.plan
	var openAIOrgId string
	var endpoint string

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
		apiKeys = hookResult.GetIntegratedModelsResult.ApiKeys
	} else {
		if apiKeys == nil {
			apiKeys = map[string]string{"OPENAI_API_KEY": apiKey}
		}

		openAIOrgId = params.openAIOrgId
		endpoint = params.openAIBase
		if endpoint == "" {
			endpoint = params.endpoint
		}
	}

	planSettings, err := db.GetPlanSettings(plan, true)
	if err != nil {
		log.Printf("Error getting plan settings: %v\n", err)
		http.Error(w, "Error getting plan settings", http.StatusInternalServerError)
		return nil
	}

	endpointsByApiKeyEnvVar := map[string]string{}
	for envVar := range apiKeys {
		if planSettings.ModelPack.Planner.BaseModelConfig.ApiKeyEnvVar == envVar {
			endpointsByApiKeyEnvVar[envVar] = planSettings.ModelPack.Planner.BaseModelConfig.BaseUrl
			continue
		}

		if planSettings.ModelPack.PlanSummary.BaseModelConfig.ApiKeyEnvVar == envVar {
			endpointsByApiKeyEnvVar[envVar] = planSettings.ModelPack.PlanSummary.BaseModelConfig.BaseUrl
			continue
		}

		if planSettings.ModelPack.Builder.BaseModelConfig.ApiKeyEnvVar == envVar {
			endpointsByApiKeyEnvVar[envVar] = planSettings.ModelPack.Builder.BaseModelConfig.BaseUrl
			continue
		}

		if planSettings.ModelPack.Namer.BaseModelConfig.ApiKeyEnvVar == envVar {
			endpointsByApiKeyEnvVar[envVar] = planSettings.ModelPack.Namer.BaseModelConfig.BaseUrl
			continue
		}

		if planSettings.ModelPack.CommitMsg.BaseModelConfig.ApiKeyEnvVar == envVar {
			endpointsByApiKeyEnvVar[envVar] = planSettings.ModelPack.CommitMsg.BaseModelConfig.BaseUrl
			continue
		}

		if planSettings.ModelPack.ExecStatus.BaseModelConfig.ApiKeyEnvVar == envVar {
			endpointsByApiKeyEnvVar[envVar] = planSettings.ModelPack.ExecStatus.BaseModelConfig.BaseUrl
			continue
		}

		if planSettings.ModelPack.GetWholeFileBuilder().BaseModelConfig.ApiKeyEnvVar == envVar {
			endpointsByApiKeyEnvVar[envVar] = planSettings.ModelPack.GetWholeFileBuilder().BaseModelConfig.BaseUrl
			continue
		}
	}

	if len(apiKeys) == 0 {
		log.Println("API key is required")
		http.Error(w, "API key is required", http.StatusBadRequest)
		return nil
	}

	clients := model.InitClients(apiKeys, endpointsByApiKeyEnvVar, endpoint, openAIOrgId)

	return clients
}
