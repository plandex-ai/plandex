package plan

import (
	"context"
	"fmt"
	"log"
	"plandex-server/db"
	diff_pkg "plandex-server/diff"
	"plandex-server/hooks"
	"plandex-server/syntax"
	"plandex-server/utils"
	"runtime"
	"runtime/debug"
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

	descLower := strings.ToLower(desc)
	isReplaceOrRemove := strings.Contains(descLower, "type: replace") || strings.Contains(descLower, "type: remove") || strings.Contains(descLower, "type: overwrite")

	var autoApplyRes *syntax.ApplyChangesResult
	var autoApplySyntaxErrors []string

	calledFastApply := false
	var fastApplyRes string
	fastApplyCh := make(chan string, 1)

	callFastApply := func() {
		log.Printf("buildStructuredEdits - %s - calling fast apply hook\n", filePath)
		fileState.builderRun.DidFastApply = true
		fileState.builderRun.FastApplyStartedAt = time.Now()
		calledFastApply = true

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in callFastApply: %v\n%s", r, debug.Stack())
					fastApplyCh <- ""
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()

			res, err := hooks.ExecHook(hooks.CallFastApply, hooks.HookParams{
				FastApplyParams: &hooks.FastApplyParams{
					InitialCode: originalFile,
					EditSnippet: proposedContent,
					Language:    fileState.language,
					Ctx:         buildCtx,
				},
			})

			if err != nil {
				log.Printf("buildStructuredEdits - error executing fast apply hook: %v\n", err)
				// empty string acts as a no-op
				fastApplyCh <- ""
				return
			} else if res.FastApplyResult == nil {
				log.Printf("buildStructuredEdits - fast apply hook returned nil result\n")
				// empty string acts as a no-op
				fastApplyCh <- ""
				return
			}

			fastApplyRes = res.FastApplyResult.MergedCode
			log.Printf("buildStructuredEdits - %s - got fast apply hook result\n", filePath)
			// fmt.Printf("buildStructuredEdits - fastApplyRes:\n%s", fastApplyRes)

			fileState.builderRun.FastApplyFinishedAt = time.Now()

			fastApplyCh <- fastApplyRes
		}()
	}

	if isReplaceOrRemove {
		callFastApply()
	}

	log.Printf("buildStructuredEdits - %s - applying changes\n", filePath)
	// Apply plan logic
	log.Printf("buildStructuredEdits - %s - calling ApplyChanges\n", filePath)
	autoApplyRes = syntax.ApplyChanges(
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
	// log.Printf("buildStructuredEdits - autoApplyRes.NewFile:\n\n%s", autoApplyRes.NewFile)
	log.Println("buildStructuredEdits - autoApplyRes.NeedsVerifyReasons:", autoApplyRes.NeedsVerifyReasons)

	autoApplySyntaxErrors = fileState.validateSyntax(buildCtx, autoApplyRes.NewFile)

	hasNeedsVerifyReasons := len(autoApplyRes.NeedsVerifyReasons) > 0

	autoApplyHasSyntaxErrors := len(autoApplySyntaxErrors) > 0
	autoApplyIsValid := !autoApplyHasSyntaxErrors && !hasNeedsVerifyReasons

	if !autoApplyIsValid && !calledFastApply {
		callFastApply()
	}

	log.Printf("buildStructuredEdits - %s - autoApplyHasSyntaxErrors: %t, hasNeedsVerifyReasons: %t, autoApplyIsValid: %t\n",
		filePath, autoApplyHasSyntaxErrors, hasNeedsVerifyReasons, autoApplyIsValid)

	updated := autoApplyRes.NewFile

	// If no problems, we trust the direct ApplyChanges result
	if autoApplyIsValid {
		log.Printf("buildStructuredEdits - %s - changes are valid, using ApplyChanges result\n", filePath)
		fileState.builderRun.AutoApplySuccess = true
	} else {
		log.Printf("buildStructuredEdits - %s - auto apply has syntax errors or NeedsVerifyReasons", filePath)
		fileState.builderRun.AutoApplyValidationReasons = make([]string, len(autoApplyRes.NeedsVerifyReasons))
		for i, reason := range autoApplyRes.NeedsVerifyReasons {
			fileState.builderRun.AutoApplyValidationReasons[i] = string(reason)
		}

		fileState.builderRun.AutoApplyValidationSyntaxErrors = autoApplySyntaxErrors

		buildRaceParams := buildRaceParams{
			updated:         updated,
			proposedContent: proposedContent,
			desc:            desc,
			reasons:         autoApplyRes.NeedsVerifyReasons,
			syntaxErrors:    autoApplySyntaxErrors,

			didCallFastApply: calledFastApply,
			fastApplyCh:      fastApplyCh,

			sessionId: activePlan.SessionId,
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

	// strip any blank lines from beginning/end of updated file
	updated = utils.StripAddedBlankLines(originalFile, updated)

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
