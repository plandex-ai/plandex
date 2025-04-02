package plan

import (
	"fmt"
	"log"
	"plandex-server/types"
	"regexp"
	"sort"
	"strings"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

type formatModelContextParams struct {
	includeMaps          bool
	smartContextEnabled  bool
	includeApplyScript   bool
	baseOnly             bool
	cacheControl         bool
	activeOnly           bool
	autoOnly             bool
	activatePaths        map[string]bool
	activatePathsOrdered []string
	maxTokens            int
}

func (state *activeTellStreamState) formatModelContext(params formatModelContextParams) []*types.ExtendedChatMessagePart {
	log.Println("Tell plan - formatModelContext")

	includeMaps := params.includeMaps
	smartContextEnabled := params.smartContextEnabled
	includeApplyScript := params.includeApplyScript
	currentStage := state.currentStage

	basicOnly := params.baseOnly
	activeOnly := params.activeOnly
	autoOnly := params.autoOnly
	activatePaths := params.activatePaths
	activatePathsOrdered := params.activatePathsOrdered
	if activatePaths == nil {
		activatePaths = map[string]bool{}
	}

	maxTokens := params.maxTokens

	// log all the flags
	log.Printf("Tell plan - formatModelContext - basicOnly: %t, activeOnly: %t, autoOnly: %t, smartContextEnabled: %t, execEnabled: %t, includeMaps: %t, activatePaths: %v, activatePathsOrdered: %v, maxTokens: %d\n",
		basicOnly, activeOnly, autoOnly, smartContextEnabled, includeApplyScript, includeMaps, activatePaths, activatePathsOrdered, params.maxTokens)

	var contextBodies []string = []string{
		"### LATEST PLAN CONTEXT ###",
	}
	addedFilesSet := map[string]bool{}

	uses := map[string]bool{}

	// log.Println("Tell plan - formatModelContext - state.currentSubtask:\n", spew.Sdump(state.currentSubtask))
	// if state.currentSubtask != nil {
	// 	log.Println("Tell plan - formatModelContext - state.currentSubtask.UsesFiles:\n", spew.Sdump(state.currentSubtask.UsesFiles))
	// }
	// log.Println("Tell plan - formatModelContext - currentStage.TellStage:\n", currentStage.TellStage)
	// log.Println("Tell plan - formatModelContext - smartContextEnabled:\n", smartContextEnabled)

	if currentStage.TellStage == shared.TellStageImplementation && smartContextEnabled && state.currentSubtask != nil {
		log.Println("Tell plan - formatModelContext - implementation stage - smart context enabled for current subtask")
		for _, path := range state.currentSubtask.UsesFiles {
			uses[path] = true
		}
		if verboseLogging {
			log.Printf("Tell plan - formatModelContext - uses: %v\n", uses)
		}
	}

	// log.Println("Tell plan - formatModelContext - state.modelContext:\n", spew.Sdump(state.modelContext))

	totalTokens := 0

	type toLoad struct {
		FilePath    string
		Name        string
		Url         string
		NumTokens   int
		Body        string
		ContextType shared.ContextType
		ImageDetail openai.ImageURLDetail
		IsPending   bool
	}
	var toLoadAll []toLoad

	for _, part := range state.modelContext {
		if verboseLogging {
			log.Printf("Tell plan - formatModelContext - part: %s - %s - %s - %d tokens\n", part.ContextType, part.Name, part.FilePath, part.NumTokens)
		}
		if !(part.ContextType == shared.ContextMapType && includeMaps) {
			if basicOnly && part.AutoLoaded {
				if verboseLogging {
					log.Println("Tell plan - formatModelContext - skipping auto loaded part -- basicOnly && part.AutoLoaded")
				}
				continue
			}

			if autoOnly && !part.AutoLoaded {
				if verboseLogging {
					log.Println("Tell plan - formatModelContext - skipping auto loaded part -- autoOnly && !part.AutoLoaded")
				}
				continue
			}
		}

		if currentStage.TellStage == shared.TellStageImplementation && smartContextEnabled && state.currentSubtask != nil && part.ContextType == shared.ContextFileType && !uses[part.FilePath] {
			if verboseLogging {
				log.Println("Tell plan - formatModelContext - skipping part -- currentStage.TellStage == shared.TellStageImplementation && smartContextEnabled && state.currentSubtask != nil && part.ContextType == shared.ContextFileType && !uses[part.FilePath]")
			}
			continue
		}

		if activeOnly && !activatePaths[part.FilePath] {
			if verboseLogging {
				log.Println("Tell plan - formatModelContext - skipping part -- activeOnly && !activatePaths[part.FilePath]")
			}
			continue
		}

		if part.ContextType == shared.ContextMapType && !includeMaps {
			if verboseLogging {
				log.Println("Tell plan - formatModelContext - skipping part -- part.ContextType == shared.ContextMapType && !includeMaps")
			}
			continue
		}

		toLoadAll = append(toLoadAll, toLoad{
			FilePath:    part.FilePath,
			NumTokens:   part.NumTokens,
			Body:        part.Body,
			ContextType: part.ContextType,
			Name:        part.Name,
			Url:         part.Url,
			ImageDetail: part.ImageDetail,
		})

		if part.ContextType == shared.ContextFileType {
			addedFilesSet[part.FilePath] = true
		}
	}

	// Add any current pendingFiles in plan that weren't added to the context
	var currentPlanFiles *shared.CurrentPlanFiles
	var pendingFiles map[string]string = map[string]string{}
	if state.currentPlanState != nil && state.currentPlanState.CurrentPlanFiles != nil && state.currentPlanState.CurrentPlanFiles.Files != nil {
		currentPlanFiles = state.currentPlanState.CurrentPlanFiles
		pendingFiles = state.currentPlanState.CurrentPlanFiles.Files
	}

	for filePath, body := range pendingFiles {
		if !addedFilesSet[filePath] {

			if currentStage.TellStage == shared.TellStageImplementation && smartContextEnabled && !uses[filePath] {
				continue
			}

			if filePath == "_apply.sh" {
				continue
			}

			if activeOnly && !activatePaths[filePath] {
				continue
			}

			numTokens := shared.GetNumTokensEstimate(body)

			toLoadAll = append(toLoadAll, toLoad{
				FilePath:    filePath,
				NumTokens:   numTokens,
				Body:        body,
				ContextType: shared.ContextFileType,
				Name:        filePath,
				IsPending:   true,
			})

			if verboseLogging {
				log.Printf("Tell plan - formatModelContext - added current plan file - %s\n", filePath)
			}
		}
	}

	if len(activatePathsOrdered) > 0 {
		indexByPath := map[string]int{}
		for i, path := range activatePathsOrdered {
			indexByPath[path] = i
		}

		sort.Slice(toLoadAll, func(i, j int) bool {
			iIndex, ok1 := indexByPath[toLoadAll[i].FilePath]
			jIndex, ok2 := indexByPath[toLoadAll[j].FilePath]

			// If neither has an index, sort by Name so we are using a stable order for caching
			if !ok1 && !ok2 {
				return toLoadAll[i].Name < toLoadAll[j].Name
			}

			// If only i doesn't have an index, it goes after j
			if !ok1 {
				return false
			}

			// If only j doesn't have an index, it goes after i
			if !ok2 {
				return true
			}

			// Both have indices, compare them
			return iIndex < jIndex
		})
	}

	for _, part := range toLoadAll {
		totalTokens += part.NumTokens

		if maxTokens > 0 && totalTokens > maxTokens {
			if verboseLogging {
				log.Printf("Tell plan - formatModelContext - total tokens: %d\n", totalTokens)
			}
			break
		}

		var message string
		var fmtStr string
		var args []any

		if part.ContextType == shared.ContextDirectoryTreeType {
			fmtStr = "\n\n- %s | directory tree:\n\n```\n%s\n```"
			args = append(args, part.FilePath, part.Body)
		} else if part.ContextType == shared.ContextFileType {
			// if we're in the context phase and the file is pending, just include that the file is pending, not the full content
			// there is generally enough related context from the conversation and summary to decide on whether to load the file or not
			// without this, the context phase can get overloaded with pending file content
			if currentStage.TellStage == shared.TellStagePlanning &&
				currentStage.PlanningPhase == shared.PlanningPhaseContext &&
				part.IsPending {
				fmtStr = "\n\n- File `%s` has pending changes (%d 🪙)"
				args = append(args, part.FilePath, part.NumTokens)
			} else {

				fmtStr = "\n\n- %s:\n\n```\n%s\n```"

				// use pending file value if available
				var body string
				var found bool
				res, ok := pendingFiles[part.FilePath]
				if ok {
					body = res
					found = true
				}
				if !found {
					body = part.Body
				}

				args = append(args, part.FilePath, body)
			}
		} else if part.ContextType == shared.ContextMapType {
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
			contextBodies = append(contextBodies, message)
		}

		if verboseLogging {
			log.Printf("Tell plan - formatModelContext - added context: %s - %s - %s - %d tokens\n", part.ContextType, part.Name, part.FilePath, part.NumTokens)
		}
	}

	if currentPlanFiles != nil && len(currentPlanFiles.Removed) > 0 {
		contextBodies = append(contextBodies, "*Removed files:*\n")
		for path := range currentPlanFiles.Removed {
			contextBodies = append(contextBodies, fmt.Sprintf("- %s", path))
		}
		contextBodies = append(contextBodies, "These files have been *removed* and are no longer in the plan. If you want to re-add them to the plan, you must explicitly create them again.")

		log.Println("Tell plan - formatModelContext - added removed files")
		log.Println(contextBodies)
	}

	var execScriptLines []string

	if includeApplyScript &&
		// don't show _apply.sh history and content if smart context is enabled and the current subtask doesn't use it
		!(currentStage.TellStage == shared.TellStageImplementation && smartContextEnabled && state.currentSubtask != nil && !uses["_apply.sh"]) {

		execHistory := state.currentPlanState.ExecHistory()

		execScriptLines = append(execScriptLines, execHistory)

		scriptContent, ok := pendingFiles["_apply.sh"]
		var isEmpty bool
		if !ok || scriptContent == "" {
			scriptContent = "[empty]"
			isEmpty = true
		}

		execScriptLines = append(execScriptLines, "*Current* state of _apply.sh script:")
		execScriptLines = append(execScriptLines, fmt.Sprintf("\n\n- _apply.sh:\n\n```\n%s\n```", scriptContent))

		if isEmpty && currentStage.TellStage == shared.TellStagePlanning && currentStage.PlanningPhase != shared.PlanningPhaseContext {
			execScriptLines = append(execScriptLines, "The _apply.sh script is *empty*. You ABSOLUTELY MUST include a '### Commands' section in your response prior to the '### Tasks' section that evaluates whether any commands should be written to _apply.sh during the plan. This is MANDATORY. Do NOT UNDER ANY CIRCUMSTANCES omit this section. If you determine that commands should be added or updated in _apply.sh, you MUST also create a subtask referencing _apply.sh in the '### Tasks' section.")

			if execHistory != "" {
				execScriptLines = append(execScriptLines, "Consider the history of previously executed _apply.sh scripts when determining which commands to include in the new _apply.sh file. Are there any commands that should be run again after code changes? If so, mention them in the '### Commands' section and then include a subtask to include them in the _apply.sh file in the '### Tasks' section.")
			}
		}
	}

	log.Println("Tell plan - formatModelContext - contextMessages:", len(contextBodies))

	textMsg := &types.ExtendedChatMessagePart{
		Type: openai.ChatMessagePartTypeText,
		Text: strings.Join(contextBodies, "\n"),
	}

	res := []*types.ExtendedChatMessagePart{textMsg}

	// now add any images that should be included
	// we'll check later for model image support once the final model config is set
	for _, load := range toLoadAll {
		if load.ContextType == shared.ContextImageType {
			res = append(res, &types.ExtendedChatMessagePart{
				Type: openai.ChatMessagePartTypeText,
				Text: fmt.Sprintf("Image: %s", load.Name),
			})
			res = append(res, &types.ExtendedChatMessagePart{
				Type:     openai.ChatMessagePartTypeImageURL,
				ImageURL: &openai.ChatMessageImageURL{URL: shared.GetImageDataURI(load.Body, load.FilePath), Detail: load.ImageDetail},
			})
		}
	}

	if params.cacheControl && len(res) > 0 {
		res[len(res)-1].CacheControl = &types.CacheControlSpec{
			Type: types.CacheControlTypeEphemeral,
		}
	}

	if len(execScriptLines) > 0 {
		res = append(res, &types.ExtendedChatMessagePart{
			Type: openai.ChatMessagePartTypeText,
			Text: strings.Join(execScriptLines, "\n"),
		})
	}

	res = append(res, &types.ExtendedChatMessagePart{
		Type: openai.ChatMessagePartTypeText,
		Text: "### END OF CONTEXT ###\n\n",
	})

	return res
}

var pathRegex = regexp.MustCompile("`(.+?)`")

type checkAutoLoadContextResult struct {
	autoLoadPaths        []string
	activatePaths        map[string]bool
	hasExplicitPaths     bool
	activatePathsOrdered []string
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

	// for chat responses, only auto load context if we're in the context phase
	if req.IsChatOnly && currentStage.PlanningPhase != shared.PlanningPhaseContext {
		return checkAutoLoadContextResult{}
	}

	log.Printf("%d existing contexts by path\n", len(contextsByPath))

	// pick out all potential file paths within backticks
	matches := pathRegex.FindAllStringSubmatch(activePlan.CurrentReplyContent, -1)

	toAutoLoad := map[string]bool{}
	toActivate := map[string]bool{}
	toActivateOrdered := []string{}
	allSet := map[string]bool{}
	allFiles := []string{}

	for _, match := range matches {
		trimmed := strings.TrimSpace(match[1])
		if trimmed == "" {
			continue
		}

		if req.ProjectPaths[trimmed] {
			if !allSet[trimmed] {
				allFiles = append(allFiles, trimmed)
				allSet[trimmed] = true

				toActivate[trimmed] = true
				toActivateOrdered = append(toActivateOrdered, trimmed)
				if contextsByPath[trimmed] == nil {
					toAutoLoad[trimmed] = true
				}

			}
		}
	}

	toAutoLoadPaths := []string{}
	for path := range toAutoLoad {
		toAutoLoadPaths = append(toAutoLoadPaths, path)
	}

	hasExplicitPaths := strings.Contains(activePlan.CurrentReplyContent, "### Files")

	log.Printf("Tell plan - checkAutoLoadContext - toAutoLoad: %v\n", toAutoLoadPaths)
	log.Printf("Tell plan - checkAutoLoadContext - toActivate: %v\n", toActivateOrdered)

	return checkAutoLoadContextResult{
		autoLoadPaths:        toAutoLoadPaths,
		activatePaths:        toActivate,
		activatePathsOrdered: toActivateOrdered,
		hasExplicitPaths:     hasExplicitPaths,
	}
}
