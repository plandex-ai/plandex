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
	w           http.ResponseWriter
	auth        *types.ServerAuth
	apiKey      string
	apiKeys     map[string]string
	endpoint    string
	openAIBase  string
	openAIOrgId string
	plan        *db.Plan
}

func initClients(params initClientsParams) map[string]model.ClientInfo {
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
		planBaseModelConfig := planSettings.ModelPack.Planner.BaseModelConfigForEnvVar(envVar)
		if planBaseModelConfig != nil {
			endpointsByApiKeyEnvVar[envVar] = planBaseModelConfig.BaseUrl
			continue
		}

		coderModelConfig := planSettings.ModelPack.GetCoder().BaseModelConfigForEnvVar(envVar)
		if coderModelConfig != nil {
			endpointsByApiKeyEnvVar[envVar] = coderModelConfig.BaseUrl
			continue
		}

		planSummaryModelConfig := planSettings.ModelPack.PlanSummary.BaseModelConfigForEnvVar(envVar)
		if planSummaryModelConfig != nil {
			endpointsByApiKeyEnvVar[envVar] = planSummaryModelConfig.BaseUrl
			continue
		}

		builderModelConfig := planSettings.ModelPack.Builder.BaseModelConfigForEnvVar(envVar)
		if builderModelConfig != nil {
			endpointsByApiKeyEnvVar[envVar] = builderModelConfig.BaseUrl
			continue
		}

		namerModelConfig := planSettings.ModelPack.Namer.BaseModelConfigForEnvVar(envVar)
		if namerModelConfig != nil {
			endpointsByApiKeyEnvVar[envVar] = namerModelConfig.BaseUrl
			continue
		}

		commitMsgModelConfig := planSettings.ModelPack.CommitMsg.BaseModelConfigForEnvVar(envVar)
		if commitMsgModelConfig != nil {
			endpointsByApiKeyEnvVar[envVar] = commitMsgModelConfig.BaseUrl
			continue
		}

		execStatusModelConfig := planSettings.ModelPack.ExecStatus.BaseModelConfigForEnvVar(envVar)
		if execStatusModelConfig != nil {
			endpointsByApiKeyEnvVar[envVar] = execStatusModelConfig.BaseUrl
			continue
		}

		wholeFileBuilderModelConfig := planSettings.ModelPack.GetWholeFileBuilder().BaseModelConfigForEnvVar(envVar)
		if wholeFileBuilderModelConfig != nil {
			endpointsByApiKeyEnvVar[envVar] = wholeFileBuilderModelConfig.BaseUrl
			continue
		}

		architectModelConfig := planSettings.ModelPack.GetArchitect().BaseModelConfigForEnvVar(envVar)
		if architectModelConfig != nil {
			endpointsByApiKeyEnvVar[envVar] = architectModelConfig.BaseUrl
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
