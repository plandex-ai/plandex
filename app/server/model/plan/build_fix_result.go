package plan

import (
	"fmt"
	"log"
	"math/rand"
	"plandex-server/types"
	"sort"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
)

func (fileState *activeBuildStreamFileState) onFixResult(res types.ChangesWithLineNums) {

	filePath := fileState.filePath
	build := fileState.build
	currentOrgId := fileState.currentOrgId
	planId := fileState.plan.Id
	branch := fileState.branch

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		log.Printf("listenStreamFixChanges - Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return
	}
	sorted := []*shared.StreamedChangeWithLineNums{}

	// Sort the changes by start line
	for _, change := range res.Changes {
		if change.HasChange {
			sorted = append(sorted, change)
		}
	}

	// Sort the changes by start line
	sort.Slice(sorted, func(i, j int) bool {
		var iStartLine int
		var jStartLine int

		// Convert the line number part to an integer
		iStartLine, _, err := sorted[i].GetLines()

		if err != nil {
			log.Printf("listenStream - Error getting start line for change %v: %v\n", sorted[i], err)
			fileState.lineNumsRetryOrError(fmt.Errorf("listenStream - error getting start line for change %v: %v", sorted[i], err))
			return false
		}

		jStartLine, _, err = sorted[j].GetLines()

		if err != nil {
			log.Printf("listenStream - Error getting start line for change %v: %v\n", sorted[j], err)
			fileState.lineNumsRetryOrError(fmt.Errorf("listenStream - error getting start line for change %v: %v", sorted[j], err))
			return false
		}

		return iStartLine < jStartLine
	})

	fileState.streamedChangesWithLineNums = sorted

	var overlapStrategy OverlapStrategy = OverlapStrategyError
	if fileState.fixFileNumRetry > 1 {
		overlapStrategy = OverlapStrategySkip
	}

	planFileResult, updated, allSucceeded, err := GetPlanResult(
		activePlan.Ctx,
		PlanResultParams{
			OrgId:               currentOrgId,
			PlanId:              planId,
			PlanBuildId:         build.Id,
			ConvoMessageId:      build.ConvoMessageId,
			FilePath:            filePath,
			PreBuildState:       fileState.updated,
			ChangesWithLineNums: res.Changes,
			OverlapStrategy:     overlapStrategy,

			IsFix:       true,
			IsSyntaxFix: fileState.isFixingSyntax,
			IsOtherFix:  fileState.isFixingOther,

			FixEpoch: fileState.syntaxNumEpoch,

			CheckSyntax: true,
		},
	)

	if err != nil {
		log.Println("listenStreamFixChanges - Error getting plan result:", err)
		fileState.fixRetryOrAbort(fmt.Errorf("listenStreamFixChanges - error getting plan result for file '%s': %v", filePath, err))
		return
	}

	if !allSucceeded {
		log.Println("listenStreamFixChanges - Failed replacements:")
		for _, replacement := range planFileResult.Replacements {
			if replacement.Failed {
				spew.Dump(replacement)
			}
		}

		// no retry here as this should never happen
		fileState.onBuildFileError(fmt.Errorf("listenStreamFixChanges - replacements failed for file '%s'", filePath))
		return

	}

	log.Println("listenStreamFixChanges - Plan file result:")
	// spew.Dump(planFileResult)

	// reset fix state
	fileState.isFixingSyntax = false
	fileState.isFixingOther = false

	// if we are below the number of FixSyntaxRetries, and the syntax is invalid, short-circuit here and retry
	// otherwise if the syntax is invalid but we're out of retries, continue to onFinishBuildFile, which will handle epoch-based retries (i.e. running additional fixes on top of this failed fix) if applicable
	if planFileResult.WillCheckSyntax && !planFileResult.SyntaxValid {
		if fileState.syntaxNumRetry < FixSyntaxRetries {
			fileState.isFixingSyntax = true
			fileState.syntaxNumRetry++
			go fileState.fixFileLineNums()
			return
		}
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

	fileState.onFinishBuildFile(planFileResult, updated)
}

func (fileState *activeBuildStreamFileState) fixRetryOrAbort(err error) {
	if fileState.fixFileNumRetry < MaxBuildStreamErrorRetries {
		fileState.fixFileNumRetry++
		fileState.activeBuild.FixBuffer = ""
		fileState.activeBuild.FixBufferTokens = 0
		log.Printf("Retrying fix file '%s' due to error: %v\n", fileState.filePath, err)

		// Exponential backoff
		time.Sleep(time.Duration((fileState.verifyFileNumRetry*fileState.verifyFileNumRetry)/2)*200*time.Millisecond + time.Duration(rand.Intn(500))*time.Millisecond)

		fileState.fixFileLineNums()
	} else {
		log.Printf("Aborting fix file '%s' due to error: %v\n", fileState.filePath, err)

		activePlan := GetActivePlan(fileState.plan.Id, fileState.branch)

		if activePlan == nil {
			log.Println("fixRetryOrAbort - Active plan not found")
			return
		}

		buildInfo := &shared.BuildInfo{
			Path:      fileState.filePath,
			NumTokens: 0,
			Finished:  true,
		}
		activePlan.Stream(shared.StreamMessage{
			Type:      shared.StreamMessageBuildInfo,
			BuildInfo: buildInfo,
		})
		time.Sleep(50 * time.Millisecond)

		fileState.onFinishBuildFile(nil, "")
	}
}
