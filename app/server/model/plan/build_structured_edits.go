package plan

import (
	"context"
	"fmt"
	"log"
	"plandex-server/db"
	diff_pkg "plandex-server/diff"
	"plandex-server/syntax"
	"strings"
	"time"

	shared "plandex-shared"
)

func (fileState *activeBuildStreamFileState) buildStructuredEdits() {
	filePath := fileState.filePath
	activeBuild := fileState.activeBuild
	planId := fileState.plan.Id
	branch := fileState.branch
	originalFile := fileState.preBuildState
	parser := fileState.parser

	if parser == nil {
		log.Printf("buildStructuredEdits - tree-sitter parser is nil for file %s\n", filePath)
	}

	activePlan := GetActivePlan(planId, branch)
	if activePlan == nil {
		log.Printf("Active plan not found for plan ID %s and branch %s\n", planId, branch)
		fileState.onBuildFileError(fmt.Errorf("active plan not found for plan ID %s and branch %s", planId, branch))
		return
	}

	buildCtx, cancelBuild := context.WithCancel(activePlan.Ctx)

	proposedContent := activeBuild.FileContent
	desc := activeBuild.FileDescription

	log.Printf("buildStructuredEdits - %s - applying changes\n", filePath)

	// Apply plan logic
	log.Printf("buildStructuredEdits - %s - calling ApplyChanges\n", filePath)
	applyRes := syntax.ApplyChanges(
		buildCtx,
		syntax.ApplyChangesParams{
			Original:               originalFile,
			Proposed:               proposedContent,
			Desc:                   desc,
			AddMissingStartEndRefs: true,
			Parser:                 fileState.parser,
			Language:               fileState.language,
		},
	)
	log.Printf("buildStructuredEdits - %s - got ApplyChanges result\n", filePath)
	// log.Println("buildStructuredEdits - applyRes.NewFile:", applyRes.NewFile)
	log.Println("buildStructuredEdits - applyRes.NeedsVerifyReasons:", applyRes.NeedsVerifyReasons)

	// Log information about removed blocks
	if len(applyRes.BlocksRemoved) > 0 {
		log.Printf("buildStructuredEdits - %s - detected %d removed code blocks\n", filePath, len(applyRes.BlocksRemoved))

		// Log details about each removed block if not too verbose
		if len(applyRes.BlocksRemoved) <= 5 {
			for i, block := range applyRes.BlocksRemoved {
				log.Printf("buildStructuredEdits - removed block %d: lines %d-%d, %d characters\n", i+1, block.Start, block.End, len(block.Content))
			}
		}

		// If we have removed blocks but no CodeRemoved reason, add it
		hasCodeRemovedReason := false
		for _, reason := range applyRes.NeedsVerifyReasons {
			if reason == syntax.NeedsVerifyReasonCodeRemoved {
				hasCodeRemovedReason = true
				break
			}
		}

		if !hasCodeRemovedReason {
			log.Printf("buildStructuredEdits - %s - adding CodeRemoved verification reason due to detected removed blocks\n", filePath)
			applyRes.NeedsVerifyReasons = append(applyRes.NeedsVerifyReasons, syntax.NeedsVerifyReasonCodeRemoved)
		}
	}

	// Syntax check
	log.Printf("buildStructuredEdits - %s - validating syntax\n", filePath)
	syntaxErrors := fileState.validateSyntax(buildCtx, applyRes.NewFile)
	log.Printf("buildStructuredEdits - %s - got syntax validation result\n", filePath)
	if len(syntaxErrors) > 0 {
		log.Println("buildStructuredEdits - syntax errors:", syntaxErrors)
	}

	hasSyntaxErrors := len(syntaxErrors) > 0
	hasNeedsVerifyReasons := len(applyRes.NeedsVerifyReasons) > 0
	isValid := !hasSyntaxErrors && !hasNeedsVerifyReasons

	log.Printf("buildStructuredEdits - %s - hasSyntaxErrors: %t, hasNeedsVerifyReasons: %t, isValid: %t\n",
		filePath, hasSyntaxErrors, hasNeedsVerifyReasons, isValid)

	updated := applyRes.NewFile

	// If no problems, we trust the direct ApplyChanges result
	if isValid {
		log.Printf("buildStructuredEdits - %s - changes are valid, using ApplyChanges result\n", filePath)
		fileState.builderRun.AutoApplySuccess = true
	} else {

		log.Printf("buildStructuredEdits - %s - changes need validation/fixing\n", filePath)
		fileState.builderRun.AutoApplyValidationReasons = make([]string, len(applyRes.NeedsVerifyReasons))
		for i, reason := range applyRes.NeedsVerifyReasons {
			fileState.builderRun.AutoApplyValidationReasons[i] = string(reason)
		}
		fileState.builderRun.AutoApplyValidationSyntaxErrors = syntaxErrors

		buildRaceParams := buildRaceParams{
			updated:         updated,
			proposedContent: proposedContent,
			desc:            desc,
			reasons:         applyRes.NeedsVerifyReasons,
			syntaxErrors:    syntaxErrors,
			blocksRemoved:   applyRes.BlocksRemoved, // Pass removed blocks to buildRace
		}

		buildRaceResult, err := fileState.buildRace(buildCtx, cancelBuild, buildRaceParams)
		if err != nil {
			if apiErr, ok := err.(*shared.ApiError); ok {
				activePlan.StreamDoneCh <- apiErr
				return
			} else {
				log.Printf("buildStructuredEdits - %s - error building race: %v\n", filePath, err)
				fileState.onBuildFileError(fmt.Errorf("error building race: %v", err))
			}
			return
		}

		updated = buildRaceResult.content
	}

	// output diff and store build results
	buildInfo := &shared.BuildInfo{
		Path:      filePath,
		NumTokens: 0,
		Finished:  true,
	}
	log.Printf("streaming build info for finished file %s\n", filePath)
	activePlan.Stream(shared.StreamMessage{
		Type:      shared.StreamMessageBuildInfo,
		BuildInfo: buildInfo,
	})
	time.Sleep(50 * time.Millisecond)

	log.Printf("buildStructuredEdits - %s - getting diff replacements\n", filePath)
	replacements, err := diff_pkg.GetDiffReplacements(originalFile, updated)
	if err != nil {
		log.Printf("buildStructuredEdits - error getting diff replacements: %v\n", err)
		fileState.onBuildFileError(fmt.Errorf("error getting diff replacements: %v", err))
		return
	}
	log.Printf("buildStructuredEdits - %s - got %d replacements\n", filePath, len(replacements))

	for _, replacement := range replacements {
		replacement.Summary = strings.TrimSpace(desc)
	}

	res := db.PlanFileResult{
		TypeVersion:    1,
		OrgId:          fileState.plan.OrgId,
		PlanId:         fileState.plan.Id,
		PlanBuildId:    fileState.build.Id,
		ConvoMessageId: fileState.convoMessageId,
		Content:        "",
		Path:           filePath,
		Replacements:   replacements,
		SyntaxErrors:   syntaxErrors,
	}

	log.Printf("buildStructuredEdits - %s - finishing build file\n", filePath)
	fileState.onFinishBuildFile(&res)
}

func (fileState *activeBuildStreamFileState) validateSyntax(buildCtx context.Context, updated string) []string {
	if fileState.parser != nil && !fileState.preBuildStateSyntaxInvalid && !fileState.syntaxCheckTimedOut {
		validationRes, err := syntax.ValidateWithParsers(buildCtx, fileState.language, fileState.parser, "", nil, updated) // fallback parser was already set as fileState.parser if needed during initial preBuildState syntax check
		if err != nil {
			log.Printf("buildStructuredEdits - error validating updated file: %v\n", err)
		} else if validationRes.TimedOut {
			log.Printf("buildStructuredEdits - syntax check timed out for updated file\n")
			fileState.syntaxCheckTimedOut = true
			return nil
		} else {
			return validationRes.Errors
		}
	}

	return nil
}
