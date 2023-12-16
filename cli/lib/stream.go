package lib

import (
	"encoding/json"
	"fmt"

	"github.com/plandex/plandex/shared"
)

func updateTokenCounts(content string, numStreamedTokensByPath map[string]int, finishedByPath map[string]bool) error {
	var planTokenCount shared.PlanTokenCount
	err := json.Unmarshal([]byte(content), &planTokenCount)
	if err != nil {
		return fmt.Errorf("error parsing plan token count update: %v", err)
	}
	numStreamedTokensByPath[planTokenCount.Path] += planTokenCount.NumTokens

	if planTokenCount.Finished {
		finishedByPath[planTokenCount.Path] = true
	}

	return nil
}
