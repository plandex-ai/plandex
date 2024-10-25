package plan

import (
	"encoding/xml"
	"fmt"
	"log"
	"math/rand"
	"plandex-server/db"
	"plandex-server/external/diff"
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

type Reference struct {
	Comment       string `xml:"comment,attr"`
	Description   string `xml:"description,attr"`
	ProposedLine  string `xml:"proposedLine,attr"`
	OriginalStart string `xml:"originalStart,attr"`
	OriginalEnd   string `xml:"originalEnd,attr"`
}

type References struct {
	XMLName    xml.Name    `xml:"references"`
	References []Reference `xml:"reference"`
}

const ExpandRefsRetries = 1

func (fileState *activeBuildStreamFileState) buildExpandReferences() {
	auth := fileState.auth
	filePath := fileState.filePath
	activeBuild := fileState.activeBuild
	clients := fileState.clients
	planId := fileState.plan.Id
	branch := fileState.branch
	config := fileState.settings.ModelPack.Builder
	originalFile := fileState.preBuildState

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		log.Printf("Active plan not found for plan ID %s and branch %s\n", planId, branch)
		return
	}

	log.Println("buildExpandReferences - getting references prompt")

	refsSysPrompt := prompts.GetReferencesPrompt(filePath, originalFile, activeBuild.FileContent, activeBuild.FileDescription)

	refsFileMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: refsSysPrompt,
		},
	}

	promptTokens, err := shared.GetNumTokens(refsSysPrompt)

	if err != nil {
		log.Printf("buildExpandReferences - error getting num tokens for prompt: %v\n", err)
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

	log.Println("buildExpandReferences - calling model for references")

	modelReq := openai.ChatCompletionRequest{
		Model:       config.BaseModelConfig.ModelName,
		Messages:    refsFileMessages,
		Temperature: config.Temperature,
		TopP:        config.TopP,
	}

	envVar := config.BaseModelConfig.ApiKeyEnvVar
	client := clients[envVar]

	resp, err := model.CreateChatCompletionWithRetries(client, activePlan.Ctx, modelReq)

	if err != nil {
		log.Printf("buildExpandReferences - error calling model: %v\n", err)
		fileState.expandRefsRetryOrError(fmt.Errorf("error calling model: %v", err))
		return
	}

	log.Println("buildExpandReferences - usage:")
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
			Purpose:       "Generated file update (ref expansion)",
		},
	})

	if apiErr != nil {
		activePlan.StreamDoneCh <- apiErr
		return
	}

	if len(resp.Choices) == 0 {
		log.Printf("buildExpandReferences - no choices in response\n")
		fileState.expandRefsRetryOrError(fmt.Errorf("no choices in response"))
		return
	}

	refsChoice := resp.Choices[0]
	refsXml := refsChoice.Message.Content

	refsSplit := strings.Split(refsXml, "<references>")
	if len(refsSplit) != 2 {
		log.Printf("buildExpandReferences - error processing references xml\n")
		fileState.expandRefsRetryOrError(fmt.Errorf("error processing references xml"))
		return
	}

	processedXml := "<references>" + refsSplit[1]
	processedXml = EscapeInvalidXMLAttributeCharacters(processedXml)
	refsXml = processedXml

	var refs References

	err = xml.Unmarshal([]byte(processedXml), &refs)
	if err != nil {
		log.Printf("buildExpandReferences - error unmarshalling xml: %v\n", err)
		fileState.expandRefsRetryOrError(fmt.Errorf("error unmarshalling xml: %v", err))
		return
	}

	log.Printf("buildExpandReferences - got %d references\n", len(refs.References))

	fileContentLines := strings.Split(activeBuild.FileContent, "\n")
	inputTextLines := strings.Split(originalFile, "\n")

	for _, ref := range refs.References {
		var proposedLine, originalStart, originalEnd int

		proposedLine, err = shared.ExtractLineNumberWithPrefix(ref.ProposedLine, "pdx-new-")
		if err != nil {
			log.Printf("Error extracting line number from proposed line: %v", err)
			fileState.expandRefsRetryOrError(fmt.Errorf("error extracting line number from proposed line: %v", err))
			return
		}

		originalStart, err = shared.ExtractLineNumberWithPrefix(ref.OriginalStart, "pdx-")
		if err != nil {
			log.Printf("Error extracting line number from start: %v", err)
			fileState.expandRefsRetryOrError(fmt.Errorf("error extracting line number from start: %v", err))
			return
		}
		originalEnd, err = shared.ExtractLineNumberWithPrefix(ref.OriginalEnd, "pdx-")
		if err != nil {
			log.Printf("Error extracting line number from end: %v", err)
			fileState.expandRefsRetryOrError(fmt.Errorf("error extracting line number from end: %v", err))
			return
		}

		if originalStart > originalEnd {
			originalStart, originalEnd = originalEnd, originalStart
		}

		if originalEnd > len(inputTextLines) {
			log.Printf("Start line is greater than end line or end line is greater than the number of lines in the file: %d > %d", originalStart, originalEnd)
			fileState.expandRefsRetryOrError(fmt.Errorf("start line is greater than end line or end line is greater than the number of lines in the file: %d > %d", originalStart, originalEnd))
			return
		}

		// Replace the proposed line with the referenced section from inputTextLines
		referencedSection := inputTextLines[originalStart-1 : originalEnd]

		if proposedLine-1 >= 0 && proposedLine-1 < len(fileContentLines) {
			fileContentLines[proposedLine-1] = strings.Join(referencedSection, "\n")
		} else {
			log.Printf("Proposed line is out of bounds: %d", proposedLine)
			fileState.expandRefsRetryOrError(fmt.Errorf("proposed line is out of bounds: %d", proposedLine))
			return
		}
	}

	updatedFile := strings.Join(fileContentLines, "\n")

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

	edits := diff.Strings(originalFile, updatedFile)
	replacements := []*shared.Replacement{}
	for _, edit := range edits {
		old := originalFile[edit.Start:edit.End]
		replacements = append(replacements, &shared.Replacement{
			Old: old,
			New: edit.New,
		})
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
		CanVerify:      true,
	}

	validationRes, err := syntax.Validate(activePlan.Ctx, filePath, updatedFile)
	if err != nil {
		log.Printf("buildExpandReferences - error validating syntax: %v\n", err)
		fileState.expandRefsRetryOrError(fmt.Errorf("error validating syntax: %v", err))
		return
	}

	res.WillCheckSyntax = validationRes.HasParser && !validationRes.TimedOut
	res.SyntaxValid = validationRes.Valid
	res.SyntaxErrors = validationRes.Errors

	fileState.onFinishBuildFile(&res, updatedFile)
}

func (fileState *activeBuildStreamFileState) expandRefsRetryOrError(err error) {
	if fileState.expandRefsNumRetry < ExpandRefsRetries {
		fileState.expandRefsNumRetry++

		log.Printf("buildExpandReferences - retrying expand refs file '%s' due to error: %v\n", fileState.filePath, err)

		activePlan := GetActivePlan(fileState.plan.Id, fileState.branch)

		if activePlan == nil {
			log.Printf("buildExpandReferences - active plan not found for plan ID %s and branch %s\n", fileState.plan.Id, fileState.branch)
			return
		}

		select {
		case <-activePlan.Ctx.Done():
			log.Printf("buildExpandReferences - context canceled\n")
			return
		case <-time.After(time.Duration(fileState.expandRefsNumRetry*fileState.expandRefsNumRetry)*200*time.Millisecond + time.Duration(rand.Intn(500))*time.Millisecond):
			break
		}

		fileState.buildExpandReferences()
	} else {
		fileState.onBuildFileError(err)
	}

}
