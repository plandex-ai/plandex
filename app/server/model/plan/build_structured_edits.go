package plan

import (
	"fmt"
	"log"
	"math/rand"
	"plandex-server/db"
	diff_pkg "plandex-server/diff"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/syntax"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

type BuildStage int

const (
	BuildStageInitial BuildStage = iota
	BuildStageValidateAndCorrect
	BuildStageValidateOnly
)

func (fileState *activeBuildStreamFileState) buildStructuredEdits() {
	auth := fileState.auth
	filePath := fileState.filePath
	activeBuild := fileState.activeBuild
	clients := fileState.clients
	planId := fileState.plan.Id
	branch := fileState.branch
	config := fileState.settings.ModelPack.Builder
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

	proposedContent := activeBuild.FileContent
	desc := activeBuild.FileDescription

	maxPotentialTokens := activeBuild.FileContentTokens + activeBuild.CurrentFileTokens

	log.Printf("buildStructuredEdits - %s - applying changes\n", filePath)

	applyRes := syntax.ApplyChanges(
		originalFile,
		proposedContent,
		desc,
		true,
		fileState.parser,
		fileState.language,
		activePlan.Ctx,
	)
	// log.Println("buildStructuredEdits - applyRes.NewFile:")
	// log.Println(applyRes.NewFile)

	updatedFile := applyRes.NewFile

	var syntaxErrors []string
	validateSyntax := func() {
		if fileState.parser != nil && !fileState.preBuildStateSyntaxInvalid && !fileState.syntaxCheckTimedOut {
			validationRes, err := syntax.ValidateWithParsers(activePlan.Ctx, fileState.language, fileState.parser, "", nil, updatedFile) // fallback parser was already set as fileState.parser if needed during initial preBuildState syntax check
			if err != nil {
				log.Printf("buildStructuredEdits - error validating updated file: %v\n", err)
			} else if validationRes.TimedOut {
				log.Printf("buildStructuredEdits - syntax check timed out for updated file\n")
				fileState.syntaxCheckTimedOut = true
				syntaxErrors = []string{}
			} else {
				syntaxErrors = validationRes.Errors
			}
		}
	}
	validateSyntax()

	isValid := len(applyRes.NeedsVerifyReasons) == 0 && len(syntaxErrors) == 0

	log.Printf("buildStructuredEdits - %s - initial isValid: %t\n", filePath, isValid)

	var buildStage BuildStage = BuildStageInitial

	for !isValid && int(buildStage) <= int(BuildStageValidateAndCorrect) {
		buildStage = BuildStage(int(buildStage) + 1)

		log.Printf("buildStructuredEdits - %s - buildStage: %d\n", filePath, buildStage)

		var diff string
		if applyRes.NewFile != "" {
			var err error
			diff, err = diff_pkg.GetDiffs(originalFile, applyRes.NewFile)
			if err != nil {
				log.Printf("buildStructuredEdits - error getting diffs: %v\n", err)
				fileState.structuredEditRetryOrError(fmt.Errorf("error getting diffs: %v", err))
				return
			}
		}
		// if diff != "" {
		// 	log.Println("buildStructuredEdits - diff:")
		// 	log.Println(diff)
		// }

		log.Printf("buildStructuredEdits - %s - desc:\n%s\n", filePath, desc)
		log.Printf("buildStructuredEdits - %s - proposedContent:\n%s\n", filePath, proposedContent)

		var reasons []syntax.NeedsVerifyReason
		if buildStage == BuildStageValidateAndCorrect {
			reasons = applyRes.NeedsVerifyReasons
		}

		validateSysPrompt := prompts.GetValidateEditsPrompt(prompts.ValidateEditsPromptParams{
			Path:           filePath,
			Original:       originalFile,
			Desc:           desc,
			Proposed:       proposedContent,
			Diff:           diff,
			Reasons:        reasons,
			SyntaxErrors:   syntaxErrors,
			FullCorrection: buildStage == BuildStageValidateAndCorrect,
		})

		// log.Printf("buildStructuredEdits - %s - validateSysPrompt:\n%s\n", filePath, validateSysPrompt)

		validateFileMessages := []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: validateSysPrompt,
			},
		}

		inputTokens := shared.GetMessagesTokenEstimate(validateFileMessages...) + shared.TokensPerRequest

		_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: fileState.plan,
			WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
				InputTokens:  inputTokens,
				OutputTokens: config.GetReservedOutputTokens(),
				ModelName:    config.BaseModelConfig.ModelName,
			},
		})
		if apiErr != nil {
			activePlan.StreamDoneCh <- apiErr
			return
		}

		log.Println("buildStructuredEdits - calling model for applied changes validation")

		modelReq := openai.ChatCompletionRequest{
			Model:       config.BaseModelConfig.ModelName,
			Messages:    validateFileMessages,
			Temperature: config.Temperature,
			TopP:        config.TopP,
			Stop:        []string{"<PlandexFinish/>"},
		}

		envVar := config.BaseModelConfig.ApiKeyEnvVar
		client := clients[envVar]

		resp, err := model.CreateChatCompletionWithRetries(client, activePlan.Ctx, modelReq)

		if err != nil {
			log.Printf("buildStructuredEdits - error calling model: %v\n", err)
			fileState.structuredEditRetryOrError(fmt.Errorf("error calling model: %v", err))
			return
		}

		log.Println("buildStructuredEdits - usage:")
		log.Println(spew.Sdump(resp.Usage))

		_, apiErr = hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: fileState.plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:   resp.Usage.PromptTokens,
				OutputTokens:  resp.Usage.CompletionTokens,
				ModelName:     config.BaseModelConfig.ModelName,
				ModelProvider: config.BaseModelConfig.Provider,
				ModelPackName: fileState.settings.ModelPack.Name,
				ModelRole:     shared.ModelRoleBuilder,
				Purpose:       "File edit (structured edits)",
			},
		})

		if apiErr != nil {
			activePlan.StreamDoneCh <- apiErr
			return
		}

		if len(resp.Choices) == 0 {
			log.Printf("buildStructuredEdits - no choices in response\n")
			fileState.structuredEditRetryOrError(fmt.Errorf("no choices in response"))
			return
		}

		refsChoice := resp.Choices[0]
		content := refsChoice.Message.Content

		log.Printf("buildStructuredEdits - %s - content:\n%s\n", filePath, content)

		if buildStage == BuildStageValidateAndCorrect {
			diffsValid := strings.Contains(content, "<PlandexCorrect/>")
			log.Printf("buildStructuredEdits - %s - diffsValid: %t\n", filePath, diffsValid)
			if diffsValid {
				isValid = true
				break
			}

			fixedDesc := GetXMLContent(content, "PlandexProposedUpdatesExplanation")

			// log.Println("buildStructuredEdits - fixed desc xml string:")
			// log.Println(fixedDescString)

			if fixedDesc != "" {
				desc = fixedDesc
			}

			replacement := GetXMLContent(content, "PlandexReplacement")

			if replacement == "" {
				log.Printf("buildStructuredEdits - no PlandexReplacement tag found in content\n")

				if fixedDesc != "" {
					fileState.structuredEditRetryOrError(fmt.Errorf("no PlandexReplacement tag found in content"))
					return
				}

				isValid = true
				break
			} else {
				log.Printf("buildStructuredEdits - replacement: %s\n", replacement)
			}

			replaceOld := GetXMLContent(replacement, "PlandexOld")

			if replaceOld == "" {
				log.Printf("buildStructuredEdits - no PlandexOld tag found in replacement\n")
				isValid = false
				break
			} else {
				replaceOld = strings.TrimSpace(replaceOld)
			}

			replaceOld = syntax.FindUniqueReplacement(originalFile, replaceOld)

			if replaceOld == "" {
				log.Printf("buildStructuredEdits - PlandexOld tag content not found or not unique in original file\n")
				isValid = false
				break
			}

			replaceNew := GetXMLContent(replacement, "PlandexNew")

			// handle replacement
			updatedFile = strings.Replace(updatedFile, replaceOld, replaceNew, 1)

			// validate updated file syntax
			validateSyntax()

		} else if buildStage == BuildStageValidateOnly {
			isValid = strings.Contains(content, "<PlandexCorrect/>")
		}
	}

	if isValid {
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

		replacements, err := diff_pkg.GetDiffReplacements(originalFile, updatedFile)
		if err != nil {
			log.Printf("buildStructuredEdits - error getting diff replacements: %v\n", err)
			fileState.structuredEditRetryOrError(fmt.Errorf("error getting diff replacements: %v", err))
			return
		}

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

		fileState.onFinishBuildFile(&res)

	} else {
		log.Println("buildStructuredEdits - isValid is false after structured edits and correction steps")

		wholeFileConfig := fileState.settings.ModelPack.GetWholeFileBuilder()

		// spew.Dump(wholeFileConfig)

		reservedOutputTokens := wholeFileConfig.GetReservedOutputTokens()
		threshold := int(float32(reservedOutputTokens) * 0.9)
		isBelowWholeFileThreshold := maxPotentialTokens < threshold

		if isBelowWholeFileThreshold {
			log.Printf("buildStructuredEdits - %s - total possible tokens %d is less than reserved output tokens %d - building whole file\n", filePath, maxPotentialTokens, reservedOutputTokens)

			fileState.buildWholeFileFallback(proposedContent, desc)
			return
		} else {
			// can't build whole file due to token output limit
			log.Printf("buildStructuredEdits - %s - can't build whole file due to token output limit\n", filePath)

			fileState.onBuildFileError(fmt.Errorf("file %s structured edits build failed", filePath))
		}
	}
}

func (fileState *activeBuildStreamFileState) structuredEditRetryOrError(err error) {
	if fileState.structuredEditNumRetry < MaxBuildErrorRetries {
		fileState.structuredEditNumRetry++

		log.Printf("buildStructuredEdits - retrying structured edits file '%s' due to error: %v\n", fileState.filePath, err)

		activePlan := GetActivePlan(fileState.plan.Id, fileState.branch)

		if activePlan == nil {
			log.Printf("buildStructuredEdits - active plan not found for plan ID %s and branch %s\n", fileState.plan.Id, fileState.branch)
			fileState.onBuildFileError(fmt.Errorf("active plan not found for plan ID %s and branch %s", fileState.plan.Id, fileState.branch))
			return
		}

		select {
		case <-activePlan.Ctx.Done():
			log.Printf("buildStructuredEdits - context canceled\n")
			return
		case <-time.After(time.Duration(fileState.structuredEditNumRetry*fileState.structuredEditNumRetry)*200*time.Millisecond + time.Duration(rand.Intn(500))*time.Millisecond):
			break
		}

		fileState.buildStructuredEdits()
	} else {
		fileState.onBuildFileError(err)
	}

}
