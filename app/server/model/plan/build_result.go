package plan

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"plandex-server/db"
	"plandex-server/syntax"
	"plandex-server/types"
	"sort"
	"strings"
	"time"

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
	OrgId               string
	PlanId              string
	PlanBuildId         string
	ConvoMessageId      string
	FilePath            string
	PreBuildState       string
	OverlapStrategy     OverlapStrategy
	ChangesWithLineNums []*shared.StreamedChangeWithLineNums

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
	streamedChangesWithLineNums := params.ChangesWithLineNums

	preBuildState = shared.AddLineNums(preBuildState)

	preBuildStateLines := strings.Split(preBuildState, "\n")

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

		startLine, endLine, err := streamedChange.GetLines()
		if err != nil {
			log.Printf("getPlanResult - File %s: Error getting lines from streamedChange: %v\n", filePath, err)
			return nil, "", false, fmt.Errorf("error getting lines from streamedChange: %v", err)
		}

		if startLine > len(preBuildStateLines) {
			log.Printf("getPlanResult - File %s: Start line is greater than preBuildStateLines length: %d > %d\n", filePath, startLine, len(preBuildStateLines))
			return nil, "", false, fmt.Errorf("start line is greater than preBuildStateLines length: %d > %d", startLine, len(preBuildStateLines))
		}

		if endLine < 1 {
			log.Printf("getPlanResult - File %s: End line is less than 1: %d\n", filePath, endLine)
			return nil, "", false, fmt.Errorf("end line is less than 1: %d", endLine)
		}
		if endLine > len(preBuildStateLines) {
			log.Printf("getPlanResult - File %s: End line is greater than preBuildStateLines length: %d > %d\n", filePath, endLine, len(preBuildStateLines))
			return nil, "", false, fmt.Errorf("end line is greater than preBuildStateLines length: %d > %d", endLine, len(preBuildStateLines))
		}

		if startLine < highestEndLine {
			log.Printf("getPlanResult - File %s: Start line is less than highestEndLine: %d < %d\n", filePath, startLine, highestEndLine)

			log.Printf("getPlanResult - File %s: streamedChange:\n", filePath)
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
				log.Printf("getPlanResult - File %s: End line is less than highestEndLine: %d < %d\n", filePath, endLine, highestEndLine)
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

		replacement := &shared.Replacement{
			Old:            old,
			New:            new,
			StreamedChange: streamedChange,
		}

		replacements = append(replacements, replacement)
	}

	log.Printf("getPlanResult - File %s: Will apply replacements\n", filePath)

	updated, allSucceeded := shared.ApplyReplacements(preBuildState, replacements, true)

	updated = shared.RemoveLineNums(updated)

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
		validationRes, err := syntax.Validate(ctx, filePath, updated)
		if err != nil {
			log.Printf("getPlanResult - File %s: Error validating syntax: %v\n", filePath, err)
			return nil, "", false, fmt.Errorf("error validating syntax: %v", err)
		}

		res.WillCheckSyntax = validationRes.HasParser && !validationRes.TimedOut
		res.SyntaxValid = validationRes.Valid
		res.SyntaxErrors = validationRes.Errors
	}

	return &res, updated, allSucceeded, nil
}

func (fileState *activeBuildStreamFileState) onBuildResult(res types.ChangesWithLineNums) {
	log.Printf("onBuildResult - File: %s\n", fileState.filePath)

	filePath := fileState.filePath
	build := fileState.build
	currentOrgId := fileState.currentOrgId
	planId := fileState.plan.Id
	branch := fileState.branch
	preBuildState := fileState.preBuildState

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		log.Printf("onBuildResult - File %s: Active plan not found for plan ID %s on branch %s\n", filePath, planId, branch)
		return
	}

	sorted := []*shared.StreamedChangeWithLineNums{}

	for _, change := range res.Changes {
		if change.HasChange {
			sorted = append(sorted, change)
		}
	}

	// Sort the streamed changes by start line
	sort.Slice(sorted, func(i, j int) bool {
		var iStartLine int
		var jStartLine int

		iStartLine, _, err := sorted[i].GetLines()

		if err != nil {
			log.Printf("onBuildResult - File %s: Error getting start line for change %v: %v\n", filePath, sorted[i], err)
			fileState.lineNumsRetryOrError(fmt.Errorf("onBuildResult - error getting start line for change %v: %v", sorted[i], err))
			return false
		}

		jStartLine, _, err = sorted[j].GetLines()

		if err != nil {
			log.Printf("onBuildResult - File %s: Error getting start line for change %v: %v\n", filePath, sorted[j], err)
			fileState.lineNumsRetryOrError(fmt.Errorf("onBuildResult - error getting start line for change %v: %v", sorted[j], err))
			return false
		}

		return iStartLine < jStartLine
	})

	log.Printf("onBuildResult - File %s: fileState.streamedChangesWithLineNums = sorted\n", filePath)
	log.Printf("onBuildResult - File %s: num changes: %d\n", filePath, len(sorted))

	fileState.streamedChangesWithLineNums = sorted

	var overlapStrategy OverlapStrategy = OverlapStrategyError
	if fileState.lineNumsNumRetry > 1 {
		overlapStrategy = OverlapStrategySkip
	}

	planFileResult, updatedFile, allSucceeded, err := GetPlanResult(
		activePlan.Ctx,
		PlanResultParams{
			OrgId:               currentOrgId,
			PlanId:              planId,
			PlanBuildId:         build.Id,
			ConvoMessageId:      build.ConvoMessageId,
			FilePath:            filePath,
			PreBuildState:       preBuildState,
			ChangesWithLineNums: res.Changes,
			OverlapStrategy:     overlapStrategy,
			CheckSyntax:         false,
		},
	)

	if err != nil {
		log.Printf("onBuildResult - File %s: Error getting plan result: %v\n", filePath, err)
		fileState.lineNumsRetryOrError(fmt.Errorf("onBuildResult - error getting plan result for file '%s': %v", filePath, err))
		return
	}

	if !allSucceeded {
		log.Printf("onBuildResult - File %s: Failed replacements:\n", filePath)
		for _, replacement := range planFileResult.Replacements {
			if replacement.Failed {
				spew.Dump(replacement)
			}
		}

		fileState.onBuildFileError(fmt.Errorf("onBuildResult - replacements failed for file '%s'", filePath))
		return
	}

	buildInfo := &shared.BuildInfo{
		Path:      filePath,
		NumTokens: 0,
		Finished:  true,
	}
	activePlan.Stream(shared.StreamMessage{
		Type:      shared.StreamMessageBuildInfo,
		BuildInfo: buildInfo,
	})
	time.Sleep(50 * time.Millisecond)

	fileState.updated = updatedFile

	log.Printf("onBuildResult - File %s: Plan file result: %v\n", filePath, planFileResult != nil)
	log.Printf("onBuildResult - File %s: updatedFile exists: %v\n", filePath, updatedFile != "")

	fileState.onFinishBuildFile(planFileResult, updatedFile)
}

func (fileState *activeBuildStreamFileState) lineNumsRetryOrError(err error) {
	if fileState.lineNumsNumRetry < MaxBuildStreamErrorRetries {
		fileState.lineNumsNumRetry++
		fileState.activeBuild.WithLineNumsBuffer = ""
		fileState.activeBuild.WithLineNumsBufferTokens = 0
		log.Printf("lineNumsRetryOrError - Retrying line nums build file '%s' due to error: %v\n", fileState.filePath, err)

		activePlan := GetActivePlan(fileState.plan.Id, fileState.branch)

		if activePlan == nil {
			log.Printf("lineNumsRetryOrError - File %s: Active plan not found\n", fileState.filePath)
			return
		}

		select {
		case <-activePlan.Ctx.Done():
			log.Printf("lineNumsRetryOrError - File %s: Context canceled. Exiting.\n", fileState.filePath)
			return
		case <-time.After(time.Duration((fileState.verifyFileNumRetry*fileState.verifyFileNumRetry)/2)*200*time.Millisecond + time.Duration(rand.Intn(500))*time.Millisecond):
			break
		}

		fileState.buildFileLineNums()
	} else {
		fileState.onBuildFileError(err)
	}
}
