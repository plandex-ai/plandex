package plan

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/notify"
	"plandex-server/types"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	shared "plandex-shared"

	"github.com/davecgh/go-spew/spew"
)

const verboseLogging = false

var openingTagRegex = regexp.MustCompile(`<PlandexBlock\s+lang="(.+?)"\s+path="(.+?)".*?>`)

type processChunkResult struct {
	shouldReturn bool
	shouldStop   bool
}

func (state *activeTellStreamState) processChunk(choice types.ExtendedChatCompletionStreamChoice) processChunkResult {
	req := state.req
	// missingFileResponse := state.missingFileResponse
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

	defer func() {
		if r := recover(); r != nil {
			log.Printf("processChunk: Panic: %v\n%s\n", r, string(debug.Stack()))

			go notify.NotifyErr(notify.SeverityError, fmt.Errorf("processChunk: Panic: %v\n%s", r, string(debug.Stack())))

			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Panic in processChunk",
			}
		}
	}()

	delta := choice.Delta
	content := delta.Content

	if state.modelConfig.BaseModelConfig.IncludeReasoning && delta.Reasoning != "" {
		content = delta.Reasoning
	}

	if content == "" {
		return processChunkResult{}
	}

	processor.chunksReceived++

	if verboseLogging {
		log.Printf("Adding chunk to parser: %s\n", content)
		log.Printf("fileOpen: %v\n", processor.fileOpen)
	}

	replyParser.AddChunk(content, true)
	parserRes := replyParser.Read()

	if !processor.fileOpen && parserRes.CurrentFilePath != "" {
		if verboseLogging {
			log.Printf("File open: %s\n", parserRes.CurrentFilePath)
		}
		processor.fileOpen = true
	}

	if processor.fileOpen && strings.HasSuffix(active.CurrentReplyContent+content, "</PlandexBlock>") {
		if verboseLogging {
			log.Println("FinishAndRead because of closing tag")
		}
		parserRes = replyParser.FinishAndRead()
		processor.fileOpen = false
	}

	if processor.fileOpen && parserRes.CurrentFilePath == "" {
		if verboseLogging {
			log.Println("File open but current file path is empty, closing file")
		}
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
	bufferOrStreamRes := processor.bufferOrStream(content, &parserRes, state.currentStage, state.manualStop)

	if currentFile != "" &&
		!req.IsChatOnly &&
		active.ContextsByPath[currentFile] == nil &&
		req.ProjectPaths[currentFile] &&
		!active.AllowOverwritePaths[currentFile] {
		return state.handleMissingFile(bufferOrStreamRes.content, currentFile, bufferOrStreamRes.blockLang)
	}

	UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
		ap.CurrentReplyContent += content
		ap.NumTokens++
	})

	if verboseLogging {
		log.Println("processor before bufferOrStream")
		spew.Dump(processor)
		log.Println("maybeFilePath", parserRes.MaybeFilePath)
		log.Println("currentFilePath", parserRes.CurrentFilePath)
		log.Println("bufferOrStreamRes")
		spew.Dump(bufferOrStreamRes)
	}

	if bufferOrStreamRes.shouldStream {
		active.Stream(shared.StreamMessage{
			Type:       shared.StreamMessageReply,
			ReplyChunk: bufferOrStreamRes.content,
		})
	}

	if verboseLogging {
		log.Println("processor after bufferOrStream")
		spew.Dump(processor)
	}

	if !req.IsChatOnly && len(operations) > len(processor.replyOperations) {
		state.handleNewOperations(&parserRes)
	}

	return processChunkResult{
		shouldStop: bufferOrStreamRes.shouldStop,
	}
}

type bufferOrStreamResult struct {
	shouldStream bool
	content      string
	blockLang    string
	shouldStop   bool
}

func (processor *chunkProcessor) bufferOrStream(content string, parserRes *types.ReplyParserRes, currentStage shared.CurrentStage, manualStopSequences []string) bufferOrStreamResult {
	if len(manualStopSequences) > 0 {
		for _, stopSequence := range manualStopSequences {

			// if the chunk contains the entire stop sequence, stream everything before it then caller can stop the stream
			if strings.Contains(content, stopSequence) {
				split := strings.Split(content, stopSequence)
				if len(split) > 1 {
					return bufferOrStreamResult{
						shouldStream: true,
						content:      split[0],
						shouldStop:   true,
					}
				} else {
					// there was nothing before the stop sequence, so nothing to stream
					return bufferOrStreamResult{
						shouldStream: false,
						shouldStop:   true,
					}
				}
			}

			// otherwise if the buffer plus chunk contains the stop sequence, don't stream anything and stop the stream
			if strings.Contains(processor.contentBuffer+content, stopSequence) {
				log.Printf("bufferOrStream - stop sequence found in buffer plus chunk\n")
				split := strings.Split(content, stopSequence)
				if len(split) > 1 {
					// we'll stream the part before the stop sequence
					return bufferOrStreamResult{
						shouldStream: true,
						content:      split[0],
						shouldStop:   true,
					}
				} else {
					// there was nothing before the stop sequence, so nothing to stream
					return bufferOrStreamResult{
						shouldStream: false,
						shouldStop:   true,
					}
				}
			}

			// otherwise if the buffer plus chunk ends with a prefix of the stop sequence, buffer it and continue

			toCheck := processor.contentBuffer + content
			tailLen := len(stopSequence) - 1
			if tailLen > len(toCheck) {
				tailLen = len(toCheck)
			}
			suffix := toCheck[len(toCheck)-tailLen:]

			if strings.HasPrefix(stopSequence, suffix) {
				log.Printf("bufferOrStream - stop sequence prefix found in buffer plus chunk. buffer and continue\n")
				processor.contentBuffer += content
				return bufferOrStreamResult{
					shouldStream: false,
					content:      content,
				}
			}

		}
	}

	// apart from manual stop sequences, no buffering in planning stages
	if currentStage.TellStage == shared.TellStagePlanning {
		return bufferOrStreamResult{
			shouldStream: true,
			content:      content,
		}
	}

	var shouldStream bool
	var blockLang string

	awaitingTag := processor.awaitingBlockOpeningTag || processor.awaitingBlockClosingTag || processor.awaitingOpClosingTag
	awaitingAny := awaitingTag || processor.awaitingBackticks

	if awaitingAny {
		if verboseLogging {
			log.Println("awaitingAny")
		}
		processor.contentBuffer += content
		content = processor.contentBuffer

		if verboseLogging {
			log.Printf("awaitingBlockOpeningTag: %v\n", processor.awaitingBlockOpeningTag)
			log.Printf("awaitingBlockClosingTag: %v\n", processor.awaitingBlockClosingTag)
			log.Printf("awaitingBackticks: %v\n", processor.awaitingBackticks)
			log.Printf("awaitingOpClosingTag: %v\n", processor.awaitingOpClosingTag)
			log.Printf("content: %q\n", content)
		}
	}

	if processor.awaitingBackticks {
		if strings.Contains(content, "```") {
			processor.awaitingBackticks = false
			content = strings.ReplaceAll(content, "```", "\\`\\`\\`")

			if !(processor.awaitingBlockOpeningTag || processor.awaitingBlockClosingTag) {
				shouldStream = true
			}
		} else if !strings.HasSuffix(content, "`") {
			// fewer than 3 backticks, no need to escape
			processor.awaitingBackticks = false

			if !(processor.awaitingBlockOpeningTag || processor.awaitingBlockClosingTag) {
				shouldStream = true
			}
		}
	}

	if awaitingTag {
		if verboseLogging {
			log.Println("awaitingTag")
		}
		if processor.awaitingBlockOpeningTag {
			if verboseLogging {
				log.Println("processor.awaitingBlockOpeningTag")
			}
			var matchedPrefix bool

			if parserRes.CurrentFilePath != "" {
				matched, replaced := replaceCodeBlockOpeningTag(content, func(lang string) string {
					blockLang = lang
					return "```" + lang
				})

				if matched {
					shouldStream = true
					processor.awaitingBlockOpeningTag = false
					processor.fileOpen = true
					content = replaced
				} else {
					// tag is missing - something is wrong - we shouldn't be here but let's try to recover anyway
					if verboseLogging {
						log.Printf("Opening <PlandexBlock> tag is missing even though parserRes.CurrentFile is set - something is wrong: %s\n", content)
					}
					processor.awaitingBlockOpeningTag = false
					processor.fileOpen = false
					content += "\n```" // add ``` to the end of the line to close the markdown code block
					shouldStream = true
				}
			} else {
				split := strings.Split(content, "<")

				if len(split) > 1 {
					last := split[len(split)-1]
					if verboseLogging {
						log.Printf("last: %s\n", last)
					}
					if strings.HasPrefix(`PlandexBlock lang="`, last) {
						if verboseLogging {
							log.Println("strings.HasPrefix(`PlandexBlock lang=", last)
						}
						shouldStream = false
						matchedPrefix = true
					} else if strings.HasPrefix(last, `PlandexBlock lang="`) {
						if verboseLogging {
							log.Println("partialOpeningTagRegex.MatchString(last)")
						}
						shouldStream = false
						matchedPrefix = true
					} else {
						if verboseLogging {
							log.Println("partialOpeningTagRegex.MatchString(last) is false")
						}
					}
				}
			}

			if !matchedPrefix && parserRes.MaybeFilePath == "" && parserRes.CurrentFilePath == "" {
				// wasn't really a file path / code block
				processor.awaitingBlockOpeningTag = false
				shouldStream = true
			}
		} else if processor.awaitingBlockClosingTag {
			if parserRes.CurrentFilePath == "" {
				if strings.Contains(content, "</PlandexBlock>") {
					shouldStream = true
					processor.awaitingBlockClosingTag = false
					processor.fileOpen = false
					// replace </PlandexBlock> with ``` to close the markdown code block
					content = strings.ReplaceAll(content, "</PlandexBlock>", "```")
				} else {
					log.Printf("Closing </PlandexBlock> tag is missing even though parserRes.CurrentOperation is nil - something is wrong: %s\n", content)
					processor.awaitingBlockClosingTag = false
					shouldStream = true
				}
			}
		} else if processor.awaitingOpClosingTag {
			if verboseLogging {
				log.Printf("awaitingOpClosingTag: %v\n", processor.awaitingOpClosingTag)
			}
			if strings.Contains(content, "<EndPlandexFileOps/>") {
				if verboseLogging {
					log.Printf("Found <EndPlandexFileOps/>\n")
				}
				processor.awaitingOpClosingTag = false
				content = strings.Replace(content, "\n<EndPlandexFileOps/>", "", 1)
				content = strings.Replace(content, "<EndPlandexFileOps/>", "", 1)
				shouldStream = true
			}
		}

	} else {
		if verboseLogging {
			log.Println("not awaiting tag")
		}

		if parserRes.MaybeFilePath != "" && parserRes.CurrentFilePath == "" {
			processor.awaitingBlockOpeningTag = true
		} else {
			// this will set processor.awaitingBlockOpeningTag to true if the content starts with any prefix of<PlandexBlock lang=" *or* any prefix of a full opening tag
			// if the full tag is in the content, it will later get set to false again when the full tag is handled
			split := strings.Split(content, "<")
			if len(split) > 1 {
				last := split[len(split)-1]

				if strings.HasPrefix(`PlandexBlock lang="`, last) {
					processor.awaitingBlockOpeningTag = true
				} else if strings.HasPrefix(last, `PlandexBlock lang="`) {
					processor.awaitingBlockOpeningTag = true
				}
			}
		}

		if parserRes.CurrentFilePath != "" {
			if verboseLogging {
				log.Println("parserRes.CurrentFilePath != \"\"")
			}
			if strings.Contains(content, "</PlandexBlock>") {
				if verboseLogging {
					log.Println("strings.Contains(content, \"</PlandexBlock>\")")
				}
				processor.awaitingBlockClosingTag = true
			} else {
				if verboseLogging {
					log.Println("not strings.Contains(content, \"</PlandexBlock>\")")
				}
				split := strings.Split(content, "<")
				// log.Printf("split: %v\n", split)
				if len(split) > 1 {
					if verboseLogging {
						log.Println("len(split) > 1")
					}
					last := split[len(split)-1]
					// log.Printf("last: %s\n", last)
					if strings.HasPrefix("/PlandexBlock>", last) {
						if verboseLogging {
							log.Println("strings.HasPrefix(\"/PlandexBlock>\", last)")
						}
						processor.awaitingBlockClosingTag = true
					}
				}
			}
		} else if parserRes.FileOperationBlockOpen() {
			if verboseLogging {
				log.Println("parserRes.FileOperationBlockOpen()")
			}
			if strings.Contains(content, "<EndPlandexFileOps/>") {
				if verboseLogging {
					log.Println("strings.Contains(content, \"<EndPlandexFileOps/>\")")
				}
				processor.awaitingOpClosingTag = true
			} else {
				if verboseLogging {
					log.Println("not strings.Contains(content, \"<EndPlandexFileOps/>\")")
				}
				split := strings.Split(content, "<")
				if len(split) > 1 {
					if verboseLogging {
						log.Println("len(split) > 1")
					}
					last := split[len(split)-1]
					if strings.HasPrefix("EndPlandexFileOps/>", last) {
						if verboseLogging {
							log.Println("strings.HasPrefix(\"EndPlandexFileOps/>\", last)")
						}
						processor.awaitingOpClosingTag = true
					}
				}
			}
		} else if strings.Contains(content, "</PlandexBlock>") {
			if verboseLogging {
				log.Println("strings.Contains(content, \"</PlandexBlock>\")")
			}
			content = strings.Replace(content, "</PlandexBlock>", "```", 1)
		} else if strings.Contains(content, "<EndPlandexFileOps/>") {
			if verboseLogging {
				log.Println("strings.Contains(content, \"<EndPlandexFileOps/>\")")
			}
			content = strings.Replace(content, "\n<EndPlandexFileOps/>", "", 1)
			content = strings.Replace(content, "<EndPlandexFileOps/>", "", 1)
		}

		if processor.fileOpen && (strings.Contains(content, "```") || strings.HasSuffix(content, "`")) {
			if verboseLogging {
				log.Println("processor.fileOpen && (strings.Contains(content, \"```\") || strings.HasSuffix(content, \"`\"))")
			}
			processor.awaitingBackticks = true
		}

		var matchedOpeningTag bool
		if processor.fileOpen {
			if verboseLogging {
				log.Println("processor.fileOpen")
			}
			var replaced string

			matchedOpeningTag, replaced = replaceCodeBlockOpeningTag(content, func(lang string) string {
				blockLang = lang
				return "```" + lang
			})

			if verboseLogging {
				log.Println("matchedOpeningTag", matchedOpeningTag)
				log.Println("replaced", replaced)
			}

			if matchedOpeningTag {
				processor.awaitingBlockOpeningTag = false
				content = replaced
			}
		}

		shouldStream = !processor.awaitingBlockOpeningTag && !processor.awaitingBlockClosingTag && !processor.awaitingOpClosingTag && !processor.awaitingBackticks

		if verboseLogging {
			log.Println("processor.awaitingBlockOpeningTag", processor.awaitingBlockOpeningTag)
			log.Println("processor.awaitingBlockClosingTag", processor.awaitingBlockClosingTag)
			log.Println("processor.awaitingOpClosingTag", processor.awaitingOpClosingTag)
			log.Println("processor.awaitingBackticks", processor.awaitingBackticks)

			log.Println("shouldStream", shouldStream)
		}
	}

	if verboseLogging {
		log.Println("returning bufferOrStreamResult")
		log.Println("shouldStream", shouldStream)
		log.Println("content", content)
		log.Println("blockLang", blockLang)
	}

	if shouldStream {
		processor.contentBuffer = ""
	} else {
		processor.contentBuffer = content
	}

	return bufferOrStreamResult{
		shouldStream: shouldStream,
		content:      content,
		blockLang:    blockLang,
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
			// log.Println(strconv.Quote(op.Content))

			buildState := &activeBuildStreamState{
				modelStreamId: state.modelStreamId,
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
				opContentTokens = shared.GetNumTokensEstimate(op.Content)
			} else {
				opContentTokens = op.NumTokens
			}

			// log.Printf("buildState.queueBuilds - op.Description:\n%s\n", op.Description)

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

func (state *activeTellStreamState) handleMissingFile(content, currentFile, blockLang string) processChunkResult {
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
		go notify.NotifyErr(notify.SeverityError, fmt.Errorf("error setting plan %s status to prompting: %v", planId, err))

		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error setting plan status to prompting",
		}
		return processChunkResult{}
	}

	var trimmedReply string

	UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
		ap.MissingFilePath = currentFile
		trimmedReply = replyParser.GetReplyForMissingFile()
		ap.CurrentReplyContent = trimmedReply
	})

	// log.Println("Content:")
	// log.Println(content)

	// log.Println("Block lang:")
	// log.Println(blockLang)

	// log.Println("Trimmed content:")
	// log.Println(trimmedReply)

	// try to replace the code block opening tag in the chunk with an empty string
	// this will remove the code block opening tag if it exists
	splitBy := "```" + blockLang
	split := strings.Split(content, splitBy)
	chunkToStream := split[0] + splitBy + "\n"

	// log.Printf("chunkToStream: %s\n", chunkToStream)

	if chunkToStream != "" {
		log.Printf("Streaming remaining chunk before missing file prompt: %s\n", chunkToStream)
		active.Stream(shared.StreamMessage{
			Type:       shared.StreamMessageReply,
			ReplyChunk: chunkToStream,
		})
		active.FlushStreamBuffer()
		time.Sleep(20 * time.Millisecond)
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
		state.execHookOnStop(false)
		return processChunkResult{shouldReturn: true}

	case <-time.After(30 * time.Minute): // long timeout here since we're waiting for user input
		log.Println("Timeout waiting for missing file choice")
		state.onError(onErrorParams{
			streamErr: fmt.Errorf("timeout waiting for missing file choice"),
			storeDesc: true,
		})
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
	execTellPlan(execTellPlanParams{
		clients:             clients,
		plan:                plan,
		branch:              branch,
		auth:                auth,
		req:                 req,
		iteration:           iteration, // keep the same iteration
		missingFileResponse: userChoice,
	})

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

func replaceCodeBlockOpeningTag(content string, replaceWithFn func(lang string) string) (bool, string) {
	// check for opening tag matching <PlandexBlock lang="..." path="...">
	match := openingTagRegex.FindStringSubmatch(content)

	if match != nil {
		// Found complete opening tag with lang and path attributes
		lang := match[1] // Extract the language from the first capture group
		return true, strings.Replace(content, match[0], replaceWithFn(lang), 1)
	} else if strings.Contains(content, "<PlandexBlock>") {
		// This is a fallback case that should probably be removed since we now require both attributes
		return true, strings.Replace(content, "<PlandexBlock>", replaceWithFn(""), 1)
	}

	return false, ""
}
