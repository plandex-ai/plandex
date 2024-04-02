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
	// fileContent := params.fileContent

	currentStateLines := strings.Split(currentState, "\n")
	// fileContentLines := strings.Split(fileContent, "\n")

	// log.Printf("\n\ngetPlanResult - path: %s\n", filePath)
	// log.Println("getPlanResult - currentState:")
	// log.Println(currentState)
	// log.Println("getPlanResult - currentStateLines:")
	// log.Println(currentStateLines)
	// log.Println("getPlanResult - fileContent:")
	// log.Println(fileContent)
	// log.Print("\n\n")

	var replacements []*shared.Replacement
	for _, streamedChange := range streamedChanges {
		var old string

		new := streamedChange.New

		// log.Printf("getPlanResult - streamedChange.Old.StartLine: %d\n", streamedChange.Old.StartLine)
		// log.Printf("getPlanResult - streamedChange.Old.EndLine: %d\n", streamedChange.Old.EndLine)

		startLine := streamedChange.Old.StartLine
		endLine := streamedChange.Old.EndLine

		if startLine < 1 {
			startLine = 1
		}
		if startLine > len(currentStateLines) {
			startLine = len(currentStateLines)
		}

		if endLine < 1 {
			endLine = 1
		}
		if endLine > len(currentStateLines) {
			endLine = len(currentStateLines)
		}

		if startLine == endLine {
			old = currentStateLines[startLine-1]
		} else {
			old = strings.Join(currentStateLines[startLine-1:endLine], "\n")
		}

		// log.Printf("getPlanResult - old: %s\n", old)

		replacement := &shared.Replacement{
			Old:            old,
			New:            new,
			StreamedChange: streamedChange,
		}

		replacements = append(replacements, replacement)
	}

	sort.Slice(replacements, func(i, j int) bool {
		iIdx := strings.Index(currentState, replacements[i].Old)
		jIdx := strings.Index(currentState, replacements[j].Old)
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
