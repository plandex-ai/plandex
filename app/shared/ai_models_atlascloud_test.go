package shared

import "testing"

func TestAtlasCloudProviderConfig(t *testing.T) {
	config, ok := BuiltInModelProviderConfigs[ModelProviderAtlasCloud]
	if !ok {
		t.Fatal("expected Atlas Cloud provider config")
	}

	if config.BaseUrl != AtlasCloudBaseUrl {
		t.Fatalf("expected Atlas Cloud base URL %q, got %q", AtlasCloudBaseUrl, config.BaseUrl)
	}

	if config.ApiKeyEnvVar != AtlasCloudApiKeyEnvVar {
		t.Fatalf("expected Atlas Cloud API key env var %q, got %q", AtlasCloudApiKeyEnvVar, config.ApiKeyEnvVar)
	}
}

func TestAtlasCloudBuiltInModels(t *testing.T) {
	deepSeek := GetAvailableModel(ModelProviderAtlasCloud, "atlascloud/deepseek-v4-pro")
	if deepSeek == nil {
		t.Fatal("expected Atlas Cloud DeepSeek V4 Pro model")
	}
	if deepSeek.ModelName != "deepseek-ai/deepseek-v4-pro" {
		t.Fatalf("expected DeepSeek V4 Pro model name, got %q", deepSeek.ModelName)
	}

	qwen := GetAvailableModel(ModelProviderAtlasCloud, "atlascloud/qwen3.5-flash")
	if qwen == nil {
		t.Fatal("expected Atlas Cloud Qwen3.5 Flash model")
	}
	if qwen.ModelName != "qwen/qwen3.5-flash" {
		t.Fatalf("expected Qwen3.5 Flash model name, got %q", qwen.ModelName)
	}
}
