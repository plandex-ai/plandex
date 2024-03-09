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
	convoMessageId  string
	filePath        string
	currentState    string
	fileContent     string
	streamedChanges []*shared.StreamedChange
}

func getPlanResult(params planResultParams) (*db.PlanFileResult, bool) {
	orgId := params.orgId
	planId := params.planId
	planBuildId := params.planBuildId
	filePath := params.filePath
	currentState := params.currentState
	streamedChanges := params.streamedChanges
	updated := params.currentState
	fileContent := params.fileContent

	currentStateLines := strings.Split(currentState, "\n")
	fileContentLines := strings.Split(fileContent, "\n")

	var replacements []*shared.Replacement
	for _, streamedChange := range streamedChanges {
		var old string
		var new string

		if streamedChange.New.StartLine == streamedChange.New.EndLine {
			new = fileContentLines[streamedChange.New.StartLine-1]
		} else {
			new = strings.Join(fileContentLines[streamedChange.New.StartLine-1:streamedChange.New.EndLine], "\n")
		}

		switch streamedChange.ChangeType {

		case shared.StreamedChangeTypeReplace:
			if streamedChange.Old.StartLine == streamedChange.Old.EndLine {
				old = currentStateLines[streamedChange.Old.StartLine-1]
			} else {
				old = strings.Join(currentStateLines[streamedChange.Old.StartLine-1:streamedChange.Old.EndLine], "\n")
			}

		case shared.StreamedChangeTypeAppend:
			old = currentStateLines[streamedChange.Old.EndLine-1]
			new = old + "\n" + new

		case shared.StreamedChangeTypePrepend:
			old = currentStateLines[streamedChange.Old.StartLine-1]
			new = new + "\n" + old
		}

		replacement := &shared.Replacement{
			Old:            old,
			New:            new,
			StreamedChange: streamedChange,
		}

		replacements = append(replacements, replacement)
	}

	sort.Slice(replacements, func(i, j int) bool {
		iIdx := strings.Index(updated, replacements[i].Old)
		jIdx := strings.Index(updated, replacements[j].Old)
		return iIdx < jIdx
	})

	_, allSucceeded := shared.ApplyReplacements(currentState, replacements, true)

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
		AnyFailed:      !allSucceeded,
	}, allSucceeded
}
