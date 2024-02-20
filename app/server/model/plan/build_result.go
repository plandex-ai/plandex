package plan

import (
	"plandex-server/db"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
)

type planResultParams struct {
	orgId           string
	planId          string
	planBuildId     string
	convoMessageIds []string
	filePath        string
	currentState    string
	context         *db.Context
	replacements    []*shared.Replacement
}

func getPlanResult(params planResultParams) (*db.PlanFileResult, bool) {
	orgId := params.orgId
	planId := params.planId
	planBuildId := params.planBuildId
	filePath := params.filePath
	currentState := params.currentState
	contextPart := params.context
	replacements := params.replacements
	updated := params.currentState

	sort.Slice(replacements, func(i, j int) bool {
		iIdx := strings.Index(updated, replacements[i].Old)
		jIdx := strings.Index(updated, replacements[j].Old)
		return iIdx < jIdx
	})

	_, allSucceeded := shared.ApplyReplacements(currentState, replacements, true)

	var contextSha string
	var contextBody string
	if contextPart != nil {
		contextSha = contextPart.Sha
		contextBody = contextPart.Body
	}

	for _, replacement := range replacements {
		id := uuid.New().String()
		replacement.Id = id
	}

	return &db.PlanFileResult{
		OrgId:           orgId,
		PlanId:          planId,
		PlanBuildId:     planBuildId,
		ConvoMessageIds: params.convoMessageIds,
		Content:         "",
		Path:            filePath,
		Replacements:    replacements,
		ContextSha:      contextSha,
		ContextBody:     contextBody,
		AnyFailed:       !allSucceeded,
	}, allSucceeded
}
