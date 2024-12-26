package plan

import (
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/types"

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

	log.Println("execPlanBuild - fileState.buildFile()")
	fileState.buildFile()
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

	// log.Println("plan files:")
	// spew.Dump(currentPlan.CurrentPlanFiles.Files)

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

	if activeBuild.IsMoveOp {
		log.Printf("File %s is a move operation. Moving to %s\n", filePath, activeBuild.MoveDestination)
		log.Println("Will remove this path and then queue another build for the new path with the current file's content")

		fileState.queueBuilds([]*types.ActiveBuild{
			{
				ReplyId:    activeBuild.ReplyId,
				Idx:        activeBuild.Idx,
				Path:       activeBuild.Path,
				IsRemoveOp: true,
			},
			{
				ReplyId:           activeBuild.ReplyId,
				Idx:               activeBuild.Idx,
				Path:              activeBuild.MoveDestination,
				FileContent:       currentState,
				FileContentTokens: 0,
			},
		})

		return
	}

	if activeBuild.IsRemoveOp {
		log.Printf("File %s is a remove operation. Removing file.\n", filePath)

		buildInfo := &shared.BuildInfo{
			Path:      filePath,
			NumTokens: 0,
			Removed:   true,
			Finished:  true,
		}

		activePlan.Stream(shared.StreamMessage{
			Type:      shared.StreamMessageBuildInfo,
			BuildInfo: buildInfo,
		})

		planRes := &db.PlanFileResult{
			OrgId:          currentOrgId,
			PlanId:         planId,
			PlanBuildId:    build.Id,
			ConvoMessageId: build.ConvoMessageId,
			Path:           filePath,
			Content:        "",
			RemovedFile:    true,
		}
		fileState.onFinishBuildFile(planRes, "")
		return
	}

	if activeBuild.IsResetOp {
		log.Printf("File %s is a reset operation. Resetting file.\n", filePath)

		if contextPart == nil {
			log.Printf("File %s not found in model context. Removing pending file.\n", filePath)

			fileState.queueBuilds([]*types.ActiveBuild{
				{
					ReplyId:    activeBuild.ReplyId,
					Idx:        activeBuild.Idx,
					Path:       activeBuild.Path,
					IsRemoveOp: true,
				},
			})

			return
		} else {
			log.Printf("File %s found in model context. Using context state.\n", filePath)

			buildInfo := &shared.BuildInfo{
				Path:      filePath,
				NumTokens: 0,
				Finished:  true,
			}

			activePlan.Stream(shared.StreamMessage{
				Type:      shared.StreamMessageBuildInfo,
				BuildInfo: buildInfo,
			})

			planRes := &db.PlanFileResult{
				OrgId:          currentOrgId,
				PlanId:         planId,
				PlanBuildId:    build.Id,
				ConvoMessageId: build.ConvoMessageId,
				Path:           filePath,
				Content:        contextPart.Body,
			}
			fileState.onFinishBuildFile(planRes, "")
			return
		}
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
		// validationRes, err := syntax.Validate(activePlan.Ctx, filePath, activeBuild.FileContent)

		// if err != nil {
		// 	log.Printf("Error validating syntax for new file '%s': %v\n", filePath, err)
		// 	fileState.onBuildFileError(fmt.Errorf("error validating syntax for new file '%s': %v", filePath, err))
		// 	return
		// }

		// new file
		planRes := &db.PlanFileResult{
			OrgId:          currentOrgId,
			PlanId:         planId,
			PlanBuildId:    build.Id,
			ConvoMessageId: build.ConvoMessageId,
			Path:           filePath,
			Content:        activeBuild.FileContent,
			// WillCheckSyntax: validationRes.HasParser && !validationRes.TimedOut,
			// SyntaxValid:     validationRes.Valid,
			// SyntaxErrors:    validationRes.Errors,
		}

		// log.Println("build exec - new file result")
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
		activePlan.DidEditFiles = true
	}

	// build structured edits strategy now works regardless of language/tree-sitter support
	log.Println("buildFile - building structured edits")
	fileState.buildStructuredEdits()
}
