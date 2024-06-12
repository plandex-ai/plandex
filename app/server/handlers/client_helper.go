package handlers

import (
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/model"

	"github.com/sashabaranov/go-openai"
)

type initClientsParams struct {
	w           http.ResponseWriter
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
	openAIOrgId := params.openAIOrgId
	plan := params.plan

	endpoint := params.openAIBase
	if endpoint == "" {
		endpoint = params.endpoint
	}
	if apiKeys == nil {
		apiKeys = map[string]string{"OPENAI_API_KEY": apiKey}
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

		if planSettings.ModelPack.GetVerifier().BaseModelConfig.ApiKeyEnvVar == envVar {
			endpointsByApiKeyEnvVar[envVar] = planSettings.ModelPack.GetVerifier().BaseModelConfig.BaseUrl
			continue
		}

		if planSettings.ModelPack.GetAutoFix().BaseModelConfig.ApiKeyEnvVar == envVar {
			endpointsByApiKeyEnvVar[envVar] = planSettings.ModelPack.GetAutoFix().BaseModelConfig.BaseUrl
			continue
		}
	}

	clients := model.InitClients(apiKeys, endpointsByApiKeyEnvVar, endpoint, openAIOrgId)

	return clients
}
