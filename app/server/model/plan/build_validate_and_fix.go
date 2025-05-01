package plan

import (
	"context"
	"errors"
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

const MaxValidationFixAttempts = 3

type buildValidateLoopParams struct {
	originalFile               string
	updated                    string
	proposedContent            string
	desc                       string
	syntaxErrors               []string
	reasons                    []syntax.NeedsVerifyReason
	initialPhaseOnStream       func(chunk string, buffer string) bool
	validateOnlyOnFinalAttempt bool
	maxAttempts                int
	isInitial                  bool
	sessionId                  string
}

type buildValidateLoopResult struct {
	valid   bool
	updated string
	problem string
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

	problems := []string{}

	maxAttempts := MaxValidationFixAttempts
	if params.maxAttempts > 0 {
		maxAttempts = params.maxAttempts
	}

	for numAttempts < maxAttempts {
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

		isLastAttempt := numAttempts == maxAttempts-1

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
			validateOnly:    isLastAttempt && params.validateOnlyOnFinalAttempt,
			phase:           currentAttempt,
			isInitial:       params.isInitial,
			sessionId:       params.sessionId,
		}

		log.Printf("Calling buildValidate for attempt %d", currentAttempt)
		res, err := fileState.buildValidate(ctx, validateParams)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Printf("Context canceled during buildValidate")
				return buildValidateLoopResult{}, err
			}

			log.Printf("Error in buildValidate during attempt %d: %v", currentAttempt, err)
			return buildValidateLoopResult{}, fmt.Errorf("error building validate: %v", err)
		}
		updated = res.updated

		syntaxErrors = fileState.validateSyntax(ctx, updated)
		log.Printf("Found %d syntax errors after attempt %d", len(syntaxErrors), currentAttempt)

		if res.valid && len(syntaxErrors) == 0 {
			log.Printf("Validation succeeded in attempt %d", currentAttempt)
			return buildValidateLoopResult{
				valid:   res.valid,
				updated: res.updated,
			}, nil
		}

		problems = append(problems, res.problem)

		log.Printf("Validation failed in attempt %d, preparing for next attempt", currentAttempt)

		numAttempts++
	}

	log.Printf("Validation failed after %d attempts", MaxValidationFixAttempts)
	return buildValidateLoopResult{
		valid:   false,
		updated: updated,
		problem: strings.Join(problems, "\n\n"),
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
	validateOnly    bool
	isInitial       bool
	sessionId       string
}

type buildValidateResult struct {
	valid   bool
	updated string
	problem string
}

func (fileState *activeBuildStreamFileState) buildValidate(
	ctx context.Context,
	params buildValidateParams,
) (buildValidateResult, error) {
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
		return buildValidateResult{}, fmt.Errorf("error getting diffs: %v", err)
	}

	originalWithLineNums := shared.AddLineNums(originalFile)
	proposedWithLineNums := shared.AddLineNums(proposedContent)

	maxExpectedOutputTokens := shared.GetNumTokensEstimate(originalFile)/2 + shared.GetNumTokensEstimate(proposedContent)

	// Choose prompt and tools based on preferred format

	log.Printf("Building XML validation replacements prompt")
	promptText, headNumTokens := prompts.GetValidationReplacementsXmlPrompt(prompts.ValidationPromptParams{
		Path:                 filePath,
		OriginalWithLineNums: originalWithLineNums,
		Desc:                 desc,
		ProposedWithLineNums: proposedWithLineNums,
		Diff:                 diff,
		SyntaxErrors:         syntaxErrors,
		Reasons:              reasons,
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

	if params.validateOnly {
		log.Printf("Making validation-only model request")
	} else {
		log.Printf("Making validation-replacements model request")
	}
	// log.Printf("Messages: %v", messages)

	stop := []string{"<PlandexFinish/>"}
	if params.validateOnly {
		stop = []string{"<PlandexComments>", "<PlandexReplacements>"}
	}

	var willCacheNumTokens int
	isFirstPass := params.isInitial && params.phase == 1
	if !isFirstPass && modelConfig.BaseModelConfig.Provider == shared.ModelProviderOpenAI {
		willCacheNumTokens = headNumTokens
	}

	log.Printf("buildValidate - calling model.ModelRequest")
	// spew.Dump(messages)

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
		Stop:           stop,
		BeforeReq: func() {
			log.Printf("Starting model request")
			fileState.builderRun.ReplacementStartedAt = time.Now()
		},
		AfterReq: func() {
			log.Printf("Finished model request")
			fileState.builderRun.ReplacementFinishedAt = time.Now()
		},
		OnStream: onStream,

		WillCacheNumTokens:    willCacheNumTokens,
		SessionId:             params.sessionId,
		EstimatedOutputTokens: maxExpectedOutputTokens,
	})

	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Printf("Context canceled during model request")
			return buildValidateResult{}, err
		}

		log.Printf("Error calling model: %v", err)
		return fileState.validationRetryOrError(ctx, params, err)
	}

	// log.Printf("Model response:\n\n%s", res.Content)

	fileState.builderRun.GenerationIds = append(fileState.builderRun.GenerationIds, res.GenerationId)
	log.Printf("Added generation ID: %s", res.GenerationId)

	// Handle response based on format
	parseRes, err := handleXMLResponse(fileState, res.Content, originalWithLineNums, updated, params.validateOnly)

	if err != nil {
		log.Printf("Error handling response: %v", err)
		return fileState.validationRetryOrError(ctx, params, err)
	}

	log.Printf("Validation result: valid=%v", parseRes.valid)

	return parseRes, nil
}

func handleXMLResponse(
	fileState *activeBuildStreamFileState,
	content string,
	originalWithLineNums shared.LineNumberedTextType,
	updated string,
	validateOnly bool,
) (buildValidateResult, error) {
	log.Printf("Handling XML response for file: %s", fileState.filePath)

	if strings.Contains(content, "<PlandexCorrect/>") {
		log.Printf("XML response indicates changes are correct")
		fileState.builderRun.ReplacementSuccess = true
		return buildValidateResult{
			valid:   true,
			updated: updated,
		}, nil
	}

	if validateOnly {
		log.Printf("Validation-only mode, skipping replacements")
		return buildValidateResult{
			valid:   false,
			updated: updated,
		}, nil
	}

	originalFileLines := strings.Split(string(originalWithLineNums), "\n")

	incremental := originalWithLineNums

	log.Printf("Processing XML replacement blocks")

	replacementsOuter := utils.GetXMLContent(content, "PlandexReplacements")

	if replacementsOuter == "" {
		log.Printf("No replacements found in XML response")
		return buildValidateResult{
			valid:   false,
			updated: shared.RemoveLineNums(incremental),
			problem: "No replacements found in XML response",
		}, nil
	}

	replacements := utils.GetAllXMLContent(replacementsOuter, "Replacement")

	for i, replacement := range replacements {
		log.Printf("Processing replacement: %d/%d", i+1, len(replacements))

		old := utils.GetXMLContent(replacement, "Old")
		new := utils.GetXMLContent(replacement, "New")

		if old == "" {
			log.Printf("No old content found for replacement")
			return buildValidateResult{valid: false, updated: updated}, fmt.Errorf("no old content found for replacement")
		}

		old = strings.TrimSpace(old)

		// log.Printf("Old content trimmed:\n\n%s", strconv.Quote(old))

		// log.Printf("New content:\n\n%s", strconv.Quote(new))

		if !strings.HasPrefix(old, "pdx-") {
			log.Printf("Old content does not have a line number prefix for first line")
			return buildValidateResult{valid: false, updated: updated}, fmt.Errorf("old content does not have a line number prefix for first line")
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
			return buildValidateResult{valid: false, updated: updated}, fmt.Errorf("error extracting line number from first line: %v", err)
		}

		if lastLine != "" {
			lastLineNum, err = shared.ExtractLineNumberWithPrefix(lastLine, "pdx-")
			if err != nil {
				log.Printf("Error extracting line number from last line: %v", err)
				return buildValidateResult{valid: false, updated: updated}, fmt.Errorf("error extracting line number from last line: %v", err)
			}
		}

		if lastLineNum == 0 {
			if !(firstLineNum > 0 && firstLineNum <= len(originalFileLines)) {
				log.Printf("Invalid line number for first line: %d", firstLineNum)
				return buildValidateResult{valid: false, updated: updated}, fmt.Errorf("invalid line number for first line: %d", firstLineNum)
			}
			old = originalFileLines[firstLineNum-1]
		} else {
			if !(firstLineNum > 0 && firstLineNum <= len(originalFileLines) && lastLineNum > firstLineNum && lastLineNum <= len(originalFileLines)) {
				log.Printf("Invalid line numbers for first and last lines: %d-%d", firstLineNum, lastLineNum)
				return buildValidateResult{valid: false, updated: updated}, fmt.Errorf("invalid line numbers: %d-%d", firstLineNum, lastLineNum)
			}
			old = strings.Join(originalFileLines[firstLineNum-1:lastLineNum], "\n")
		}

		// log.Printf("Applying replacement.\n\nOld:\n\n%s\n\nNew:\n\n%s", old, new)

		incremental = shared.LineNumberedTextType(strings.Replace(string(incremental), old, new, 1))

		// log.Printf("Updated content:\n\n%s", string(incremental))
	}

	var problem string

	if strings.Contains(content, "<PlandexIncorrect/>") {
		split := strings.Split(content, "<PlandexIncorrect/>")
		problem = split[0]
	} else if strings.Contains(content, "<PlandexReplacements>") {
		split := strings.Split(content, "<PlandexReplacements>")
		problem = split[0]
	}

	final := shared.RemoveLineNums(incremental)

	// log.Printf("Final content:\n\n%s", final)

	return buildValidateResult{valid: false, updated: final, problem: problem}, nil
}

func (fileState *activeBuildStreamFileState) validationRetryOrError(buildCtx context.Context, validateParams buildValidateParams, err error) (buildValidateResult, error) {
	log.Printf("Handling validation error for file: %s", fileState.filePath)
	if fileState.validationNumRetry < MaxBuildErrorRetries {
		fileState.validationNumRetry++

		log.Printf("Retrying validation (attempt %d/%d) due to error: %v",
			fileState.validationNumRetry, MaxBuildErrorRetries, err)

		activePlan := GetActivePlan(fileState.plan.Id, fileState.branch)

		if activePlan == nil {
			log.Printf("Active plan not found for plan ID %s and branch %s",
				fileState.plan.Id, fileState.branch)
			return buildValidateResult{}, fmt.Errorf("active plan not found for plan ID %s and branch %s",
				fileState.plan.Id, fileState.branch)
		}

		select {
		case <-buildCtx.Done():
			log.Printf("Context canceled during retry wait")
			return buildValidateResult{}, context.Canceled
		case <-time.After(time.Duration(fileState.validationNumRetry*fileState.validationNumRetry)*200*time.Millisecond + time.Duration(rand.Intn(500))*time.Millisecond):
			log.Printf("Retry wait completed, attempting validation again")
			break
		}

		return fileState.buildValidate(buildCtx, validateParams)
	} else {
		log.Printf("Max retries (%d) exceeded, returning error", MaxBuildErrorRetries)
		return buildValidateResult{}, err
	}
}
