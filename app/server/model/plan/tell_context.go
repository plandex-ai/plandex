package plan

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) formatModelContext(includeMaps, includeTrees, isImplementationStage, smartContextEnabled, execEnabled bool) (string, error) {
	log.Println("Tell plan - formatModelContext")

	var contextMessages []string = []string{
		"### LATEST PLAN CONTEXT ###",
	}
	addedFilesSet := map[string]bool{}

	uses := map[string]bool{}
	if isImplementationStage && smartContextEnabled && state.currentSubtask != nil {
		log.Println("Tell plan - formatModelContext - implementation stage - smart context enabled for current subtask")
		for _, path := range state.currentSubtask.UsesFiles {
			uses[path] = true
		}
		log.Printf("Tell plan - formatModelContext - uses: %v\n", uses)
	}

	for _, part := range state.modelContext {
		if isImplementationStage && smartContextEnabled && state.currentSubtask != nil && part.ContextType == shared.ContextFileType && !uses[part.FilePath] {
			continue
		}

		var message string
		var fmtStr string
		var args []any

		if part.ContextType == shared.ContextDirectoryTreeType {
			if !includeTrees {
				continue
			}
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

			if isImplementationStage && smartContextEnabled && !uses[filePath] {
				continue
			}

			if filePath == "_apply.sh" {
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
		!(isImplementationStage && smartContextEnabled && state.currentSubtask != nil && !uses["_apply.sh"]) {

		contextMessages = append(contextMessages, state.currentPlanState.ExecHistory())

		scriptContent, ok := state.currentPlanState.CurrentPlanFiles.Files["_apply.sh"]
		if !ok || scriptContent == "" {
			scriptContent = "[empty]"
		}

		contextMessages = append(contextMessages, "*Current* state of _apply.sh script:")
		contextMessages = append(contextMessages, fmt.Sprintf("\n\n- _apply.sh:\n\n```\n%s\n```", scriptContent))
	}

	return strings.Join(contextMessages, "\n\n") + "\n\n### END OF CONTEXT ###\n\n", nil
}

var pathRegex = regexp.MustCompile("`(.+?)`")

func (state *activeTellStreamState) checkAutoLoadContext() []string {
	req := state.req
	activePlan := state.activePlan
	contextsByPath := activePlan.ContextsByPath

	if !activePlan.AutoContext {
		return nil
	}

	if state.req.IsUserContinue {
		return nil
	}

	if state.req.IsChatOnly && state.iteration > 0 {
		return nil
	}

	if !(state.isContextStage || state.isPlanningStage || state.req.IsChatOnly) {
		return nil
	}

	// pick out all potential file paths within backticks
	matches := pathRegex.FindAllStringSubmatch(activePlan.CurrentReplyContent, -1)

	files := []string{}

	for _, match := range matches {
		trimmed := strings.TrimSpace(match[1])
		if trimmed == "" {
			continue
		}

		if req.ProjectPaths[trimmed] && contextsByPath[trimmed] == nil {
			files = append(files, trimmed)
		}
	}

	log.Printf("Tell plan - checkAutoLoadContext - files: %v\n", files)

	return files
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

			imageMessage := openai.ChatCompletionMessage{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
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
