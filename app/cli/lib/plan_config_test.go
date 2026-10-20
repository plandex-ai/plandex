package lib

import (
	"testing"

	shared "plandex-shared"
)

func TestGetCachedPlanConfigRequiresMatchingPlan(t *testing.T) {
	previousConfig := cachedPlanConfig
	previousPlanId := cachedPlanConfigPlanId
	previousCurrentPlanId := CurrentPlanId
	t.Cleanup(func() {
		cachedPlanConfig = previousConfig
		cachedPlanConfigPlanId = previousPlanId
		CurrentPlanId = previousCurrentPlanId
	})

	config := &shared.PlanConfig{}
	CurrentPlanId = "plan-a"
	SetCachedPlanConfig(config)

	if got := getCachedPlanConfig("plan-b"); got != nil {
		t.Fatal("expected a cache miss for a different plan")
	}
	if got := getCachedPlanConfig("plan-a"); got != config {
		t.Fatal("expected a cache hit for the plan that populated the cache")
	}
}
