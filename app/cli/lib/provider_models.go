package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	shared "plandex-shared"
)

// ProviderCatalogModel is one model entry from a provider's OpenAI-compatible
// GET /models endpoint. Only Id is guaranteed—the other fields are extensions
// served by meta-routers like OpenRouter and OrcaRouter, and are 0 when the
// provider doesn't include them.
type ProviderCatalogModel struct {
	Id                  string
	ContextLength       int
	MaxCompletionTokens int
	InputPerM           float64
	OutputPerM          float64
	HasPricing          bool
}

type providerCatalogResponse struct {
	Data []providerCatalogModelJSON `json:"data"`
}

type providerCatalogModelJSON struct {
	Id                  string `json:"id"`
	ContextLength       int    `json:"context_length"`
	MaxCompletionTokens int    `json:"max_completion_tokens"`
	TopProvider         *struct {
		ContextLength       int `json:"context_length"`
		MaxCompletionTokens int `json:"max_completion_tokens"`
	} `json:"top_provider"`
	Pricing *struct {
		Prompt               string `json:"prompt"`
		Completion           string `json:"completion"`
		PromptPerMillion     string `json:"prompt_per_million"`
		CompletionPerMillion string `json:"completion_per_million"`
	} `json:"pricing"`
}

// FetchProviderModels lists the models a provider is currently serving via its
// OpenAI-compatible GET /models endpoint. It sends the provider's API key as a
// bearer token when one is set, and also works unauthenticated for providers
// whose models endpoint is public.
func FetchProviderModels(cfg *shared.ModelProviderConfigSchema) ([]ProviderCatalogModel, error) {
	if cfg.BaseUrl == "" || cfg.BaseUrl == shared.LiteLLMBaseUrl {
		return nil, fmt.Errorf("provider '%s' doesn't expose a public models endpoint", cfg.ToComposite())
	}

	url := strings.TrimSuffix(cfg.BaseUrl, "/") + "/models"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	authVars, err := ResolveProviderAuthVars(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.ApiKeyEnvVar != "" && authVars[cfg.ApiKeyEnvVar] != "" {
		req.Header.Set("Authorization", "Bearer "+authVars[cfg.ApiKeyEnvVar])
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		if cfg.ApiKeyEnvVar != "" {
			return nil, fmt.Errorf("%s requires authentication—set %s and try again", url, cfg.ApiKeyEnvVar)
		}
		return nil, fmt.Errorf("%s requires authentication", url)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching %s: %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var parsed providerCatalogResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("error parsing response from %s: %v", url, err)
	}

	var models []ProviderCatalogModel
	for _, m := range parsed.Data {
		if m.Id == "" {
			continue
		}

		model := ProviderCatalogModel{
			Id:                  m.Id,
			ContextLength:       m.ContextLength,
			MaxCompletionTokens: m.MaxCompletionTokens,
		}

		if m.TopProvider != nil {
			if model.ContextLength == 0 {
				model.ContextLength = m.TopProvider.ContextLength
			}
			if model.MaxCompletionTokens == 0 {
				model.MaxCompletionTokens = m.TopProvider.MaxCompletionTokens
			}
		}

		if m.Pricing != nil {
			input, inputOk := pricePerMillion(m.Pricing.PromptPerMillion, m.Pricing.Prompt)
			output, outputOk := pricePerMillion(m.Pricing.CompletionPerMillion, m.Pricing.Completion)
			if inputOk || outputOk {
				model.InputPerM = input
				model.OutputPerM = output
				model.HasPricing = true
			}
		}

		models = append(models, model)
	}

	return models, nil
}

// pricePerMillion resolves a USD-per-million-tokens price from either a
// per-million string or a per-token string (the two formats meta-routers use).
func pricePerMillion(perMillion, perToken string) (float64, bool) {
	if perMillion != "" {
		if v, err := strconv.ParseFloat(perMillion, 64); err == nil && v >= 0 {
			return v, true
		}
	}
	if perToken != "" {
		if v, err := strconv.ParseFloat(perToken, 64); err == nil && v >= 0 {
			return v * 1_000_000, true
		}
	}
	return 0, false
}
