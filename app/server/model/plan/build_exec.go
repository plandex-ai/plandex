package plan

import (
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/model/prompts"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func Build(client *openai.Client, plan *db.Plan, branch string, auth *types.ServerAuth) (int, error) {
	log.Printf("Build: Called with plan ID %s on branch %s\n", plan.Id, branch)
	log.Println("Build: Starting Build operation")

	state := activeBuildStreamState{
		client:        client,
		auth:          auth,
		currentOrgId:  auth.OrgId,
		currentUserId: auth.User.Id,
		plan:          plan,
		branch:        branch,
	}

	active := GetActivePlan(plan.Id, branch)

	onErr := func(err error) (int, error) {
		log.Printf("Build error: %v\n", err)
		active.StreamDoneCh <- nil
		return 0, err
	}

	pendingBuildsByPath, err := state.loadBuild()
	if err != nil {
		return onErr(err)
	}

	if len(pendingBuildsByPath) == 0 {
		log.Println("No pending builds")
		active.StreamDoneCh <- nil
		return 0, nil
	}

	err = db.SetPlanStatus(plan.Id, branch, shared.PlanStatusBuilding, "")

	if err != nil {
		log.Printf("Error setting plan status to building: %v\n", err)
		return onErr(fmt.Errorf("error setting plan status to building: %v", err))
	}

	log.Printf("Starting %d builds\n", len(pendingBuildsByPath))

	for _, pendingBuilds := range pendingBuildsByPath {
		go state.execPlanBuild(pendingBuilds)
	}

	return len(pendingBuildsByPath), nil
}

func (state *activeBuildStreamState) queueBuilds(activeBuilds []*types.ActiveBuild) {
	planId := state.plan.Id
	branch := state.branch

	activePlan := GetActivePlan(planId, branch)
	filePath := activeBuilds[0].Path

	// log.Printf("Queue:")
	// spew.Dump(activePlan.BuildQueuesByPath[filePath])

	UpdateActivePlan(planId, branch, func(active *types.ActivePlan) {
		active.BuildQueuesByPath[filePath] = append(active.BuildQueuesByPath[filePath], activeBuilds...)
	})
	log.Printf("Queued %d build(s) for file %s\n", len(activeBuilds), filePath)

	if activePlan.IsBuildingByPath[filePath] {
		log.Printf("Already building file %s\n", filePath)
		return
	} else {
		log.Printf("Not building file %s, will execute now\n", filePath)
		go state.execPlanBuild(activeBuilds)
	}
}

func (buildState *activeBuildStreamState) execPlanBuild(activeBuilds []*types.ActiveBuild) {
	log.Printf("execPlanBuild for %d active builds\n", len(activeBuilds))

	if len(activeBuilds) == 0 {
		log.Println("No active builds")
		return
	}

	planId := buildState.plan.Id
	branch := buildState.branch

	activePlan := GetActivePlan(planId, branch)
	filePath := activeBuilds[0].Path

	// stream initial status to client
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
		activeBuilds:           activeBuilds,
	}
	err := fileState.loadBuildFile(activeBuilds)
	if err != nil {
		log.Printf("Error loading build file: %v\n", err)
		return
	}

	fileState.buildFile()
}

func (fileState *activeBuildStreamFileState) buildFile() {
	filePath := fileState.filePath
	activeBuilds := fileState.activeBuilds
	planId := fileState.plan.Id
	branch := fileState.branch
	currentPlan := fileState.currentPlanState
	currentOrgId := fileState.currentOrgId
	client := fileState.client
	config := fileState.settings.ModelSet.Builder
	build := fileState.build

	activePlan := GetActivePlan(planId, branch)

	log.Printf("Building file %s\n", filePath)

	log.Println("activePlan.ContextsByPath files:")
	for k := range activePlan.ContextsByPath {
		log.Println(k)
	}

	// get relevant file context (if any)
	contextPart := activePlan.ContextsByPath[filePath]

	var currentState string
	currentPlanFile, fileInCurrentPlan := currentPlan.CurrentPlanFiles.Files[filePath]

	if fileInCurrentPlan {
		log.Printf("File %s found in current plan.\n", filePath)
		if contextPart == nil {
			log.Println("No context - using current plan state.")
			currentState = currentPlanFile
		} else {
			currentFileUpdatedAt := currentPlan.CurrentPlanFiles.UpdatedAtByPath[filePath]
			contextFileUpdatedAt := contextPart.UpdatedAt

			if currentFileUpdatedAt.After(contextFileUpdatedAt) {
				log.Println("Current plan file is newer than context. Using current plan state.")
				currentState = currentPlanFile
			} else {
				log.Println("Context is newer than current plan file. Using context state.")
				currentState = contextPart.Body
			}
		}
	} else if contextPart != nil {
		log.Printf("File %s found in model context. Using context state.\n", filePath)

		currentState = contextPart.Body

		if currentState == "" {
			log.Println("Context state is empty. That's bad.")
		}
	}

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

		// new file
		planRes := &db.PlanFileResult{
			OrgId:           currentOrgId,
			PlanId:          planId,
			PlanBuildId:     build.Id,
			ConvoMessageIds: build.ConvoMessageIds,
			Path:            filePath,
			Content:         activeBuilds[0].FileContent,
		}
		fileState.onFinishBuildFile(planRes)
		return
	}

	fileState.currentState = currentState
	fileState.contextPart = contextPart

	log.Println("Getting file from model: " + filePath)
	// log.Println("File context:", fileContext)

	replacePrompt := prompts.GetReplacePrompt(filePath)
	currentStatePrompt := prompts.GetBuildCurrentStatePrompt(filePath, currentState)
	sysPrompt := prompts.GetBuildSysPrompt(filePath, currentStatePrompt)

	var mergedReply string
	for _, activeBuild := range activeBuilds {
		mergedReply += "\n\n" + activeBuild.ReplyContent
	}

	log.Println("Num active builds: " + fmt.Sprintf("%d", len(activeBuilds)))
	// log.Println("Merged reply:")
	// log.Println(mergedReply)

	fileMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: sysPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: activePlan.Prompt,
		},
		{
			Role:    openai.ChatMessageRoleAssistant,
			Content: mergedReply,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: replacePrompt,
		},
	}

	log.Println("Calling model for file: " + filePath)

	// for _, msg := range fileMessages {
	// 	log.Printf("%s: %s\n", msg.Role, msg.Content)
	// }

	modelReq := openai.ChatCompletionRequest{
		Model: config.BaseModelConfig.ModelName,
		Tools: []openai.Tool{
			{
				Type:     "function",
				Function: prompts.ReplaceFn,
			},
		},
		ToolChoice: openai.ToolChoice{
			Type: "function",
			Function: openai.ToolFunction{
				Name: prompts.ReplaceFn.Name,
			},
		},
		Messages:       fileMessages,
		Temperature:    config.Temperature,
		TopP:           config.TopP,
		ResponseFormat: config.OpenAIResponseFormat,
	}

	stream, err := client.CreateChatCompletionStream(activePlan.Ctx, modelReq)
	if err != nil {
		log.Printf("Error creating plan file stream for path '%s': %v\n", filePath, err)
		return
	}

	go fileState.listenStream(stream)

}
