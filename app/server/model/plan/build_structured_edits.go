package plan

import (
	"encoding/xml"
	"fmt"
	"log"
	"math/rand"
	"plandex-server/db"
	diff_pkg "plandex-server/diff"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/syntax"
	"plandex-server/types"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

const BuildStructuredEditsMaxTries = 4

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

	log.Printf("buildStructuredEdits - %s - applying changes\n", filePath)
	applyRes := syntax.ApplyChanges(
		originalFile,
		proposedContent,
		desc,
		true,
	)
	// log.Println("buildStructuredEdits - applyRes.NewFile:")
	// log.Println(applyRes.NewFile)

	proposedContent = applyRes.Proposed

	for len(applyRes.NeedsVerifyReasons) > 0 && numTries < BuildStructuredEditsMaxTries {
		log.Printf("buildStructuredEdits verify call - numTries: %d\n", numTries)

		numTries++

		log.Println("buildStructuredEdits - needs verify reasons:")
		spew.Dump(applyRes.NeedsVerifyReasons)

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

		validateSysPrompt := prompts.GetValidateEditsPrompt(filePath, originalFile, desc, proposedContent, diff, applyRes.NeedsVerifyReasons)

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

		fileState.inputTokens = inputTokens

		_, apiErr := hooks.ExecHook(hooks.WillSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: fileState.plan,
			WillSendModelRequestParams: &hooks.WillSendModelRequestParams{
				InputTokens:  inputTokens,
				OutputTokens: shared.AvailableModelsByName[fileState.settings.ModelPack.Builder.BaseModelConfig.ModelName].DefaultReservedOutputTokens,
				ModelName:    fileState.settings.ModelPack.Builder.BaseModelConfig.ModelName,
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
		spew.Dump(resp.Usage)

		_, apiErr = hooks.ExecHook(hooks.DidSendModelRequest, hooks.HookParams{
			Auth: auth,
			Plan: fileState.plan,
			DidSendModelRequestParams: &hooks.DidSendModelRequestParams{
				InputTokens:   resp.Usage.PromptTokens,
				OutputTokens:  resp.Usage.CompletionTokens,
				ModelName:     fileState.settings.ModelPack.Builder.BaseModelConfig.ModelName,
				ModelProvider: fileState.settings.ModelPack.Builder.BaseModelConfig.Provider,
				ModelPackName: fileState.settings.ModelPack.Name,
				ModelRole:     shared.ModelRoleBuilder,
				Purpose:       "File edit",
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

		fixedProposedUpdatesXmlString := GetXMLTag(content, "PlandexProposedUpdates", true)

		// log.Println("buildStructuredEdits - fixed proposed updates xml string:")
		// log.Println(fixedProposedUpdatesXmlString)

		if fixedProposedUpdatesXmlString == "" {
			// edits are valid
			break
		} else {
			var fixedProposedUpdates syntax.FixedProposedUpdatesTag
			err = xml.Unmarshal([]byte(fixedProposedUpdatesXmlString), &fixedProposedUpdates)
			if err != nil {
				log.Printf("buildStructuredEdits - error unmarshalling PlandexProposedUpdates xml: %v\n", err)
			}

			proposedContent = fixedProposedUpdates.Content

			// remove code block formatting if it was mistakenly included in the proposed content
			trimmedProposedContent := strings.TrimSpace(proposedContent)
			splitProposedContent := strings.Split(trimmedProposedContent, "\n")

			if len(splitProposedContent) > 2 {
				firstLine := strings.TrimSpace(splitProposedContent[0])
				secondLine := strings.TrimSpace(splitProposedContent[1])
				lastLine := strings.TrimSpace(splitProposedContent[len(splitProposedContent)-1])
				if types.LineMaybeHasFilePath(firstLine) && strings.HasPrefix(secondLine, "```") {
					if lastLine == "```" {
						proposedContent = strings.Join(splitProposedContent[1:len(splitProposedContent)-1], "\n")
					}
				} else if strings.HasPrefix(firstLine, "```") && lastLine == "```" {
					proposedContent = strings.Join(splitProposedContent[1:len(splitProposedContent)-1], "\n")
				}
			}

			// log.Println("buildStructuredEdits - fixed proposed content:")
			// log.Println(proposedContent)

			fixedDescString := GetXMLTag(content, "PlandexProposedUpdatesExplanation", false)

			// log.Println("buildStructuredEdits - fixed desc xml string:")
			// log.Println(fixedDescString)

			if fixedDescString != "" {
				var descElement syntax.ProposedUpdatesExplanationTag
				err = xml.Unmarshal([]byte(fixedDescString), &descElement)
				if err != nil {
					log.Printf("buildStructuredEdits - error unmarshalling PlandexProposedUpdatesExplanation xml: %v\n", err)
				}
				if descElement.Content != "" {
					desc = descElement.Content
				}
			}

			log.Printf("buildStructuredEdits - %s - applying changes again\n", filePath)

			applyRes = syntax.ApplyChanges(
				originalFile,
				proposedContent,
				desc,
				true,
			)

			log.Printf("buildStructuredEdits - %s - applyRes.NeedsVerifyReasons: %v\n", filePath, applyRes.NeedsVerifyReasons)

			proposedContent = applyRes.Proposed
		}
	}

	updatedFile := applyRes.NewFile

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
		CanVerify:      false, // no verification step with structural edits
	}

	fileState.onFinishBuildFile(&res, updatedFile)
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
