package plan

import (
	"context"
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/syntax"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
)

type OverlapStrategy int

const (
	OverlapStrategySkip OverlapStrategy = iota
	OverlapStrategyError
)

type PlanResultParams struct {
	OrgId                       string
	PlanId                      string
	PlanBuildId                 string
	ConvoMessageId              string
	FilePath                    string
	PreBuildState               string
	OverlapStrategy             OverlapStrategy
	StreamedChangesWithLineNums []*shared.StreamedChangeWithLineNums

	CheckSyntax bool

	IsFix       bool
	IsSyntaxFix bool
	IsOtherFix  bool
	FixEpoch    int
}

func GetPlanResult(ctx context.Context, params PlanResultParams) (*db.PlanFileResult, string, bool, error) {
	orgId := params.OrgId
	planId := params.PlanId
	planBuildId := params.PlanBuildId
	filePath := params.FilePath
	preBuildState := params.PreBuildState
	streamedChangesWithLineNums := params.StreamedChangesWithLineNums

	preBuildState = shared.AddLineNums(preBuildState)

	preBuildStateLines := strings.Split(preBuildState, "\n")

	// log.Printf("\n\ngetPlanResult - path: %s\n", filePath)
	// log.Println("getPlanResult - preBuildState:")
	// log.Println(preBuildState)
	// log.Println("getPlanResult - preBuildStateLines:")
	// log.Println(preBuildStateLines)
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

		if streamedChange.Old.EntireFile {
			replacements = append(replacements, &shared.Replacement{
				EntireFile:     true,
				Old:            old,
				New:            new,
				StreamedChange: streamedChange,
			})
			continue
		}

		// log.Printf("getPlanResult - streamedChange.Old.StartLine: %d\n", streamedChange.Old.StartLine)
		// log.Printf("getPlanResult - streamedChange.Old.EndLine: %d\n", streamedChange.Old.EndLine)

		startLine, endLine, err := streamedChange.GetLines()

		if err != nil {
			log.Println("getPlanResult - Error getting lines from streamedChange:", err)
			return nil, "", false, fmt.Errorf("error getting lines from streamedChange: %v", err)
		}

		if startLine > len(preBuildStateLines) {
			log.Printf("Start line is greater than preBuildStateLines length: %d > %d\n", startLine, len(preBuildStateLines))
			return nil, "", false, fmt.Errorf("start line is greater than preBuildStateLines length: %d > %d", startLine, len(preBuildStateLines))
		}

		if endLine < 1 {
			log.Printf("End line is less than 1: %d\n", endLine)
			return nil, "", false, fmt.Errorf("end line is less than 1: %d", endLine)
		}
		if endLine > len(preBuildStateLines) {
			log.Printf("End line is greater than preBuildStateLines length: %d > %d\n", endLine, len(preBuildStateLines))
			return nil, "", false, fmt.Errorf("end line is greater than preBuildStateLines length: %d > %d", endLine, len(preBuildStateLines))
		}

		if startLine < highestEndLine {
			log.Printf("Start line is less than highestEndLine: %d < %d\n", startLine, highestEndLine)

			log.Printf("streamedChange:\n")
			log.Println(spew.Sdump(streamedChangesWithLineNums))

			if params.OverlapStrategy == OverlapStrategyError {
				return nil, "", false, fmt.Errorf("start line is less than highestEndLine: %d < %d", startLine,
					highestEndLine)
			} else {
				continue
			}
		}

		if endLine < highestEndLine {
			if params.OverlapStrategy == OverlapStrategyError {
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
			old = preBuildStateLines[startLine-1]
		} else {
			old = strings.Join(preBuildStateLines[startLine-1:endLine], "\n")
		}

		// log.Printf("getPlanResult - old: %s\n", old)

		replacement := &shared.Replacement{
			Old:            old,
			New:            new,
			StreamedChange: streamedChange,
		}

		replacements = append(replacements, replacement)
	}

	log.Println("Will apply replacements")
	// log.Println("preBuildState:", preBuildState)

	// log.Println("Replacements:")
	// spew.Dump(replacements)

	updated, allSucceeded := shared.ApplyReplacements(preBuildState, replacements, true)

	updated = shared.RemoveLineNums(updated)

	// log sha256 hash of updated content
	// hash := sha256.Sum256([]byte(updated))
	// sha := hex.EncodeToString(hash[:])

	// log.Printf("apply result - %s - updated content hash: %s\n", filePath, sha)

	for _, replacement := range replacements {
		id := uuid.New().String()
		replacement.Id = id
	}

	res := db.PlanFileResult{
		TypeVersion:         1,
		ReplaceWithLineNums: true,
		OrgId:               orgId,
		PlanId:              planId,
		PlanBuildId:         planBuildId,
		ConvoMessageId:      params.ConvoMessageId,
		Content:             "",
		Path:                filePath,
		Replacements:        replacements,
		AnyFailed:           !allSucceeded,
		CanVerify:           !params.IsOtherFix,
		IsFix:               params.IsFix,
		IsSyntaxFix:         params.IsSyntaxFix,
		IsOtherFix:          params.IsOtherFix,
		FixEpoch:            params.FixEpoch,
	}

	if params.CheckSyntax {
		// validate syntax (if we have a parser)
		validationRes, err := syntax.Validate(ctx, filePath, updated)

		if err != nil {
			log.Println("Error validating syntax:", err)
			return nil, "", false, fmt.Errorf("error validating syntax: %v", err)
		}

		res.WillCheckSyntax = validationRes.HasParser && !validationRes.TimedOut
		res.SyntaxValid = validationRes.Valid
		res.SyntaxErrors = validationRes.Errors

	}

	// spew.Dump(res)

	return &res, updated, allSucceeded, nil
}
