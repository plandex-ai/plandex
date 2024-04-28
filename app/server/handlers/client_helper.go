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

		if planSettings.ModelPack.Planner.BaseModelConfig.ApiKeyEnvVar == envVar {
			endpointsByApiKeyEnvVar[envVar] = planSettings.ModelPack.Planner.BaseModelConfig.BaseUrl
			continue
		}

		if planSettings.ModelPack.Planner.BaseModelConfig.ApiKeyEnvVar == envVar {
			endpointsByApiKeyEnvVar[envVar] = planSettings.ModelPack.Planner.BaseModelConfig.BaseUrl
			continue
		}

		if planSettings.ModelPack.Planner.BaseModelConfig.ApiKeyEnvVar == envVar {
			endpointsByApiKeyEnvVar[envVar] = planSettings.ModelPack.Planner.BaseModelConfig.BaseUrl
			continue
		}

		if planSettings.ModelPack.Planner.BaseModelConfig.ApiKeyEnvVar == envVar {
			endpointsByApiKeyEnvVar[envVar] = planSettings.ModelPack.Planner.BaseModelConfig.BaseUrl
			continue
		}
	}

	clients := model.InitClients(apiKeys, endpointsByApiKeyEnvVar, endpoint, openAIOrgId)

	return clients
}
