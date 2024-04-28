package plan

import (
	"fmt"
	"log"
	"plandex-server/db"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
)

type OverlapStrategy int

const (
	OverlapStrategySkip OverlapStrategy = iota
	OverlapStrategyError
)

type planResultParams struct {
	orgId                       string
	planId                      string
	planBuildId                 string
	convoMessageId              string
	filePath                    string
	currentState                string
	fileContent                 string
	overlapStrategy             OverlapStrategy
	streamedChangesWithLineNums []*shared.StreamedChangeWithLineNums
}

func getPlanResult(params planResultParams) (*db.PlanFileResult, string, bool, error) {
	orgId := params.orgId
	planId := params.planId
	planBuildId := params.planBuildId
	filePath := params.filePath
	currentState := params.currentState
	streamedChangesWithLineNums := params.streamedChangesWithLineNums
	// fileContent := params.fileContent

	currentState = shared.AddLineNums(currentState)

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

	var highestEndLine int = 0

	for _, streamedChange := range streamedChangesWithLineNums {
		if !streamedChange.HasChange {
			continue
		}

		var old string

		new := streamedChange.New

		// log.Printf("getPlanResult - streamedChange.Old.StartLine: %d\n", streamedChange.Old.StartLine)
		// log.Printf("getPlanResult - streamedChange.Old.EndLine: %d\n", streamedChange.Old.EndLine)

		startLine, endLine, err := streamedChange.GetLines()

		if err != nil {
			log.Println("getPlanResult - Error getting lines from streamedChange:", err)
			return nil, "", false, fmt.Errorf("error getting lines from streamedChange: %v", err)
		}

		if startLine > len(currentStateLines) {
			log.Printf("Start line is greater than currentStateLines length: %d > %d\n", startLine, len(currentStateLines))
			return nil, "", false, fmt.Errorf("start line is greater than currentStateLines length: %d > %d", startLine, len(currentStateLines))
		}

		if endLine < 1 {
			log.Printf("End line is less than 1: %d\n", endLine)
			return nil, "", false, fmt.Errorf("end line is less than 1: %d", endLine)
		}
		if endLine > len(currentStateLines) {
			log.Printf("End line is greater than currentStateLines length: %d > %d\n", endLine, len(currentStateLines))
			return nil, "", false, fmt.Errorf("end line is greater than currentStateLines length: %d > %d", endLine, len(currentStateLines))
		}

		if startLine < highestEndLine {
			log.Printf("Start line is less than highestEndLine: %d < %d\n", startLine, highestEndLine)

			if params.overlapStrategy == OverlapStrategyError {
				return nil, "", false, fmt.Errorf("start line is less than highestEndLine: %d < %d", startLine,
					highestEndLine)
			} else {
				continue
			}
		}

		if endLine < highestEndLine {
			if params.overlapStrategy == OverlapStrategyError {
				log.Printf("End line is less than highestEndLine: %d < %d\n", endLine, highestEndLine)
				return nil, "", false, fmt.Errorf("end line is less than highestEndLine: %d < %d", endLine, highestEndLine)
			} else {
				continue
			}
		}

		if endLine > highestEndLine {
			highestEndLine = endLine
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

	log.Println("Will apply replacements")
	// log.Println("currentState:", currentState)

	// log.Println("Replacements:")
	// spew.Dump(replacements)

	updated, allSucceeded := shared.ApplyReplacements(currentState, replacements, true)

	updated = shared.RemoveLineNums(updated)

	// log sha256 hash of updated content
	// hash := sha256.Sum256([]byte(updated))
	// sha := hex.EncodeToString(hash[:])

	// log.Printf("apply result - %s - updated content hash: %s\n", filePath, sha)

	for _, replacement := range replacements {
		id := uuid.New().String()
		replacement.Id = id
	}

	return &db.PlanFileResult{
		TypeVersion:         1,
		ReplaceWithLineNums: true,
		OrgId:               orgId,
		PlanId:              planId,
		PlanBuildId:         planBuildId,
		ConvoMessageId:      params.convoMessageId,
		Content:             "",
		Path:                filePath,
		Replacements:        replacements,
		AnyFailed:           !allSucceeded,
	}, updated, allSucceeded, nil
}
