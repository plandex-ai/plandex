package plan

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/types"
	"regexp"
	"strings"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

type formatModelContextParams struct {
	includeMaps         bool
	smartContextEnabled bool
	execEnabled         bool
	basicOnly           bool
	cache               bool
	activeOnly          bool
	autoOnly            bool
	activatedPaths      map[string]bool
}

func (state *activeTellStreamState) formatModelContext(params formatModelContextParams) *types.ExtendedChatMessagePart {
	log.Println("Tell plan - formatModelContext")

	includeMaps := params.includeMaps
	smartContextEnabled := params.smartContextEnabled
	execEnabled := params.execEnabled
	currentStage := state.currentStage

	basicOnly := params.basicOnly
	activeOnly := params.activeOnly
	autoOnly := params.autoOnly
	activatedPaths := params.activatedPaths

	if activatedPaths == nil {
		activatedPaths = map[string]bool{}
	}

	var contextMessages []string = []string{
		"### LATEST PLAN CONTEXT ###",
	}
	addedFilesSet := map[string]bool{}

	uses := map[string]bool{}
	if currentStage.TellStage == shared.TellStageImplementation && smartContextEnabled && state.currentSubtask != nil {
		log.Println("Tell plan - formatModelContext - implementation stage - smart context enabled for current subtask")
		for _, path := range state.currentSubtask.UsesFiles {
			uses[path] = true
		}
		log.Printf("Tell plan - formatModelContext - uses: %v\n", uses)
	}

	for _, part := range state.modelContext {
		if basicOnly && part.AutoLoaded {
			continue
		}

		if autoOnly && !part.AutoLoaded {
			continue
		}

		if currentStage.TellStage == shared.TellStageImplementation && smartContextEnabled && state.currentSubtask != nil && part.ContextType == shared.ContextFileType && !uses[part.FilePath] {
			continue
		}

		if activeOnly && !activatedPaths[part.FilePath] {
			continue
		}

		var message string
		var fmtStr string
		var args []any

		if part.ContextType == shared.ContextDirectoryTreeType {
			fmtStr = "\n\n- %s | directory tree:\n\n```\n%s\n```"
			args = append(args, part.FilePath, part.Body)
		} else if part.ContextType == shared.ContextFileType {
			fmtStr = "\n\n- %s:\n\n```\n%s\n```"

			var body string
			var found bool
			if state.currentPlanState.CurrentPlanFiles != nil {
				res, ok := state.currentPlanState.CurrentPlanFiles.Files[part.FilePath]
				if ok {
					body = res
					found = true
				}
			}
			if !found {
				body = part.Body
			}
			addedFilesSet[part.FilePath] = true

			args = append(args, part.FilePath, body)
		} else if part.ContextType == shared.ContextMapType {
			if !includeMaps {
				continue
			}
			fmtStr = "\n\n- %s | map:\n\n```\n%s\n```"
			args = append(args, part.FilePath, part.Body)
		} else if part.Url != "" {
			fmtStr = "\n\n- %s:\n\n```\n%s\n```"
			args = append(args, part.Url, part.Body)
		} else if part.ContextType != shared.ContextImageType {
			fmtStr = "\n\n- content%s:\n\n```\n%s\n```"
			args = append(args, part.Name, part.Body)
		}

		if part.ContextType != shared.ContextImageType {
			message = fmt.Sprintf(fmtStr, args...)
			contextMessages = append(contextMessages, message)
		}

		log.Printf("Tell plan - formatModelContext - added context - %s\n", part.Name)
	}

	// Add any current files in plan that weren't added to the context
	for filePath, body := range state.currentPlanState.CurrentPlanFiles.Files {
		if !addedFilesSet[filePath] {

			if currentStage.TellStage == shared.TellStageImplementation && smartContextEnabled && !uses[filePath] {
				continue
			}

			if filePath == "_apply.sh" {
				continue
			}

			if activeOnly && !activatedPaths[filePath] {
				continue
			}

			contextMessages = append(contextMessages, fmt.Sprintf("\n\n- %s:\n\n```\n%s\n```", filePath, body))

			log.Printf("Tell plan - formatModelContext - added current plan file - %s\n", filePath)
		}
	}

	if len(state.currentPlanState.CurrentPlanFiles.Removed) > 0 {
		contextMessages = append(contextMessages, "*Removed files:*\n")
		for path := range state.currentPlanState.CurrentPlanFiles.Removed {
			contextMessages = append(contextMessages, fmt.Sprintf("- %s", path))
		}
		contextMessages = append(contextMessages, "These files have been *removed* and are no longer in the plan. If you want to re-add them to the plan, you must explicitly create them again.")

		log.Println("Tell plan - formatModelContext - added removed files")
		log.Println(contextMessages)
	}

	if execEnabled &&
		// don't show _apply.sh history and content if smart context is enabled and the current subtask doesn't use it
		!(currentStage.TellStage == shared.TellStageImplementation && smartContextEnabled && state.currentSubtask != nil && !uses["_apply.sh"]) {

		execHistory := state.currentPlanState.ExecHistory()

		contextMessages = append(contextMessages, execHistory)

		scriptContent, ok := state.currentPlanState.CurrentPlanFiles.Files["_apply.sh"]
		var isEmpty bool
		if !ok || scriptContent == "" {
			scriptContent = "[empty]"
			isEmpty = true
		}

		contextMessages = append(contextMessages, "*Current* state of _apply.sh script:")
		contextMessages = append(contextMessages, fmt.Sprintf("\n\n- _apply.sh:\n\n```\n%s\n```", scriptContent))

		if isEmpty && currentStage.TellStage == shared.TellStagePlanning && currentStage.PlanningPhase != shared.PlanningPhaseContext {
			contextMessages = append(contextMessages, "The _apply.sh script is *empty*. You ABSOLUTELY MUST include a '### Commands' section in your response prior to the '### Tasks' section that evaluates whether any commands should be written to _apply.sh during the plan. This is MANDATORY. Do NOT UNDER ANY CIRCUMSTANCES omit this section. If you determine that commands should be added or updated in _apply.sh, you MUST also create a subtask referencing _apply.sh in the '### Tasks' section.")

			if execHistory != "" {
				contextMessages = append(contextMessages, "Consider the history of previously executed _apply.sh scripts when determining which commands to include in the new _apply.sh file. Are there any commands that should be run again after code changes? If so, mention them in the '### Commands' section and then include a subtask to include them in the _apply.sh file in the '### Tasks' section.")
			}
		}
	}

	s := strings.Join(contextMessages, "\n\n") + "\n\n### END OF CONTEXT ###\n\n"

	res := &types.ExtendedChatMessagePart{
		Type: openai.ChatMessagePartTypeText,
		Text: s,
	}

	if params.cache {
		res.CacheControl = &types.CacheControlSpec{
			Type: types.CacheControlTypeEphemeral,
		}
	}

	return res
}

var pathRegex = regexp.MustCompile("`(.+?)`")

type checkAutoLoadContextResult struct {
	autoLoadPaths []string
	activatePaths map[string]bool
}

func (state *activeTellStreamState) checkAutoLoadContext() checkAutoLoadContextResult {
	req := state.req
	activePlan := state.activePlan
	contextsByPath := activePlan.ContextsByPath
	currentStage := state.currentStage

	// can only auto load context in planning stage
	// context phase is primary loading phase
	// planning phase can still load additional context files as a backup
	if currentStage.TellStage != shared.TellStagePlanning {
		return checkAutoLoadContextResult{}
	}

	log.Printf("%d existing contexts by path\n", len(contextsByPath))

	// pick out all potential file paths within backticks
	matches := pathRegex.FindAllStringSubmatch(activePlan.CurrentReplyContent, -1)

	toAutoLoad := []string{}
	toActivate := map[string]bool{}

	for _, match := range matches {
		trimmed := strings.TrimSpace(match[1])
		if trimmed == "" {
			continue
		}

		if req.ProjectPaths[trimmed] {
			toActivate[trimmed] = true
			if contextsByPath[trimmed] == nil {
				toAutoLoad = append(toAutoLoad, trimmed)
			}
		}
	}

	log.Printf("Tell plan - checkAutoLoadContext - toAutoLoad: %v\n", toAutoLoad)
	log.Printf("Tell plan - checkAutoLoadContext - toActivate: %v\n", toActivate)

	return checkAutoLoadContextResult{
		autoLoadPaths: toAutoLoad,
		activatePaths: toActivate,
	}
}

func (state *activeTellStreamState) addImageContext() (int, bool) {
	active := state.activePlan

	var imageContextTokens int

	for _, context := range state.modelContext {
		if context.ContextType == shared.ContextImageType {
			if !state.settings.ModelPack.Planner.BaseModelConfig.HasImageSupport {
				err := fmt.Errorf("%s does not support images in context", state.settings.ModelPack.Planner.BaseModelConfig.ModelName)
				log.Println(err)
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusBadRequest,
					Msg:    "Model does not support images in context",
				}
				return 0, false
			}

			imageContextTokens += context.NumTokens

			imageMessage := types.ExtendedChatMessage{
				Role: openai.ChatMessageRoleUser,
				Content: []types.ExtendedChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: fmt.Sprintf("Image: %s", context.Name),
					},
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL:    shared.GetImageDataURI(context.Body, context.FilePath),
							Detail: context.ImageDetail,
						},
					},
				},
			}
			state.messages = append(state.messages, imageMessage)
		}
	}

	return imageContextTokens, true
}
