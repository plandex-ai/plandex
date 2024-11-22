package plan

import (
	"encoding/xml"
	"fmt"
	"log"
	"math/rand"
	"plandex-server/db"
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
		log.Println("buildStructuredEdits - tree-sitter parser is nil")
		fileState.onBuildFileError(fmt.Errorf("tree-sitter parser is nil"))
		return
	}

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		log.Printf("Active plan not found for plan ID %s and branch %s\n", planId, branch)
		fileState.onBuildFileError(fmt.Errorf("active plan not found for plan ID %s and branch %s", planId, branch))
		return
	}

	proposedContentLines := strings.Split(activeBuild.FileContent, "\n")
	originalContentLines := strings.Split(originalFile, "\n")

	log.Println("buildStructuredEdits - getting references prompt")

	anchorsSysPrompt := prompts.GetSemanticAnchorsPrompt(filePath, originalFile, activeBuild.FileContent, activeBuild.FileDescription)

	anchorsFileMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: anchorsSysPrompt,
		},
	}

	promptTokens, err := shared.GetNumTokens(anchorsSysPrompt)

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

	log.Println("buildStructuredEdits - calling model for references")

	modelReq := openai.ChatCompletionRequest{
		Model:       config.BaseModelConfig.ModelName,
		Messages:    anchorsFileMessages,
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
			Purpose:       "Generated file update (structured edits)",
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

	log.Println("buildStructuredEdits - content:")
	log.Println(content)

	anchorsXmlString, err := GetXMLTag(content, "PlandexSemanticAnchors")
	if err != nil {
		log.Printf("buildStructuredEdits - error parsing PlandexSemanticAnchors xml: %v\n", err)
		fileState.structuredEditRetryOrError(fmt.Errorf("error parsing PlandexSemanticAnchors xml xml: %v", err))
		return
	}

	summaryXmlString, err := GetXMLTag(content, "PlandexSummary")
	if err != nil {
		log.Printf("buildStructuredEdits - error parsing PlandexSummary xml: %v\n", err)
	}

	var summaryElement types.SummaryTag
	if summaryXmlString != "" {
		err = xml.Unmarshal([]byte(summaryXmlString), &summaryElement)
		if err != nil {
			log.Printf("buildStructuredEdits - error unmarshalling Summary xml: %v\n", err)
		}
	}

	var anchorsElement types.SemanticAnchorsTag

	err = xml.Unmarshal([]byte(anchorsXmlString), &anchorsElement)
	if err != nil {
		log.Printf("buildStructuredEdits - error unmarshalling xml: %v\n", err)
		fileState.structuredEditRetryOrError(fmt.Errorf("error unmarshalling xml: %v", err))
		return
	}

	log.Printf("buildStructuredEdits - got %d anchors\n", len(anchorsElement.Anchors))

	anchorLines := make(map[int]int)

	for _, anchor := range anchorsElement.Anchors {
		fmt.Printf("anchor: %v\n", anchor)

		var originalLine, proposedLine int
		originalLine, err = shared.ExtractLineNumberWithPrefix(anchor.OriginalLine, "pdx-")
		if err != nil {
			log.Printf("buildStructuredEdits - error parsing anchor original line num: %v\n", err)
			fileState.structuredEditRetryOrError(fmt.Errorf("error parsing anchor original line num: %v", err))
			return
		}

		proposedLine, err = shared.ExtractLineNumberWithPrefix(anchor.ProposedLine, "pdx-new-")
		if err != nil {
			log.Printf("buildStructuredEdits - error parsing anchor proposed line num: %v\n", err)
			fileState.structuredEditRetryOrError(fmt.Errorf("error parsing anchor proposed line num: %v", err))
			return
		}

		proposedContent := proposedContentLines[proposedLine-1]
		originalContent := originalContentLines[originalLine-1]

		if proposedContent != originalContent {
			anchorLines[proposedLine] = originalLine
		}
	}

	fileContent := activeBuild.FileContent
	fileContentLines := strings.Split(fileContent, "\n")

	var references []syntax.Reference
	var removals []syntax.Removal

	var beginsWithRef bool = false
	var endsWithRef bool = false
	var foundNonRefLine bool = false

	for i, line := range fileContentLines {
		line = strings.ToLower(strings.TrimSpace(line))

		if strings.Contains(line, "... existing code ...") {
			references = append(references, syntax.Reference(i+1))
			if !foundNonRefLine {
				beginsWithRef = true
			}
			endsWithRef = true
		} else if strings.Contains(line, "plandex: removed code") {
			removals = append(removals, syntax.Removal(i+1))
		} else if line != "" {
			foundNonRefLine = true
			endsWithRef = false
		}
	}

	if !beginsWithRef &&
		!strings.Contains(activeBuild.FileDescription, "overwrite the entire file") &&
		!strings.Contains(activeBuild.FileDescription, "the start of the file") {

		// structured edits handle normalization of comments, so just use // ... existing code ... here
		fileContentLines = append([]string{"// ... existing code ..."}, fileContentLines...)

		// bump all existing references up by 1
		for i, ref := range references {
			references[i] = syntax.Reference(int(ref) + 1)
		}
		references = append([]syntax.Reference{syntax.Reference(1)}, references...)
	}

	if !endsWithRef &&
		!strings.Contains(activeBuild.FileDescription, "overwrite the entire file") &&
		!strings.Contains(activeBuild.FileDescription, "the end of the file") {
		fileContentLines = append(fileContentLines, "// ... existing code ...")
		references = append(references, syntax.Reference(len(fileContentLines)))
	}

	fileContent = strings.Join(fileContentLines, "\n")

	updatedFile, err := syntax.ApplyChanges(
		activePlan.Ctx,
		fileState.language,
		parser,
		originalFile,
		fileContent,
		references,
		removals,
		anchorLines,
	)
	if err != nil {
		log.Printf("buildStructuredEdits - error applying references: %v\n", err)
		fileState.structuredEditRetryOrError(fmt.Errorf("error applying references: %v", err))
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

	replacements, err := db.GetDiffReplacements(originalFile, updatedFile)
	if err != nil {
		log.Printf("buildStructuredEdits - error getting diff replacements: %v\n", err)
		fileState.structuredEditRetryOrError(fmt.Errorf("error getting diff replacements: %v", err))
		return
	}

	if summaryElement.Content != "" {
		for _, replacement := range replacements {
			replacement.Summary = strings.TrimSpace(summaryElement.Content)
		}
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

	if !fileState.preBuildStateSyntaxInvalid {
		validationRes, err := syntax.Validate(activePlan.Ctx, filePath, updatedFile)
		if err != nil {
			log.Printf("buildStructuredEdits - error validating syntax: %v\n", err)
			fileState.structuredEditRetryOrError(fmt.Errorf("error validating syntax: %v", err))
			return
		}

		if validationRes.HasParser && !validationRes.TimedOut && !validationRes.Valid {
			log.Printf("buildStructuredEdits - syntax is invalid\n")
		}

		res.SyntaxValid = validationRes.Valid
		res.SyntaxErrors = validationRes.Errors
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
