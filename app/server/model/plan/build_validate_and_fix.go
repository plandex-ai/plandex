package plan

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	diff_pkg "plandex-server/diff"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/syntax"
	"plandex-server/types"
	"plandex-server/utils"
	shared "plandex-shared"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

const MaxValidationFixAttempts = 4

type buildValidateLoopParams struct {
	originalFile         string
	updated              string
	proposedContent      string
	desc                 string
	syntaxErrors         []string
	reasons              []syntax.NeedsVerifyReason
	initialPhaseOnStream func(chunk string, buffer string) bool
}

type buildValidateLoopResult struct {
	valid   bool
	updated string
}

func (fileState *activeBuildStreamFileState) buildValidateLoop(
	ctx context.Context,
	params buildValidateLoopParams,
) (buildValidateLoopResult, error) {
	log.Printf("Starting buildValidateLoop for file: %s", fileState.filePath)

	originalFile := params.originalFile
	updated := params.updated
	proposedContent := params.proposedContent
	desc := params.desc

	syntaxErrors := params.syntaxErrors
	numAttempts := 0

	for numAttempts < MaxValidationFixAttempts {
		currentAttempt := numAttempts + 1
		log.Printf("Starting validation attempt %d/%d", currentAttempt, MaxValidationFixAttempts)

		// check for context cancellation
		if ctx.Err() != nil {
			log.Printf("Context cancelled during attempt %d", currentAttempt)
			return buildValidateLoopResult{}, ctx.Err()
		}

		// reset retry count for each phase
		fileState.validationNumRetry = 0
		log.Printf("Reset validation retry count for attempt %d", currentAttempt)

		var onStream func(chunk string, buffer string) bool
		if numAttempts == 0 {
			onStream = params.initialPhaseOnStream
			log.Printf("Using initial phase onStream handler")
		} else {
			onStream = nil
			log.Printf("No onStream handler for attempt %d", currentAttempt)
		}

		var reasons []syntax.NeedsVerifyReason
		if numAttempts == 0 {
			reasons = params.reasons
			log.Printf("Using initial reasons for validation")
		} else {
			reasons = []syntax.NeedsVerifyReason{}
			log.Printf("Using empty reasons list for attempt %d", currentAttempt)
		}

		modelConfig := fileState.settings.ModelPack.Builder
		// if available, switch to stronger model after the first attempt failed
		if currentAttempt > 2 && modelConfig.StrongModel != nil {
			log.Printf("Switching to strong model for attempt %d", currentAttempt)
			modelConfig = *modelConfig.StrongModel
		}

		// build validate params
		validateParams := buildValidateParams{
			originalFile:    originalFile,
			updated:         updated,
			proposedContent: proposedContent,
			desc:            desc,
			onStream:        onStream,
			syntaxErrors:    syntaxErrors,
			reasons:         reasons,
			modelConfig:     &modelConfig,
		}

		log.Printf("Calling buildValidate for attempt %d", currentAttempt)
		valid, res, err := fileState.buildValidate(ctx, validateParams)
		if err != nil {
			log.Printf("Error in buildValidate during attempt %d: %v", currentAttempt, err)
			return buildValidateLoopResult{}, fmt.Errorf("error building validate: %v", err)
		}
		updated = res

		if valid {
			log.Printf("Validation succeeded in attempt %d", currentAttempt)
			return buildValidateLoopResult{
				valid:   valid,
				updated: updated,
			}, nil
		}

		log.Printf("Validation failed in attempt %d, preparing for next attempt", currentAttempt)
		syntaxErrors = fileState.validateSyntax(ctx, updated)
		log.Printf("Found %d syntax errors after attempt %d", len(syntaxErrors), currentAttempt)

		numAttempts++
	}

	log.Printf("Validation failed after %d attempts", MaxValidationFixAttempts)
	return buildValidateLoopResult{
		valid:   false,
		updated: updated,
	}, nil
}

type buildValidateParams struct {
	originalFile    string
	updated         string
	proposedContent string
	desc            string
	syntaxErrors    []string
	reasons         []syntax.NeedsVerifyReason
	onStream        func(chunk string, buffer string) bool
	phase           int
	modelConfig     *shared.ModelRoleConfig
}

func (fileState *activeBuildStreamFileState) buildValidate(
	ctx context.Context,
	params buildValidateParams,
) (bool, string, error) {
	log.Printf("Starting buildValidate for phase %d", params.phase)

	auth := fileState.auth
	filePath := fileState.filePath
	clients := fileState.clients
	modelConfig := params.modelConfig

	originalFile := params.originalFile
	updated := params.updated
	proposedContent := params.proposedContent
	desc := params.desc
	onStream := params.onStream
	syntaxErrors := params.syntaxErrors
	reasons := params.reasons

	// Get diff for validation
	log.Printf("Getting diffs between original and updated content")
	diff, err := diff_pkg.GetDiffs(originalFile, updated)
	if err != nil {
		log.Printf("Error getting diffs: %v", err)
		return false, "", fmt.Errorf("error getting diffs: %v", err)
	}

	// Choose prompt and tools based on preferred format

	log.Printf("Building XML validation replacements prompt")
	promptText := prompts.GetValidationReplacementsXmlPrompt(prompts.ValidationPromptParams{
		Path:         filePath,
		Original:     originalFile,
		Desc:         desc,
		Proposed:     proposedContent,
		Diff:         diff,
		SyntaxErrors: syntaxErrors,
		Reasons:      reasons,
	})

	// log.Printf("Prompt to LLM: %s", promptText)

	log.Printf("Creating initial messages for phase 1")
	messages := []types.ExtendedChatMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: promptText,
				},
			},
		},
	}
	reqStarted := time.Now()
	fileState.builderRun.ReplacementStartedAt = reqStarted

	log.Printf("Making validation-replacements model request")
	// log.Printf("Messages: %v", messages)

	// Use ModelRequest for both formats
	res, err := model.ModelRequest(ctx, model.ModelRequestParams{
		Clients:        clients,
		Auth:           auth,
		Plan:           fileState.plan,
		ModelConfig:    modelConfig,
		Purpose:        "File edit",
		Messages:       messages,
		ModelStreamId:  fileState.modelStreamId,
		ConvoMessageId: fileState.convoMessageId,
		BuildId:        fileState.build.Id,
		ModelPackName:  fileState.settings.ModelPack.Name,
		Stop:           []string{"<PlandexFinish/>"},
		BeforeReq: func() {
			log.Printf("Starting model request")
			fileState.builderRun.ReplacementStartedAt = time.Now()
		},
		AfterReq: func() {
			log.Printf("Finished model request")
			fileState.builderRun.ReplacementFinishedAt = time.Now()
		},
		OnStream: onStream,
	})

	if err != nil {
		log.Printf("Error calling model: %v", err)
		return fileState.validationRetryOrError(ctx, params, err)
	}

	fileState.builderRun.GenerationIds = append(fileState.builderRun.GenerationIds, res.GenerationId)
	log.Printf("Added generation ID: %s", res.GenerationId)

	// Handle response based on format
	valid, updated, err := handleXMLResponse(fileState, res.Content, originalFile, updated)

	if err != nil {
		log.Printf("Error handling response: %v", err)
		return fileState.validationRetryOrError(ctx, params, err)
	}

	if valid {
		log.Printf("Validation result: valid=%v", valid)
		return valid, updated, nil
	}

	return false, updated, nil
}

func handleXMLResponse(
	fileState *activeBuildStreamFileState,
	content string,
	originalFile string,
	updated string,
) (bool, string, error) {
	log.Printf("Handling XML response for file: %s", fileState.filePath)

	if strings.Contains(content, "<PlandexCorrect/>") {
		log.Printf("XML response indicates changes are correct")
		fileState.builderRun.ReplacementSuccess = true
		return true, updated, nil
	}
	originalFileLines := strings.Split(originalFile, "\n")

	updated = originalFile

	log.Printf("Processing XML replacement blocks")

	replacementsOuter := utils.GetXMLContent(content, "PlandexReplacements")

	if replacementsOuter == "" {
		log.Printf("No replacements found in XML response")
		return false, updated, fmt.Errorf("no replacements found in XML response")
	}

	replacements := utils.GetAllXMLContent(replacementsOuter, "Replacement")

	for i, replacement := range replacements {
		log.Printf("Processing replacement: %d/%d", i+1, len(replacements))

		old := utils.GetXMLContent(replacement, "Old")
		new := utils.GetXMLContent(replacement, "New")

		if old == "" {
			log.Printf("No old content found for replacement")
			return false, updated, fmt.Errorf("no old content found for replacement")
		}

		old = strings.TrimSpace(old)

		log.Printf("Old content trimmed:\n\n%s", old)

		if !strings.HasPrefix(old, "pdx-") {
			log.Printf("Old content does not have a line number prefix for first line")
			return false, updated, fmt.Errorf("old content does not have a line number prefix for first line")
		}

		oldLines := strings.Split(old, "\n")

		var lastLine string
		var lastLineNum int
		firstLine := oldLines[0]
		if len(oldLines) > 1 {
			lastLine = oldLines[len(oldLines)-1]
		}

		firstLineNum, err := shared.ExtractLineNumberWithPrefix(firstLine, "pdx-")
		if err != nil {
			log.Printf("Error extracting line number from first line: %v", err)
			return false, updated, fmt.Errorf("error extracting line number from first line: %v", err)
		}

		if lastLine != "" {
			lastLineNum, err = shared.ExtractLineNumberWithPrefix(lastLine, "pdx-")
			if err != nil {
				log.Printf("Error extracting line number from last line: %v", err)
				return false, updated, fmt.Errorf("error extracting line number from last line: %v", err)
			}
		}

		if lastLineNum == 0 {
			old = originalFileLines[firstLineNum-1]
		} else {
			old = strings.Join(originalFileLines[firstLineNum-1:lastLineNum], "\n")
		}

		log.Printf("Applying replacement.\n\nOld:\n\n%s\n\nNew:\n\n%s", old, new)

		updated = strings.Replace(updated, old, new, 1)

		log.Printf("Updated content:\n\n%s", updated)
	}

	return false, updated, nil
}

func (fileState *activeBuildStreamFileState) validationRetryOrError(buildCtx context.Context, validateParams buildValidateParams, err error) (bool, string, error) {
	log.Printf("Handling validation error for file: %s", fileState.filePath)
	if fileState.validationNumRetry < MaxBuildErrorRetries {
		fileState.validationNumRetry++

		log.Printf("Retrying validation (attempt %d/%d) due to error: %v",
			fileState.validationNumRetry, MaxBuildErrorRetries, err)

		activePlan := GetActivePlan(fileState.plan.Id, fileState.branch)

		if activePlan == nil {
			log.Printf("Active plan not found for plan ID %s and branch %s",
				fileState.plan.Id, fileState.branch)
			return false, "", fmt.Errorf("active plan not found for plan ID %s and branch %s",
				fileState.plan.Id, fileState.branch)
		}

		select {
		case <-buildCtx.Done():
			log.Printf("Context canceled during retry wait")
			return false, "", context.Canceled
		case <-time.After(time.Duration(fileState.validationNumRetry*fileState.validationNumRetry)*200*time.Millisecond + time.Duration(rand.Intn(500))*time.Millisecond):
			log.Printf("Retry wait completed, attempting validation again")
			break
		}

		return fileState.buildValidate(buildCtx, validateParams)
	} else {
		log.Printf("Max retries (%d) exceeded, returning error", MaxBuildErrorRetries)
		return false, "", err
	}
}
