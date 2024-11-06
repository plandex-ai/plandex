package plan

import (
	"encoding/json"
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/hooks"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/syntax"
	"plandex-server/types"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func Build(
	clients map[string]*openai.Client,
	plan *db.Plan,
	branch string,
	auth *types.ServerAuth,
) (int, error) {
	log.Printf("Build: Called with plan ID %s on branch %s\n", plan.Id, branch)
	log.Println("Build: Starting Build operation")

	state := activeBuildStreamState{
		clients:       clients,
		auth:          auth,
		currentOrgId:  auth.OrgId,
		currentUserId: auth.User.Id,
		plan:          plan,
		branch:        branch,
	}

	streamDone := func() {
		active := GetActivePlan(plan.Id, branch)
		if active != nil {
			active.StreamDoneCh <- nil
		}
	}

	onErr := func(err error) (int, error) {
		log.Printf("Build error: %v\n", err)
		streamDone()
		return 0, err
	}

	pendingBuildsByPath, err := state.loadPendingBuilds()
	if err != nil {
		return onErr(err)
	}

	if len(pendingBuildsByPath) == 0 {
		log.Println("No pending builds")
		streamDone()
		return 0, nil
	}

	err = db.SetPlanStatus(plan.Id, branch, shared.PlanStatusBuilding, "")

	if err != nil {
		log.Printf("Error setting plan status to building: %v\n", err)
		return onErr(fmt.Errorf("error setting plan status to building: %v", err))
	}

	log.Printf("Starting %d builds\n", len(pendingBuildsByPath))

	for _, pendingBuilds := range pendingBuildsByPath {
		go state.queueBuilds(pendingBuilds)
	}

	return len(pendingBuildsByPath), nil
}

func (state *activeBuildStreamState) queueBuilds(activeBuilds []*types.ActiveBuild) {
	planId := state.plan.Id
	branch := state.branch

	queueBuild := func(activeBuild *types.ActiveBuild) {
		filePath := activeBuild.Path

		// log.Printf("Queue:")
		// spew.Dump(activePlan.BuildQueuesByPath[filePath])

		var isBuilding bool

		UpdateActivePlan(planId, branch, func(active *types.ActivePlan) {
			active.BuildQueuesByPath[filePath] = append(active.BuildQueuesByPath[filePath], activeBuilds...)
			isBuilding = active.IsBuildingByPath[filePath]
		})
		log.Printf("Queued %d build(s) for file %s\n", len(activeBuilds), filePath)

		if isBuilding {
			log.Printf("Already building file %s\n", filePath)
			return
		} else {
			log.Printf("Not building file %s\n", filePath)

			active := GetActivePlan(planId, branch)
			if active == nil {
				log.Printf("Active plan not found for plan ID %s and branch %s\n", planId, branch)
				return
			}

			UpdateActivePlan(planId, branch, func(active *types.ActivePlan) {
				active.IsBuildingByPath[filePath] = true
			})

			go state.execPlanBuild(activeBuild)
		}
	}

	for _, activeBuild := range activeBuilds {
		queueBuild(activeBuild)
	}
}

func (buildState *activeBuildStreamState) execPlanBuild(activeBuild *types.ActiveBuild) {
	log.Println("execPlanBuild")

	if activeBuild == nil {
		log.Println("No active build")
		return
	}

	planId := buildState.plan.Id
	branch := buildState.branch

	activePlan := GetActivePlan(planId, branch)
	if activePlan == nil {
		log.Printf("Active plan not found for plan ID %s and branch %s\n", planId, branch)
		return
	}
	filePath := activeBuild.Path

	if !activePlan.IsBuildingByPath[filePath] {
		UpdateActivePlan(activePlan.Id, activePlan.Branch, func(ap *types.ActivePlan) {
			ap.IsBuildingByPath[filePath] = true
		})
	}

	// stream initial status to client
	log.Printf("streaming initial build info for file %s\n", filePath)
	buildInfo := &shared.BuildInfo{
		Path:      filePath,
		NumTokens: 0,
		Finished:  false,
	}
	activePlan.Stream(shared.StreamMessage{
		Type:      shared.StreamMessageBuildInfo,
		BuildInfo: buildInfo,
	})

	fileState := &activeBuildStreamFileState{
		activeBuildStreamState: buildState,
		filePath:               filePath,
		activeBuild:            activeBuild,
	}

	log.Println("execPlanBuild - fileState.loadBuildFile()")
	err := fileState.loadBuildFile(activeBuild)
	if err != nil {
		log.Printf("Error loading build file: %v\n", err)
		return
	}

	if activeBuild.IsVerification {
		log.Println("execPlanBuild - fileState.verifyFileBuild()")
		fileState.verifyFileBuild()
	} else {
		log.Println("execPlanBuild - fileState.buildFile()")
		fileState.buildFile()
	}
}

func (fileState *activeBuildStreamFileState) buildFile() {
	filePath := fileState.filePath
	activeBuild := fileState.activeBuild
	planId := fileState.plan.Id
	branch := fileState.branch
	currentPlan := fileState.currentPlanState
	currentOrgId := fileState.currentOrgId
	build := fileState.build

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		log.Printf("Active plan not found for plan ID %s and branch %s\n", planId, branch)
		return
	}

	log.Printf("Building file %s\n", filePath)

	log.Printf("%d files in context\n", len(activePlan.ContextsByPath))

	// log.Println("activePlan.ContextsByPath files:")
	// for k := range activePlan.ContextsByPath {
	// 	log.Println(k)
	// }

	// get relevant file context (if any)
	contextPart := activePlan.ContextsByPath[filePath]

	var currentState string
	currentPlanFile, fileInCurrentPlan := currentPlan.CurrentPlanFiles.Files[filePath]

	if fileInCurrentPlan {
		log.Printf("File %s found in current plan.\n", filePath)
		currentState = currentPlanFile

		// log.Println("\n\nCurrent state:\n", currentState, "\n\n")

	} else if contextPart != nil {
		log.Printf("File %s found in model context. Using context state.\n", filePath)
		currentState = contextPart.Body

		if currentState == "" {
			log.Println("Context state is empty. That's bad.")
		}

		// log.Println("\n\nCurrent state:\n", currentState, "\n\n")
	}

	fileState.preBuildState = currentState

	if currentState == "" {
		log.Printf("File %s not found in model context or current plan. Creating new file.\n", filePath)

		buildInfo := &shared.BuildInfo{
			Path:      filePath,
			NumTokens: 0,
			Finished:  true,
		}

		activePlan.Stream(shared.StreamMessage{
			Type:      shared.StreamMessageBuildInfo,
			BuildInfo: buildInfo,
		})

		// validate syntax of new file
		validationRes, err := syntax.Validate(activePlan.Ctx, filePath, activeBuild.FileContent)

		if err != nil {
			log.Printf("Error validating syntax for new file '%s': %v\n", filePath, err)
			fileState.onBuildFileError(fmt.Errorf("error validating syntax for new file '%s': %v", filePath, err))
			return
		}

		// new file
		planRes := &db.PlanFileResult{
			OrgId:           currentOrgId,
			PlanId:          planId,
			PlanBuildId:     build.Id,
			ConvoMessageId:  build.ConvoMessageId,
			Path:            filePath,
			Content:         activeBuild.FileContent,
			WillCheckSyntax: validationRes.HasParser && !validationRes.TimedOut,
			SyntaxValid:     validationRes.Valid,
			SyntaxErrors:    validationRes.Errors,
		}

		log.Println("build exec - Plan file result:")
		// spew.Dump(planRes)

		fileState.isNewFile = true

		fileState.onFinishBuildFile(planRes, activeBuild.FileContent)
		return
	} else {
		currentNumTokens, err := shared.GetNumTokens(currentState)

		if err != nil {
			log.Printf("Error getting num tokens for current state: %v\n", err)
			fileState.onBuildFileError(fmt.Errorf("error getting num tokens for current state: %v", err))
			return
		}

		log.Printf("Current state num tokens: %d\n", currentNumTokens)

		activeBuild.CurrentFileTokens = currentNumTokens
	}

	if fileState.parser != nil && !fileState.preBuildStateSyntaxInvalid {
		log.Println("buildFile - building structured edits")

		fileState.buildStructuredEdits()
	} else {
		log.Println("buildFile - building expand references")
		log.Printf("fileState.parser == nil: %v\n", fileState.parser == nil)
		log.Printf("fileState.preBuildStateSyntaxInvalid: %v\n", fileState.preBuildStateSyntaxInvalid)

		fileState.buildExpandReferences()
	}
}

func (fileState *activeBuildStreamFileState) buildFileLineNums() {
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

	log.Println("buildFileLineNums - getting file from model: " + filePath)
	// log.Println("File context:", fileContext)
	// log.Println("currentState:", currentState)

	sysPrompt := prompts.GetBuildLineNumbersSysPrompt(filePath, originalFile, fmt.Sprintf("%s\n\n```%s```", activeBuild.FileDescription, activeBuild.FileContent))

	fileMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: sysPrompt,
		},
	}

	promptTokens, err := shared.GetNumTokens(sysPrompt)

	if err != nil {
		log.Printf("Error getting num tokens for prompt: %v\n", err)
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

	log.Println("buildFileLineNums - calling model for file: " + filePath)

	// for _, msg := range fileMessages {
	// 	log.Printf("%s: %s\n", msg.Role, msg.Content)
	// }

	var responseFormat *openai.ChatCompletionResponseFormat
	if config.BaseModelConfig.HasJsonResponseMode {
		responseFormat = &openai.ChatCompletionResponseFormat{Type: "json_object"}
	}

	// log.Println("responseFormat:", responseFormat)
	// log.Println("Model:", config.BaseModelConfig.ModelName)

	modelReq := openai.ChatCompletionRequest{
		Model: config.BaseModelConfig.ModelName,
		Tools: []openai.Tool{
			{
				Type:     "function",
				Function: &prompts.ListReplacementsFn,
			},
		},
		ToolChoice: openai.ToolChoice{
			Type: "function",
			Function: openai.ToolFunction{
				Name: prompts.ListReplacementsFn.Name,
			},
		},
		Messages:       fileMessages,
		Temperature:    config.Temperature,
		TopP:           config.TopP,
		ResponseFormat: responseFormat,
	}

	envVar := config.BaseModelConfig.ApiKeyEnvVar
	client := clients[envVar]

	if config.BaseModelConfig.HasStreamingFunctionCalls {
		modelReq.StreamOptions = &openai.StreamOptions{
			IncludeUsage: true,
		}

		stream, err := model.CreateChatCompletionStreamWithRetries(client, activePlan.Ctx, modelReq)
		if err != nil {
			log.Printf("Error creating plan file stream for path '%s': %v\n", filePath, err)
			fileState.onBuildFileError(fmt.Errorf("error creating plan file stream for path '%s': %v", filePath, err))
			return
		}

		go fileState.listenStreamChangesWithLineNums(stream)
	} else {

		log.Println("request:")
		log.Println(spew.Sdump(modelReq))

		resp, err := model.CreateChatCompletionWithRetries(client, activePlan.Ctx, modelReq)

		if err != nil {
			log.Printf("Error building file '%s': %v\n", filePath, err)
			fileState.onBuildFileError(fmt.Errorf("error building file '%s': %v", filePath, err))
			return
		}

		log.Println("buildFileLineNums - usage:")
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

		var s string
		var res types.ChangesWithLineNums

		for _, choice := range resp.Choices {
			if len(choice.Message.ToolCalls) == 1 &&
				choice.Message.ToolCalls[0].Function.Name == prompts.ListReplacementsFn.Name {
				fnCall := choice.Message.ToolCalls[0].Function
				s = fnCall.Arguments
				break
			}
		}

		if s == "" {
			log.Println("no ListReplacements function call found in response")
			fileState.lineNumsRetryOrError(fmt.Errorf("no ListReplacements function call found in response"))
			return
		}

		bytes := []byte(s)

		err = json.Unmarshal(bytes, &res)
		if err != nil {
			log.Printf("Error unmarshalling build response: %v\n", err)
			fileState.lineNumsRetryOrError(fmt.Errorf("error unmarshalling build response: %v", err))
			return
		}

		fileState.onLineNumsBuildResult(res)
	}
}
