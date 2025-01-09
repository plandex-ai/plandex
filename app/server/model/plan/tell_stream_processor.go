package plan

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/types"
	"regexp"
	"strings"
	"time"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

var openingTagRegex = regexp.MustCompile(`<PlandexBlock\s+lang="(.+?)".*?>`)

type processChunkResult struct {
	shouldReturn bool
}

func (state *activeTellStreamState) processChunk(choice openai.ChatCompletionStreamChoice) processChunkResult {
	req := state.req
	missingFileResponse := state.missingFileResponse
	processor := state.chunkProcessor
	replyParser := state.replyParser
	plan := state.plan
	planId := plan.Id
	branch := state.branch
	active := GetActivePlan(planId, branch)

	if active == nil {
		state.onActivePlanMissingError()
		return processChunkResult{}
	}

	processor.chunksReceived++
	delta := choice.Delta
	content := delta.Content

	// log.Printf("content: %s\n", content)

	// buffer if we're continuing after a missing file response to avoid sending redundant opening tags
	if missingFileResponse != "" {
		if processor.maybeRedundantOpeningTagContent != "" {
			if strings.Contains(content, "\n") {
				processor.maybeRedundantOpeningTagContent = ""
			} else {
				processor.maybeRedundantOpeningTagContent += content
			}

			// skip processing this chunk
			return processChunkResult{}
		} else if processor.chunksReceived < 3 && strings.Contains(content, "<PlandexBlock") {
			// received <PlandexBlock in first 3 chunks after missing file response
			// means this is a redundant start of a new file block, so just ignore it

			processor.maybeRedundantOpeningTagContent += content

			// skip processing this chunk
			return processChunkResult{}
		}
	}

	// log.Printf("Adding chunk to parser: %s\n", content)
	// log.Printf("fileOpen: %v\n", processor.fileOpen)

	replyParser.AddChunk(content, true)
	parserRes := replyParser.Read()

	if !processor.fileOpen && parserRes.CurrentFilePath != "" {
		log.Printf("File open: %s\n", parserRes.CurrentFilePath)
		processor.fileOpen = true
	}

	if processor.fileOpen && strings.HasSuffix(active.CurrentReplyContent+content, "</PlandexBlock>") {
		log.Println("FinishAndRead because of closing tag")
		parserRes = replyParser.FinishAndRead()
		processor.fileOpen = false
	}

	if processor.fileOpen && parserRes.CurrentFilePath == "" {
		log.Println("File open but current file path is empty, closing file")
		processor.fileOpen = false
	}

	operations := parserRes.Operations
	state.replyNumTokens = parserRes.TotalTokens
	currentFile := parserRes.CurrentFilePath

	// log.Printf("currentFile: %s\n", currentFile)
	// log.Println("files:")
	// spew.Dump(files)

	// Handle file that is present in project paths but not in context
	// Prompt user for what to do on the client side, stop the stream, and wait for user response before proceeding
	if currentFile != "" &&
		!req.IsChatOnly &&
		active.ContextsByPath[currentFile] == nil &&
		req.ProjectPaths[currentFile] && !active.AllowOverwritePaths[currentFile] {
		return state.handleMissingFile(content, currentFile)
	}

	UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
		ap.CurrentReplyContent += content
		ap.NumTokens++
	})

	// log.Println("processor before bufferOrStream")
	// spew.Dump(processor)
	// log.Println("maybeFilePath", parserRes.MaybeFilePath)
	// log.Println("currentFilePath", parserRes.CurrentFilePath)

	res := processor.bufferOrStream(content, &parserRes, state.isImplementationStage)

	// log.Println("res")
	// spew.Dump(res)

	if res.shouldStream {
		active.Stream(shared.StreamMessage{
			Type:       shared.StreamMessageReply,
			ReplyChunk: res.content,
		})
	}

	// log.Println("processor after bufferOrStream")
	// spew.Dump(processor)

	if !req.IsChatOnly && len(operations) > len(processor.replyOperations) {
		state.handleNewOperations(&parserRes)
	}

	return processChunkResult{}
}

type bufferOrStreamResult struct {
	shouldStream bool
	content      string
}

func (processor *chunkProcessor) bufferOrStream(content string, parserRes *types.ReplyParserRes, isImplementationStage bool) bufferOrStreamResult {
	// no buffering in planning stages
	if !isImplementationStage {
		return bufferOrStreamResult{
			shouldStream: true,
			content:      content,
		}
	}

	var shouldStream bool
	if processor.awaitingBlockOpeningTag || processor.awaitingBlockClosingTag || processor.awaitingBackticks || processor.awaitingOpClosingTag {
		processor.contentBuffer.WriteString(content)
		s := processor.contentBuffer.String()

		// log.Printf("awaitingBlockOpeningTag: %v\n", processor.awaitingBlockOpeningTag)
		// log.Printf("awaitingBlockClosingTag: %v\n", processor.awaitingBlockClosingTag)
		// log.Printf("awaitingBackticks: %v\n", processor.awaitingBackticks)
		// log.Printf("awaitingOpClosingTag: %v\n", processor.awaitingOpClosingTag)
		// log.Printf("s: %q\n", s)

		if processor.awaitingBackticks {
			if strings.Contains(s, "```") {
				processor.awaitingBackticks = false
				s = strings.ReplaceAll(s, "```", "\\`\\`\\`")

				if processor.awaitingBlockOpeningTag || processor.awaitingBlockClosingTag {
					// update buffer with escaped backticks
					processor.contentBuffer.Reset()
					processor.contentBuffer.WriteString(s)
				} else {
					content = s
					processor.contentBuffer.Reset()
					shouldStream = true
				}
			} else if !strings.HasSuffix(s, "`") {
				// fewer than 3 backticks, no need to escape
				processor.awaitingBackticks = false

				if !(processor.awaitingBlockOpeningTag || processor.awaitingBlockClosingTag) {
					content = s
					processor.contentBuffer.Reset()
					shouldStream = true
				}
			}
		}

		if processor.awaitingBlockOpeningTag {
			if parserRes.MaybeFilePath == "" && parserRes.CurrentFilePath == "" {
				// wasn't really a file path / code block
				processor.awaitingBlockOpeningTag = false
				content = s
				processor.contentBuffer.Reset()
				shouldStream = true
			} else if parserRes.CurrentFilePath != "" {
				matched, replaced := matchCodeBlockOpeningTag(s)

				if matched {
					shouldStream = true
					processor.awaitingBlockOpeningTag = false
					processor.fileOpen = true
					content = replaced
					processor.contentBuffer.Reset()
				} else {
					// tag is missing - something is wrong - we shouldn't be here but let's try to recover anyway
					log.Printf("Opening <PlandexBlock> tag is missing even though parserRes.CurrentFile is set - something is wrong: %s\n", s)
					processor.awaitingBlockOpeningTag = false
					processor.fileOpen = false
					s += "\n```" // add ``` to the end of the line to close the markdown code block
					content = s
					processor.contentBuffer.Reset()
					shouldStream = true
				}
			}
		} else if processor.awaitingBlockClosingTag {
			if parserRes.CurrentFilePath == "" {
				if strings.Contains(s, "</PlandexBlock>") {
					shouldStream = true
					processor.awaitingBlockClosingTag = false
					processor.fileOpen = false
					// replace </PlandexBlock> with ``` to close the markdown code block
					s = strings.ReplaceAll(s, "</PlandexBlock>", "```")
					content = s
					processor.contentBuffer.Reset()
				} else {
					log.Printf("Closing </PlandexBlock> tag is missing even though parserRes.CurrentOperation is nil - something is wrong: %s\n", s)
					processor.awaitingBlockClosingTag = false
					content = s
					processor.contentBuffer.Reset()
					shouldStream = true
				}
			}
		} else if processor.awaitingOpClosingTag {
			// log.Printf("awaitingOpClosingTag: %v\n", processor.awaitingOpClosingTag)
			if strings.Contains(s, "<EndPlandexFileOps/>") {
				log.Printf("Found <EndPlandexFileOps/>\n")
				processor.awaitingOpClosingTag = false
				s = strings.Replace(s, "\n<EndPlandexFileOps/>", "", 1)
				s = strings.Replace(s, "<EndPlandexFileOps/>", "", 1)
				content = s
				processor.contentBuffer.Reset()
				shouldStream = true
			}
		}

	} else {
		if parserRes.MaybeFilePath != "" && parserRes.CurrentFilePath == "" {
			processor.awaitingBlockOpeningTag = true
		}

		if parserRes.CurrentFilePath != "" {
			if strings.Contains(content, "</PlandexBlock>") {
				processor.awaitingBlockClosingTag = true
			} else {
				split := strings.Split(content, "<")
				// log.Printf("split: %v\n", split)
				if len(split) > 1 {
					last := split[len(split)-1]
					// log.Printf("last: %s\n", last)
					if strings.HasPrefix("/PlandexBlock>", last) {
						processor.awaitingBlockClosingTag = true
					}
				}
			}
		} else if parserRes.FileOperationBlockOpen() {
			if strings.Contains(content, "<EndPlandexFileOps/>") {
				processor.awaitingOpClosingTag = true
			} else {
				split := strings.Split(content, "<")
				if len(split) > 1 {
					last := split[len(split)-1]
					if strings.HasPrefix("EndPlandexFileOps/>", last) {
						processor.awaitingOpClosingTag = true
					}
				}
			}
		} else if strings.Contains(content, "</PlandexBlock>") {
			content = strings.Replace(content, "</PlandexBlock>", "```", 1)
		} else if strings.Contains(content, "<EndPlandexFileOps/>") {
			content = strings.Replace(content, "\n<EndPlandexFileOps/>", "", 1)
			content = strings.Replace(content, "<EndPlandexFileOps/>", "", 1)
		}

		if processor.fileOpen && (strings.Contains(content, "```") || strings.HasSuffix(content, "`")) {
			processor.awaitingBackticks = true
		}

		var matchedOpeningTag bool
		if processor.fileOpen {
			var replaced string
			matchedOpeningTag, replaced = matchCodeBlockOpeningTag(content)
			if matchedOpeningTag {
				content = replaced
			}
		}

		shouldStream = !processor.awaitingBlockOpeningTag && !processor.awaitingBlockClosingTag && !processor.awaitingOpClosingTag && !processor.awaitingBackticks

		if !shouldStream {
			processor.contentBuffer.WriteString(content)
		}
	}

	return bufferOrStreamResult{
		shouldStream: shouldStream,
		content:      content,
	}
}

func (state *activeTellStreamState) handleNewOperations(parserRes *types.ReplyParserRes) {

	processor := state.chunkProcessor
	plan := state.plan
	planId := plan.Id
	branch := state.branch
	clients := state.clients
	auth := state.auth
	req := state.req
	replyId := state.replyId
	currentOrgId := state.currentOrgId
	currentUserId := state.currentUserId
	settings := state.settings

	operations := parserRes.Operations

	log.Printf("%d new operations\n", len(operations)-len(processor.replyOperations))

	for i, op := range operations {
		if i < len(processor.replyOperations) {
			continue
		}

		log.Printf("Detected operation: %s\n", op.Name())

		if req.BuildMode == shared.BuildModeAuto {
			log.Printf("Queuing build for %s\n", op.Name())
			// log.Println("Content:")
			// log.Println(fileContents[i])

			buildState := &activeBuildStreamState{
				tellState:     state,
				clients:       clients,
				auth:          auth,
				currentOrgId:  currentOrgId,
				currentUserId: currentUserId,
				plan:          plan,
				branch:        branch,
				settings:      settings,
				modelContext:  state.modelContext,
			}

			var opContentTokens int
			if op.Type == shared.OperationTypeFile {
				var err error
				opContentTokens, err = shared.GetNumTokens(op.Content)
				if err != nil {
					log.Printf("Error getting num tokens for operation %s: %v\n", op.Name(), err)
					state.onError(fmt.Errorf("error getting num tokens for operation %s: %v", op.Name(), err), true, "", "")
					return
				}
			} else {
				opContentTokens = op.NumTokens
			}

			buildState.queueBuilds([]*types.ActiveBuild{{
				ReplyId:           replyId,
				FileDescription:   op.Description,
				FileContent:       op.Content,
				FileContentTokens: opContentTokens,
				Path:              op.Path,
				MoveDestination:   op.Destination,
				IsMoveOp:          op.Type == shared.OperationTypeMove,
				IsRemoveOp:        op.Type == shared.OperationTypeRemove,
				IsResetOp:         op.Type == shared.OperationTypeReset,
			}})
		}
		processor.replyOperations = append(processor.replyOperations, op)
		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			ap.Operations = append(ap.Operations, op)
		})
	}

}

func (state *activeTellStreamState) handleMissingFile(content, currentFile string) processChunkResult {
	branch := state.branch
	plan := state.plan
	planId := plan.Id
	replyParser := state.replyParser
	iteration := state.iteration
	clients := state.clients
	auth := state.auth
	req := state.req

	active := GetActivePlan(planId, branch)

	if active == nil {
		state.onActivePlanMissingError()
		return processChunkResult{}
	}

	log.Printf("Attempting to overwrite a file that isn't in context: %s\n", currentFile)

	// attempting to overwrite a file that isn't in context
	// we will stop the stream and ask the user what to do
	err := db.SetPlanStatus(planId, branch, shared.PlanStatusMissingFile, "")

	if err != nil {
		log.Printf("Error setting plan %s status to prompting: %v\n", planId, err)
		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error setting plan status to prompting",
		}
		return processChunkResult{}
	}

	var previousReplyContent string
	var trimmedContent string

	UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
		ap.MissingFilePath = currentFile
		previousReplyContent = ap.CurrentReplyContent
		trimmedContent = replyParser.GetReplyForMissingFile()
		ap.CurrentReplyContent = trimmedContent
	})

	log.Println("Previous reply content:")
	log.Println(previousReplyContent)

	// log.Println("Trimmed content:")
	// log.Println(trimmedContent)

	chunkToStream := getCroppedChunk(previousReplyContent+content, trimmedContent, content)

	// log.Printf("chunkToStream: %s\n", chunkToStream)

	if chunkToStream != "" {
		log.Printf("Streaming remaining chunk before missing file prompt: %s\n", chunkToStream)
		active.Stream(shared.StreamMessage{
			Type:       shared.StreamMessageReply,
			ReplyChunk: chunkToStream,
		})
	}

	log.Printf("Prompting user for missing file: %s\n", currentFile)

	active.Stream(shared.StreamMessage{
		Type:                   shared.StreamMessagePromptMissingFile,
		MissingFilePath:        currentFile,
		MissingFileAutoContext: active.AutoContext,
	})

	log.Printf("Stopping stream for missing file: %s\n", currentFile)
	// log.Printf("Chunk content: %s\n", content)
	// log.Printf("Current reply content: %s\n", active.CurrentReplyContent)

	// stop stream for now
	active.CancelModelStreamFn()

	log.Printf("Stopped stream for missing file: %s\n", currentFile)

	// wait for user response to come in
	var userChoice shared.RespondMissingFileChoice
	select {
	case <-active.Ctx.Done():
		log.Println("Context cancelled while waiting for missing file response")
		state.execHookOnStop(true)
		return processChunkResult{shouldReturn: true}

	case <-time.After(30 * time.Minute): // long timeout here since we're waiting for user input
		log.Println("Timeout waiting for missing file choice")
		state.onError(fmt.Errorf("timeout waiting for missing file choice"), true, "", "")
		return processChunkResult{}

	case userChoice = <-active.MissingFileResponseCh:
	}

	log.Printf("User choice for missing file: %s\n", userChoice)

	active.ResetModelCtx()

	UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
		ap.MissingFilePath = ""
		ap.CurrentReplyContent = replyParser.GetReplyForMissingFile()
	})

	log.Println("Continuing stream")

	// continue plan
	execTellPlan(
		clients,
		plan,
		branch,
		auth,
		req,
		iteration, // keep the same iteration
		userChoice,
		false,
		0,
	)

	return processChunkResult{shouldReturn: true}
}

func getCroppedChunk(uncropped, cropped, chunk string) string {
	uncroppedIdx := strings.Index(uncropped, chunk)
	if uncroppedIdx == -1 {
		return ""
	}
	croppedChunk := cropped[uncroppedIdx:]
	return croppedChunk
}

func matchCodeBlockOpeningTag(content string) (bool, string) {
	// check for opening tag matching <PlandexBlock lang="...">
	match := openingTagRegex.FindStringSubmatch(content)

	if match != nil {
		// Found complete opening tag with lang attribute
		lang := match[1] // Extract the language from the first capture group
		return true, strings.Replace(content, match[0], "```"+lang, 1)
	} else if strings.Contains(content, "<PlandexBlock>") {
		return true, strings.Replace(content, "<PlandexBlock>", "```", 1)
	}

	return false, ""
}
