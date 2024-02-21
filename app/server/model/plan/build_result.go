package plan

import (
	"plandex-server/db"
	"plandex-server/types"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
)

type planResultParams struct {
	orgId                string
	planId               string
	planBuildId          string
	convoMessageId       string
	filePath             string
	currentState         string
	context              *db.Context
	fileContent          string
	streamedReplacements []*types.StreamedReplacement
}

func getPlanResult(params planResultParams) (*db.PlanFileResult, bool) {
	orgId := params.orgId
	planId := params.planId
	planBuildId := params.planBuildId
	filePath := params.filePath
	currentState := params.currentState
	contextPart := params.context
	streamedReplacements := params.streamedReplacements
	updated := params.currentState
	fileContent := params.fileContent

	currentStateLines := strings.Split(currentState, "\n")
	fileContentLines := strings.Split(fileContent, "\n")

	var replacements []*shared.Replacement
	for _, streamedReplacement := range streamedReplacements {
		var old string
		var new string

		if streamedReplacement.Old.StartLine == streamedReplacement.Old.EndLine {
			old = currentStateLines[streamedReplacement.Old.StartLine-1]
		} else {
			old = strings.Join(currentStateLines[streamedReplacement.Old.StartLine-1:streamedReplacement.Old.EndLine], "\n")
		}

		if streamedReplacement.New.StartLine == streamedReplacement.New.EndLine {
			new = fileContentLines[streamedReplacement.New.StartLine-1]
		} else {
			new = strings.Join(fileContentLines[streamedReplacement.New.StartLine-1:streamedReplacement.New.EndLine], "\n")
		}

		replacement := &shared.Replacement{
			Old:     old,
			New:     new,
			Summary: streamedReplacement.ShortSummary,
		}

		replacements = append(replacements, replacement)
	}

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
		OrgId:          orgId,
		PlanId:         planId,
		PlanBuildId:    planBuildId,
		ConvoMessageId: params.convoMessageId,
		Content:        "",
		Path:           filePath,
		Replacements:   replacements,
		ContextSha:     contextSha,
		ContextBody:    contextBody,
		AnyFailed:      !allSucceeded,
	}, allSucceeded
}
