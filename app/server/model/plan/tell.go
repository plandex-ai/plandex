package plan

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"plandex-server/db"
	"plandex-server/host"
	"plandex-server/model"
	"plandex-server/model/lib"
	"plandex-server/model/prompts"
	"plandex-server/types"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func Tell(plan *db.Plan, branch string, auth *types.ServerAuth, req *shared.TellPlanRequest) error {
	// goEnv := os.Getenv("GOENV") // Fetch the GOENV environment variable

	// log.Println("GOENV: " + goEnv)
	// if goEnv == "test" {
	// 	streamLoremIpsum(onStream)
	// 	return nil
	// }

	active := GetActivePlan(plan.Id, branch)
	if active != nil {
		return fmt.Errorf("plan %s branch %s already has an active stream on this host", plan.Id, branch)
	}

	modelStream, err := db.GetActiveModelStream(plan.Id, branch)
	if err != nil {
		log.Printf("Error getting active model stream: %v\n", err)
		return fmt.Errorf("error getting active model stream: %v", err)
	}

	if modelStream != nil {
		return fmt.Errorf("plan %s branch %s already has an active stream on host %s", plan.Id, branch, modelStream.InternalIp)
	}

	active = CreateActivePlan(plan.Id, branch, req.Prompt)

	modelStream = &db.ModelStream{
		OrgId:      auth.OrgId,
		PlanId:     plan.Id,
		InternalIp: host.Ip,
		Branch:     branch,
	}
	err = db.StoreModelStream(modelStream)

	if err != nil {
		log.Printf("Error storing model stream: %v\n", err)
		return fmt.Errorf("error storing model stream: %v", err)
	}

	active.ModelStreamId = modelStream.Id

	log.Println("Model stream id:", modelStream.Id)

	go execTellPlan(plan, branch, auth, req, active, 0)

	return nil
}

func execTellPlan(plan *db.Plan, branch string, auth *types.ServerAuth, req *shared.TellPlanRequest, active *types.ActivePlan, iteration int) {
	currentUserId := auth.User.Id
	currentOrgId := auth.OrgId

	if os.Getenv("IS_CLOUD") != "" {
		if auth.User.IsTrial {
			if plan.TotalReplies >= types.TrialMaxReplies {
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeTrialMessagesExceeded,
					Status: http.StatusForbidden,
					Msg:    "Free trial message limit exceeded",
					TrialMessagesExceededError: &shared.TrialMessagesExceededError{
						MaxReplies: types.TrialMaxReplies,
					},
				}
				return
			}
		}
	}

	planId := plan.Id
	err := db.SetPlanStatus(planId, branch, shared.PlanStatusReplying, "")
	if err != nil {
		log.Printf("Error setting plan %s status to replying: %v\n", planId, err)
		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error setting plan status to replying",
		}

		return
	}

	lockScope := db.LockScopeWrite
	if iteration > 0 {
		lockScope = db.LockScopeRead
	}
	repoLockId, err := db.LockRepo(auth.OrgId, auth.User.Id, planId, branch, lockScope)

	if err != nil {
		log.Printf("Error locking repo: %v\n", err)
		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error locking repo",
		}
		return
	}

	errCh := make(chan error)
	var modelContext []*db.Context
	var convo []*db.ConvoMessage
	var summaries []*db.ConvoSummary

	// get name for plan and rename it's a draft
	go func() {
		if plan.Name == "draft" {
			name, err := model.GenPlanName(req.Prompt)

			if err != nil {
				log.Printf("Error generating plan name: %v\n", err)
				errCh <- fmt.Errorf("error generating plan name: %v", err)
				return
			}

			err = db.RenamePlan(planId, name)

			if err != nil {
				log.Printf("Error renaming plan: %v\n", err)
				errCh <- fmt.Errorf("error renaming plan: %v", err)
				return
			}

			err = db.IncNumNonDraftPlans(currentUserId)

			if err != nil {
				log.Printf("Error incrementing num non draft plans: %v\n", err)
				errCh <- fmt.Errorf("error incrementing num non draft plans: %v", err)
				return
			}
		}

		errCh <- nil
	}()

	go func() {
		res, err := db.GetPlanContexts(currentOrgId, planId, true)
		if err != nil {
			log.Printf("Error getting plan modelContext: %v\n", err)
			errCh <- fmt.Errorf("error getting plan modelContext: %v", err)
			return
		}
		modelContext = res
		errCh <- nil
	}()

	go func() {
		res, err := db.GetPlanConvo(currentOrgId, planId)
		if err != nil {
			log.Printf("Error getting plan convo: %v\n", err)
			errCh <- fmt.Errorf("error getting plan convo: %v", err)
			return
		}
		convo = res

		promptTokens, err := shared.GetNumTokens(req.Prompt)
		if err != nil {
			log.Printf("Error getting prompt num tokens: %v\n", err)
			errCh <- fmt.Errorf("error getting prompt num tokens: %v", err)
			return
		}

		innerErrCh := make(chan error)

		go func() {
			if iteration == 0 {
				userMsg := db.ConvoMessage{
					OrgId:   currentOrgId,
					PlanId:  planId,
					UserId:  currentUserId,
					Role:    openai.ChatMessageRoleUser,
					Tokens:  promptTokens,
					Num:     len(convo) + 1,
					Message: req.Prompt,
				}

				_, err = db.StoreConvoMessage(&userMsg, auth.User.Id, branch, true)

				if err != nil {
					log.Printf("Error storing user message: %v\n", err)
					innerErrCh <- fmt.Errorf("error storing user message: %v", err)
					return
				}
			}

			innerErrCh <- nil
		}()

		go func() {
			var convoMessageIds []string

			for _, convoMessage := range convo {
				convoMessageIds = append(convoMessageIds, convoMessage.Id)
			}

			res, err := db.GetPlanSummaries(planId, convoMessageIds)
			if err != nil {
				log.Printf("Error getting plan summaries: %v\n", err)
				innerErrCh <- fmt.Errorf("error getting plan summaries: %v", err)
				return
			}
			summaries = res

			innerErrCh <- nil
		}()

		for i := 0; i < 2; i++ {
			err := <-innerErrCh
			if err != nil {
				errCh <- err
				return
			}
		}

		errCh <- nil
	}()

	err = func() error {
		defer func() {
			err = db.UnlockRepo(repoLockId)
			if err != nil {
				log.Printf("Error unlocking repo: %v\n", err)
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Error unlocking repo",
				}
				return
			}
		}()

		for i := 0; i < 3; i++ {
			err := <-errCh
			if err != nil {
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Error getting plan, context, convo, or summaries",
				}
				return err
			}
		}

		return nil
	}()

	if err != nil {
		return
	}

	if iteration == 0 {
		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			ap.Contexts = modelContext
			ap.PromptMessageNum = len(convo) + 1

			for _, context := range modelContext {
				if context.FilePath != "" {
					ap.ContextsByPath[context.FilePath] = context
				}
			}
		})
	} else {
		// reset current reply content and num tokens
		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			ap.CurrentReplyContent = ""
			ap.NumTokens = 0
		})
	}

	modelContextText, modelContextTokens, err := lib.FormatModelContext(modelContext)
	if err != nil {
		err = fmt.Errorf("error formatting model modelContext: %v", err)
		log.Println(err)

		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error formatting model modelContext",
		}
		return
	}

	systemMessageText := prompts.SysCreate + modelContextText
	systemMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemMessageText,
	}

	messages := []openai.ChatCompletionMessage{
		systemMessage,
	}

	var (
		numPromptTokens int
		promptTokens    int
	)
	if iteration == 0 {
		numPromptTokens, err = shared.GetNumTokens(req.Prompt)
		if err != nil {
			err = fmt.Errorf("error getting number of tokens in prompt: %v", err)
			log.Println(err)
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Error getting number of tokens in prompt",
			}
			return
		}
		promptTokens = prompts.PromptWrapperTokens + numPromptTokens
	}

	tokensBeforeConvo := prompts.CreateSysMsgNumTokens + modelContextTokens + promptTokens

	// print out breakdown of token usage
	log.Printf("System message tokens: %d\n", prompts.CreateSysMsgNumTokens)
	log.Printf("Context tokens: %d\n", modelContextTokens)
	log.Printf("Prompt tokens: %d\n", promptTokens)
	log.Printf("Total tokens before convo: %d\n", tokensBeforeConvo)

	if tokensBeforeConvo > shared.MaxTokens {
		// token limit already exceeded before adding conversation
		err := fmt.Errorf("token limit exceeded before adding conversation")
		log.Printf("Error: %v\n", err)
		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Token limit exceeded before adding conversation",
		}
		return
	}

	conversationTokens := 0
	tokensUpToTimestamp := make(map[int64]int)
	for _, convoMessage := range convo {
		conversationTokens += convoMessage.Tokens
		timestamp := convoMessage.CreatedAt.UnixNano() / int64(time.Millisecond)
		tokensUpToTimestamp[timestamp] = conversationTokens
		// log.Printf("Timestamp: %s | Tokens: %d | Total: %d | conversationTokens\n", convoMessage.Timestamp, convoMessage.Tokens, conversationTokens)
	}

	log.Printf("Conversation tokens: %d\n", conversationTokens)
	log.Printf("Max conversation tokens: %d\n", shared.MaxConvoTokens)

	// log.Println("Tokens up to timestamp:")
	// spew.Dump(tokensUpToTimestamp)

	log.Printf("Total tokens: %d\n", tokensBeforeConvo+conversationTokens)
	log.Printf("Max tokens: %d\n", shared.MaxTokens)

	var summary *db.ConvoSummary
	var summarizedToMessageId string
	if (tokensBeforeConvo+conversationTokens) > shared.MaxTokens ||
		conversationTokens > shared.MaxConvoTokens {
		log.Println("Token limit exceeded. Attempting to reduce via conversation summary.")

		// log.Printf("(tokensBeforeConvo+conversationTokens) > shared.MaxTokens: %v\n", (tokensBeforeConvo+conversationTokens) > shared.MaxTokens)
		// log.Printf("conversationTokens > shared.MaxConvoTokens: %v\n", conversationTokens > shared.MaxConvoTokens)

		// token limit exceeded after adding conversation
		// get summary for as much as the conversation as necessary to stay under the token limit
		for _, s := range summaries {
			timestamp := s.LatestConvoMessageCreatedAt.UnixNano() / int64(time.Millisecond)

			tokens, ok := tokensUpToTimestamp[timestamp]

			log.Printf("Last message timestamp: %d | found: %v\n", timestamp, ok)
			log.Printf("Tokens up to timestamp: %d\n", tokens)

			if !ok {
				err := fmt.Errorf("conversation summary timestamp not found in conversation")
				log.Printf("Error: %v\n", err)

				log.Println("timestamp:", timestamp)

				log.Println("Conversation summary:")
				spew.Dump(s)

				log.Println("tokensUpToTimestamp:")
				spew.Dump(tokensUpToTimestamp)

				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Conversation summary timestamp not found in conversation",
				}
				return
			}

			updatedConversationTokens := (conversationTokens - tokens) + s.Tokens
			savedTokens := conversationTokens - updatedConversationTokens

			log.Printf("Conversation summary tokens: %d\n", tokens)
			log.Printf("Updated conversation tokens: %d\n", updatedConversationTokens)
			log.Printf("Saved tokens: %d\n", savedTokens)

			if updatedConversationTokens <= shared.MaxConvoTokens &&
				(tokensBeforeConvo+updatedConversationTokens) <= shared.MaxTokens {
				log.Printf("Summarizing up to %s | saving %d tokens\n", s.LatestConvoMessageCreatedAt.Format(time.RFC3339), savedTokens)
				summary = s
				break
			}
		}

		if summary == nil {
			err := errors.New("couldn't get under token limit with conversation summary")
			log.Printf("Error: %v\n", err)
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Couldn't get under token limit with conversation summary",
			}
			return
		}
	}

	if summary == nil {
		for _, convoMessage := range convo {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    convoMessage.Role,
				Content: convoMessage.Message,
			})
		}
	} else {
		if (tokensBeforeConvo + summary.Tokens) > shared.MaxTokens {
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Token limit still exceeded after summarizing conversation",
			}
			return
		}
		summarizedToMessageId = summary.LatestConvoMessageId
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: summary.Summary,
		})

		// add messages after the last message in the summary
		for _, convoMessage := range convo {
			if convoMessage.CreatedAt.After(summary.LatestConvoMessageCreatedAt) {
				messages = append(messages, openai.ChatCompletionMessage{
					Role:    convoMessage.Role,
					Content: convoMessage.Message,
				})
			}
		}
	}

	var prompt string
	if iteration == 0 {
		prompt = req.Prompt
	} else {
		prompt = "Continue the plan."
	}

	promptMessage := &openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: fmt.Sprintf(prompts.PromptWrapperFormatStr, prompt),
	}
	messages = append(messages, *promptMessage)

	// log.Println("\n\nMessages:")
	// for _, message := range messages {
	// 	log.Printf("%s: %s\n", message.Role, message.Content)
	// }

	replyId := uuid.New().String()
	replyParser := types.NewReplyParser()
	replyFiles := []string{}
	replyNumTokens := 0

	modelReq := openai.ChatCompletionRequest{
		Model:       model.PlannerModel,
		Messages:    messages,
		Stream:      true,
		Temperature: 0.6,
		TopP:        0.7,
	}

	stream, err := model.Client.CreateChatCompletionStream(active.Ctx, modelReq)
	if err != nil {
		log.Printf("Error creating proposal GPT4 stream: %v\n", err)
		log.Println(err)

		errStr := err.Error()
		if strings.Contains(errStr, "status code: 400") &&
			strings.Contains(errStr, "reduce the length of the messages") {
			log.Println("Token limit exceeded")
		}

		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Error creating proposal GPT4 stream",
		}
		return
	}

	storeAssistantReply := func() (*db.ConvoMessage, []string, string, error) {
		num := len(convo) + 1
		if iteration == 0 {
			num++
		}

		assistantMsg := db.ConvoMessage{
			Id:      replyId,
			OrgId:   currentOrgId,
			PlanId:  planId,
			UserId:  currentUserId,
			Role:    openai.ChatMessageRoleAssistant,
			Tokens:  replyNumTokens,
			Num:     num,
			Message: GetActivePlan(planId, branch).CurrentReplyContent,
		}

		commitMsg, err := db.StoreConvoMessage(&assistantMsg, auth.User.Id, branch, false)

		if err != nil {
			log.Printf("Error storing assistant message: %v\n", err)
			return nil, replyFiles, "", err
		}

		return &assistantMsg, replyFiles, commitMsg, err
	}

	onError := func(streamErr error, storeDesc bool, convoMessageId, commitMsg string) {
		log.Printf("\nStream error: %v\n", streamErr)
		active.StreamDoneCh <- &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    "Stream error: " + streamErr.Error(),
		}

		storedMessage := false
		storedDesc := false

		if convoMessageId == "" {
			assistantMsg, _, msg, err := storeAssistantReply()
			if err == nil {
				convoMessageId = assistantMsg.Id
				commitMsg = msg
				storedMessage = true
			} else {
				log.Printf("Error storing assistant message after stream error: %v\n", err)
			}
		}

		if storeDesc && convoMessageId != "" {
			err = db.StoreDescription(&db.ConvoMessageDescription{
				OrgId:                 currentOrgId,
				PlanId:                planId,
				SummarizedToMessageId: summarizedToMessageId,
				MadePlan:              false,
				ConvoMessageId:        convoMessageId,
				Error:                 streamErr.Error(),
			})
			if err == nil {
				storedDesc = true
			} else {
				log.Printf("Error storing description after stream error: %v\n", err)
			}
		}

		if storedMessage || storedDesc {
			err = db.GitAddAndCommit(currentOrgId, planId, branch, commitMsg)
			if err != nil {
				log.Printf("Error committing after stream error: %v\n", err)
			}
		}
	}

	go func() {
		defer stream.Close()

		// Create a timer that will trigger if no chunk is received within the specified duration
		timer := time.NewTimer(model.OPENAI_STREAM_CHUNK_TIMEOUT)
		defer timer.Stop()

		for {
			select {
			case <-active.Ctx.Done():
				// The main modelContext was canceled (not the timer)
				log.Println("\nTell: stream canceled")
				return
			case <-timer.C:
				// Timer triggered because no new chunk was received in time
				log.Println("\nTell: stream timeout due to inactivity")
				onError(fmt.Errorf("stream timeout due to inactivity"), true, "", "")
				return
			default:
				response, err := stream.Recv()

				if err == nil {
					// Successfully received a chunk, reset the timer
					if !timer.Stop() {
						<-timer.C
					}
					timer.Reset(model.OPENAI_STREAM_CHUNK_TIMEOUT)
				}

				if err != nil {
					onError(fmt.Errorf("stream error: %v", err), true, "", "")
					return
				}

				if len(response.Choices) == 0 {
					onError(fmt.Errorf("stream finished with no choices"), true, "", "")
					return
				}

				if len(response.Choices) > 1 {
					onError(fmt.Errorf("stream finished with more than one choice"), true, "", "")
					return
				}

				choice := response.Choices[0]

				if choice.FinishReason != "" {
					active.Stream(shared.StreamMessage{
						Type: shared.StreamMessageDescribing,
					})

					err := db.SetPlanStatus(planId, branch, shared.PlanStatusDescribing, "")
					if err != nil {
						onError(fmt.Errorf("failed to set plan status to describing: %v", err), true, "", "")
						return
					}

					// log.Println("summarize convo:", spew.Sdump(convo))

					if len(convo) > 0 {
						// summarize in the background
						go summarizeConvo(summarizeConvoParams{
							planId:        planId,
							branch:        branch,
							convo:         convo,
							summaries:     summaries,
							promptMessage: promptMessage,
							currentOrgId:  currentOrgId,
						})
					}

					repoLockId, err := db.LockRepo(auth.OrgId, auth.User.Id, planId, branch, db.LockScopeWrite)

					var execStatus *types.PlanExecStatus
					err = func() error {
						defer func() {
							err = db.UnlockRepo(repoLockId)
							if err != nil {
								log.Printf("Error unlocking repo: %v\n", err)
								active.StreamDoneCh <- &shared.ApiError{
									Type:   shared.ApiErrorTypeOther,
									Status: http.StatusInternalServerError,
									Msg:    "Error unlocking repo",
								}
							}
						}()

						assistantMsg, files, convoCommitMsg, err := storeAssistantReply()

						if err != nil {
							onError(fmt.Errorf("failed to store assistant message: %v", err), true, "", "")
							return err
						}

						var description *db.ConvoMessageDescription

						errCh := make(chan error, 2)

						go func() {
							if len(files) == 0 {
								description = &db.ConvoMessageDescription{
									OrgId:                 currentOrgId,
									PlanId:                planId,
									ConvoMessageId:        assistantMsg.Id,
									SummarizedToMessageId: summarizedToMessageId,
									MadePlan:              false,
								}
							} else {
								description, err = genPlanDescription(planId, branch, active.Ctx)
								if err != nil {
									onError(fmt.Errorf("failed to generate plan description: %v", err), true, assistantMsg.Id, convoCommitMsg)
									return
								}

								description.OrgId = currentOrgId
								description.ConvoMessageId = assistantMsg.Id
								description.SummarizedToMessageId = summarizedToMessageId
								description.MadePlan = true
								description.Files = files
							}

							err = db.StoreDescription(description)

							if err != nil {
								onError(fmt.Errorf("failed to store description: %v", err), false, assistantMsg.Id, convoCommitMsg)
								errCh <- err
								return
							}

							errCh <- nil
						}()

						go func() {

							execStatus, err = ExecStatus(assistantMsg.Message, active.Ctx)
							if err != nil {
								onError(fmt.Errorf("failed to get exec status: %v", err), false, assistantMsg.Id, convoCommitMsg)
								errCh <- err
								return
							}

							errCh <- nil
						}()

						for i := 0; i < 2; i++ {
							err := <-errCh
							if err != nil {
								return err
							}
						}

						err = db.GitAddAndCommit(currentOrgId, planId, branch, convoCommitMsg)
						if err != nil {
							onError(fmt.Errorf("failed to commit: %v", err), false, assistantMsg.Id, convoCommitMsg)
							return err
						}

						return nil
					}()

					if err != nil {
						return
					}

					if !req.AutoContinue || execStatus.Finished || execStatus.NeedsInput {
						if GetActivePlan(planId, branch).BuildFinished() {
							active.Stream(shared.StreamMessage{
								Type: shared.StreamMessageFinished,
							})
						} else {
							UpdateActivePlan(planId, branch, func(active *types.ActivePlan) {
								active.RepliesFinished = true
							})
						}

					} else {
						// continue plan
						execTellPlan(plan, branch, auth, req, active, iteration+1)
					}

					return
				}

				delta := choice.Delta
				content := delta.Content
				UpdateActivePlan(planId, branch, func(active *types.ActivePlan) {
					active.CurrentReplyContent += content
					active.NumTokens++
				})

				// log.Printf("%s", content)
				active.Stream(shared.StreamMessage{
					Type:       shared.StreamMessageReply,
					ReplyChunk: content,
				})
				replyParser.AddChunk(content, true)

				files, fileContents, _, numTokens := replyParser.Read()
				replyNumTokens = numTokens

				// log.Printf("Reply num tokens: %d\n", replyNumTokens)

				if len(files) > len(replyFiles) {
					log.Printf("Files: %v\n", files)
					for i := len(files) - 1; i > len(replyFiles)-1; i-- {
						file := files[i]
						log.Printf("Queuing build for %s\n", file)
						QueueBuild(currentOrgId, currentUserId, planId, branch, &types.ActiveBuild{
							AssistantMessageId: replyId,
							ReplyContent:       active.CurrentReplyContent,
							FileContent:        fileContents[i],
							Path:               file,
						})
						replyFiles = append(replyFiles, file)
						UpdateActivePlan(planId, branch, func(active *types.ActivePlan) {
							active.Files = append(active.Files, file)
						})
					}
				}

			}
		}
	}()
}

type summarizeConvoParams struct {
	planId        string
	branch        string
	convo         []*db.ConvoMessage
	summaries     []*db.ConvoSummary
	promptMessage *openai.ChatCompletionMessage
	currentOrgId  string
}

func summarizeConvo(params summarizeConvoParams) error {
	planId := params.planId
	branch := params.branch
	convo := params.convo
	summaries := params.summaries
	promptMessage := params.promptMessage
	currentOrgId := params.currentOrgId

	log.Println("Generating plan summary for planId:", planId)

	// log the parameters above
	// log.Printf("planId: %s\n", planId)
	// log.Printf("convo: ")
	// spew.Dump(convo)
	// log.Printf("summaries: ")
	// spew.Dump(summaries)
	// log.Printf("promptMessage: ")
	// spew.Dump(promptMessage)
	// log.Printf("currentOrgId: %s\n", currentOrgId)

	var summaryMessages []*openai.ChatCompletionMessage
	var latestSummary *db.ConvoSummary
	var numMessagesSummarized int = 0
	var latestMessageSummarizedAt time.Time
	var latestMessageId string
	if len(summaries) > 0 {
		latestSummary = summaries[len(summaries)-1]
		numMessagesSummarized = latestSummary.NumMessages
	}

	// log.Println("Latest summary:")
	// spew.Dump(latestSummary)

	if latestSummary == nil {
		for _, convoMessage := range convo {
			summaryMessages = append(summaryMessages, &openai.ChatCompletionMessage{
				Role:    convoMessage.Role,
				Content: convoMessage.Message,
			})
			latestMessageId = convoMessage.Id
			latestMessageSummarizedAt = convoMessage.CreatedAt
		}
	} else {
		summaryMessages = append(summaryMessages, &openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: latestSummary.Summary,
		})

		latestConvoMessage := convo[len(convo)-1]
		latestMessageId = latestConvoMessage.Id
		latestMessageSummarizedAt = latestConvoMessage.CreatedAt

		summaryMessages = append(summaryMessages, &openai.ChatCompletionMessage{
			Role:    latestConvoMessage.Role,
			Content: latestConvoMessage.Message,
		})
	}

	if promptMessage != nil {
		summaryMessages = append(summaryMessages, promptMessage, &openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: GetActivePlan(planId, branch).CurrentReplyContent,
		})
	}

	summary, err := model.PlanSummary(model.PlanSummaryParams{
		Conversation:                summaryMessages,
		LatestConvoMessageId:        latestMessageId,
		LatestConvoMessageCreatedAt: latestMessageSummarizedAt,
		NumMessages:                 numMessagesSummarized + 1,
		OrgId:                       currentOrgId,
		PlanId:                      planId,
	})

	if err != nil {
		log.Printf("Error generating plan summary for plan %s: %v\n", planId, err)
		return err
	}

	log.Println("Generated plan summary for plan", planId)

	err = db.StoreSummary(summary)

	if err != nil {
		log.Printf("Error storing plan summary for plan %s: %v\n", planId, err)
		return err
	}

	return nil
}
