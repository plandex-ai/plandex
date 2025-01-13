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

// for smaller files, build whole file after the initial apply attempt and one retry
// for larger files that are under the whole file output limit, build whole file after the initial apply attempt and two retries
// for large files that are over the whole file output limit, build whole file after the initial apply attempt and four retries
const SmallFileThreshold = 2000
const BuildStructuredEditsMaxTries = 5
const BuildTriesBeforeWholeFile = 3
const SmallFileTriesBeforeWholeFile = 2

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

	numTries := 1

	proposedContent := activeBuild.FileContent
	desc := activeBuild.FileDescription

	maxPotentialTokens := activeBuild.FileContentTokens + activeBuild.CurrentFileTokens

	log.Printf("buildStructuredEdits - %s - applying changes\n", filePath)

	applyRes := syntax.ApplyChanges(
		originalFile,
		proposedContent,
		desc,
		true,
	)
	// log.Println("buildStructuredEdits - applyRes.NewFile:")
	// log.Println(applyRes.NewFile)

	var syntaxErrors []string
	validateSyntax := func() {
		if fileState.parser != nil && !fileState.preBuildStateSyntaxInvalid && !fileState.syntaxCheckTimedOut {
			validationRes, err := syntax.ValidateWithParsers(activePlan.Ctx, fileState.language, fileState.parser, "", nil, proposedContent) // fallback parser was already set as fileState.parser if needed during initial preBuildState syntax check
			if err != nil {
				log.Printf("buildStructuredEdits - error validating proposed content: %v\n", err)
			} else if validationRes.TimedOut {
				log.Printf("buildStructuredEdits - syntax check timed out for proposed content\n")
				fileState.syntaxCheckTimedOut = true
				syntaxErrors = []string{}
			} else {
				syntaxErrors = validationRes.Errors
			}
		}
	}
	validateSyntax()

	for (len(applyRes.NeedsVerifyReasons) > 0 || len(syntaxErrors) > 0) && numTries < BuildStructuredEditsMaxTries {
		log.Printf("buildStructuredEdits verify call - numTries: %d\n", numTries)
		log.Println("buildStructuredEdits - needs verify reasons:")
		log.Println(spew.Sdump(applyRes.NeedsVerifyReasons))

		wholeFileConfig := fileState.settings.ModelPack.GetWholeFileBuilder()

		// spew.Dump(wholeFileConfig)

		reservedOutputTokens := wholeFileConfig.GetReservedOutputTokens()
		isSmallFile := maxPotentialTokens < SmallFileThreshold
		threshold := int(float32(reservedOutputTokens) * 0.9)
		isBelowWholeFileThreshold := maxPotentialTokens < threshold
		shouldBuildSmallFile := isSmallFile && numTries == SmallFileTriesBeforeWholeFile && isBelowWholeFileThreshold
		shouldBuildAnyFile := numTries == BuildTriesBeforeWholeFile && isBelowWholeFileThreshold
		shouldBuildWholeFile := shouldBuildSmallFile || shouldBuildAnyFile

		// log all the above variables
		log.Printf("buildStructuredEdits - variables for whole file decision:\n"+
			"filePath: %s\n"+
			"content tokens: %d\n"+
			"current file tokens: %d\n"+
			"maxPotentialTokens: %d\n"+
			"reservedOutputTokens: %d\n"+
			"threshold: %d\n"+
			"isSmallFile: %t\n"+
			"isBelowWholeFileThreshold: %t\n"+
			"shouldBuildSmallFile: %t\n"+
			"shouldBuildAnyFile: %t\n"+
			"shouldBuildWholeFile: %t\n"+
			"numTries: %d",
			filePath,
			activeBuild.FileContentTokens,
			activeBuild.CurrentFileTokens,
			maxPotentialTokens,
			reservedOutputTokens,
			threshold,
			isSmallFile,
			isBelowWholeFileThreshold,
			shouldBuildSmallFile,
			shouldBuildAnyFile,
			shouldBuildWholeFile,
			numTries)

		log.Printf("buildStructuredEdits - %s - num tries %d - should build small file %t - should build any file %t - should build whole file %t\n", filePath, numTries, shouldBuildSmallFile, shouldBuildAnyFile, shouldBuildWholeFile)

		if shouldBuildWholeFile {
			log.Printf("buildStructuredEdits - %s - num tries %d - total possible tokens %d is less than reserved output tokens %d - building whole file\n", filePath, numTries, maxPotentialTokens, reservedOutputTokens)

			fileState.buildWholeFileFallback(proposedContent, desc)
			return
		}

		log.Println("buildStructuredEdits - getting verify edits prompt")

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

		validateSysPrompt := prompts.GetValidateEditsPrompt(filePath, originalFile, desc, proposedContent, diff, applyRes.NeedsVerifyReasons, syntaxErrors)

		validateFileMessages := []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: validateSysPrompt,
			},
		}

		promptTokens, err := shared.GetNumTokens(validateSysPrompt)

		if err != nil {
			log.Printf("buildStructuredEdits - error getting num tokens for prompt: %v\n", err)
			fileState.onBuildFileError(fmt.Errorf("error getting num tokens for prompt: %v", err))
			return
		}

		inputTokens := prompts.ExtraTokensPerRequest + prompts.ExtraTokensPerMessage + promptTokens

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

		fixedProposedUpdates := GetXMLContent(content, "PlandexProposedUpdates")

		// log.Println("buildStructuredEdits - fixed proposed updates xml string:")
		// log.Println(fixedProposedUpdatesXmlString)

		if fixedProposedUpdates == "" {
			// edits are valid
			break
		} else {
			proposedContent = fixedProposedUpdates

			// remove code block formatting if it was mistakenly included in the proposed content
			proposedContent = StripBackticksWrapper(proposedContent)

			// log.Println("buildStructuredEdits - fixed proposed content:")
			// log.Println(proposedContent)

			fixedDesc := GetXMLContent(content, "PlandexProposedUpdatesExplanation")

			// log.Println("buildStructuredEdits - fixed desc xml string:")
			// log.Println(fixedDescString)

			if fixedDesc != "" {
				desc = fixedDesc
			}

			log.Printf("buildStructuredEdits - %s - applying changes again\n", filePath)

			applyRes = syntax.ApplyChanges(
				originalFile,
				proposedContent,
				desc,
				true,
			)

			log.Printf("buildStructuredEdits - %s - applyRes.NeedsVerifyReasons: %v\n", filePath, applyRes.NeedsVerifyReasons)

			validateSyntax()
			numTries++
		}
	}

	updatedFile := applyRes.NewFile

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
