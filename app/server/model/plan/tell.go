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
	"plandex-server/model"
	"plandex-server/model/lib"
	"plandex-server/model/prompts"
	"plandex-server/types"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func Tell(client *openai.Client, plan *db.Plan, branch string, auth *types.ServerAuth, req *shared.TellPlanRequest) error {
	log.Printf("Tell: Called with plan ID %s on branch %s\n", plan.Id, branch)

	_, err := activatePlan(client, plan, branch, auth, req.Prompt, false)

	if err != nil {
		log.Printf("Error activating plan: %v\n", err)
		return err
	}

	go execTellPlan(
		client,
		plan,
		branch,
		auth,
		req,
		0,
		"",
		req.BuildMode == shared.BuildModeAuto,
	)

	log.Printf("Tell: Tell operation completed successfully for plan ID %s on branch %s\n", plan.Id, branch)
	return nil
}

func execTellPlan(
	client *openai.Client,
	plan *db.Plan,
	branch string,
	auth *types.ServerAuth,
	req *shared.TellPlanRequest,
	iteration int,
	missingFileResponse shared.RespondMissingFileChoice,
	shouldBuildPending bool,
) {
	log.Printf("execTellPlan: Called for plan ID %s on branch %s, iteration %d\n", plan.Id, branch, iteration)
	currentUserId := auth.User.Id
	currentOrgId := auth.OrgId

	active := GetActivePlan(plan.Id, branch)

	if os.Getenv("IS_CLOUD") != "" &&
		missingFileResponse == "" {
		log.Println("execTellPlan: IS_CLOUD environment variable is set")
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

		log.Printf("execTellPlan: execTellPlan operation completed for plan ID %s on branch %s, iteration %d\n", plan.Id, branch, iteration)
		return
	}

	lockScope := db.LockScopeWrite
	if iteration > 0 || missingFileResponse != "" {
		lockScope = db.LockScopeRead
	}
	repoLockId, err := db.LockRepo(
		db.LockRepoParams{
			OrgId:    auth.OrgId,
			UserId:   auth.User.Id,
			PlanId:   planId,
			Branch:   branch,
			Scope:    lockScope,
			Ctx:      active.ModelStreamCtx,
			CancelFn: active.CancelModelStreamFn,
		},
	)

	if err != nil {
		log.Printf("execTellPlan: Error locking repo for plan ID %s on branch %s: %v\n", plan.Id, branch, err)
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
			name, err := model.GenPlanName(client, req.Prompt)

			if err != nil {
				log.Printf("Error generating plan name: %v\n", err)
				errCh <- fmt.Errorf("error generating plan name: %v", err)
				return
			}

			tx, err := db.Conn.Begin()
			if err != nil {
				log.Printf("Error starting transaction: %v\n", err)
				errCh <- fmt.Errorf("error starting transaction: %v", err)
			}

			// Ensure that rollback is attempted in case of failure
			defer func() {
				if err != nil {
					if rbErr := tx.Rollback(); rbErr != nil {
						log.Printf("transaction rollback error: %v\n", rbErr)
					} else {
						log.Println("transaction rolled back")
					}
				}
			}()

			err = db.RenamePlan(planId, name, tx)

			if err != nil {
				log.Printf("Error renaming plan: %v\n", err)
				errCh <- fmt.Errorf("error renaming plan: %v", err)
				return
			}

			err = db.IncNumNonDraftPlans(currentUserId, tx)

			if err != nil {
				log.Printf("Error incrementing num non draft plans: %v\n", err)
				errCh <- fmt.Errorf("error incrementing num non draft plans: %v", err)
				return
			}

			err = tx.Commit()
			if err != nil {
				log.Printf("Error committing transaction: %v\n", err)
				errCh <- fmt.Errorf("error committing transaction: %v", err)
				return
			}
		}

		errCh <- nil
	}()

	go func() {
		if iteration > 0 || missingFileResponse != "" {
			modelContext = active.Contexts
		} else {
			res, err := db.GetPlanContexts(currentOrgId, planId, true)
			if err != nil {
				log.Printf("Error getting plan modelContext: %v\n", err)
				errCh <- fmt.Errorf("error getting plan modelContext: %v", err)
				return
			}
			modelContext = res
		}
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
			if iteration == 0 && missingFileResponse == "" {
				num := len(convo) + 1

				userMsg := db.ConvoMessage{
					OrgId:   currentOrgId,
					PlanId:  planId,
					UserId:  currentUserId,
					Role:    openai.ChatMessageRoleUser,
					Tokens:  promptTokens,
					Num:     num,
					Message: req.Prompt,
				}

				_, err = db.StoreConvoMessage(&userMsg, auth.User.Id, branch, true)

				if err != nil {
					log.Printf("Error storing user message: %v\n", err)
					innerErrCh <- fmt.Errorf("error storing user message: %v", err)
					return
				}

				UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
					ap.MessageNum = num
				})
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
		var err error
		defer func() {
			if err != nil {
				log.Printf("Error: %v\n", err)
				err = db.GitClearUncommittedChanges(auth.OrgId, planId)
				if err != nil {
					log.Printf("Error clearing uncommitted changes: %v\n", err)
				}
			}

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
			err = <-errCh
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

	if iteration == 0 && missingFileResponse == "" {
		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			ap.Contexts = modelContext

			for _, context := range modelContext {
				if context.FilePath != "" {
					ap.ContextsByPath[context.FilePath] = context
				}
			}
		})
	} else if missingFileResponse == "" {
		// reset current reply content and num tokens
		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			ap.CurrentReplyContent = ""
			ap.NumTokens = 0
		})
	}

	// if any skipped paths have since been added to context, remove them from skipped paths
	if len(active.SkippedPaths) > 0 {
		var toUnskipPaths []string
		for contextPath := range active.ContextsByPath {
			if active.SkippedPaths[contextPath] {
				toUnskipPaths = append(toUnskipPaths, contextPath)
			}
		}
		if len(toUnskipPaths) > 0 {
			UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
				for _, path := range toUnskipPaths {
					delete(ap.SkippedPaths, path)
				}
			})
		}
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

	if len(active.SkippedPaths) > 0 {
		systemMessageText += "\n\nSome files have been skipped by the user and *must not* be generated. The user will handle any updates to these files themselves. Skip any parts of the plan that require generating these files. You *must not* generate a file block for any of these files.\nSkipped files:\n"
		for skippedPath := range active.SkippedPaths {
			systemMessageText += fmt.Sprintf("- %s\n", skippedPath)
		}
	}

	messages := []openai.ChatCompletionMessage{
		systemMessage,
	}

	var (
		numPromptTokens int
		promptTokens    int
	)
	if iteration == 0 && missingFileResponse == "" {
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

	replyId := uuid.New().String()
	replyParser := types.NewReplyParser()
	replyFiles := []string{}
	replyNumTokens := 0
	chunksReceived := 0
	maybeRedundantBacktickContent := ""

	var promptMessage *openai.ChatCompletionMessage
	var prompt string
	if iteration == 0 {
		prompt = req.Prompt
	} else {
		prompt = "Continue the plan."
	}

	promptMessage = &openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: fmt.Sprintf(prompts.PromptWrapperFormatStr, prompt),
	}

	if missingFileResponse == "" {
		messages = append(messages, *promptMessage)
	}

	if missingFileResponse != "" {
		log.Println("Missing file response:", missingFileResponse, "setting replyParser")

		replyParser.AddChunk(active.CurrentReplyContent, true)
		res := replyParser.Read()
		currentFile := res.CurrentFilePath

		log.Printf("Current file: %s\n", currentFile)
		// log.Println("Current reply content:\n", active.CurrentReplyContent)

		replyContent := active.CurrentReplyContent
		numTokens := active.NumTokens

		if missingFileResponse == shared.RespondMissingFileChoiceSkip {
			replyBeforeCurrentFile := replyParser.GetReplyBeforeCurrentPath()
			numTokens, err = shared.GetNumTokens(replyBeforeCurrentFile)
			if err != nil {
				log.Printf("Error getting num tokens for reply before current file: %v\n", err)
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Error getting num tokens for reply before current file",
				}
				return
			}

			replyContent = replyBeforeCurrentFile
			replyParser = types.NewReplyParser()
			replyParser.AddChunk(replyContent, true)

			UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
				ap.CurrentReplyContent = replyContent
				ap.NumTokens = numTokens
				ap.SkippedPaths[currentFile] = true
			})

		} else {
			if missingFileResponse == shared.RespondMissingFileChoiceOverwrite {
				UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
					ap.AllowOverwritePaths[currentFile] = true
				})
			}
		}
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: active.CurrentReplyContent,
	})

	if missingFileResponse != "" {
		if missingFileResponse == shared.RespondMissingFileChoiceSkip {
			res := replyParser.Read()

			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf(`You *must not* generate content for the file %s. Skip this file and continue with the plan according to the 'Your instructions' section if there are any remaining tasks or subtasks. Don't repeat any part of the previous message. If there are no remaining tasks or subtasks, say "All tasks have been completed." per your instructions`, res.CurrentFilePath),
			})

		} else {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: "Continue generating exactly where you left off in the previous message. Don't produce any other output before continuing or repeat any part of the previous message. Do *not* duplicate the last line of the previous response before continuing. Do *not* include triple backticks and a language name like '```python' or '```yaml' at the start of the response, since these have already been included in the previous message. Continue from where you left off seamlessly to generate the rest of the file block. You must include closing triple backticks at the end of the file block. When the file block is finished, continue with the plan according to the 'Your instructions' sections if there are any remaining tasks or subtasks. If there are no remaining tasks or subtasks, say 'All tasks have been completed.' per your instructions.",
			})
		}
	}

	// log.Println("\n\nMessages:")
	// for _, message := range messages {
	// 	log.Printf("%s: %s\n", message.Role, message.Content)
	// }

	modelReq := openai.ChatCompletionRequest{
		Model:       model.PlannerModel,
		Messages:    messages,
		Stream:      true,
		Temperature: 0.6,
		TopP:        0.7,
	}

	stream, err := client.CreateChatCompletionStream(active.ModelStreamCtx, modelReq)
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

	if shouldBuildPending {
		go func() {
			pendingBuildsByPath, err := active.PendingBuildsByPath(auth.OrgId, auth.User.Id, convo)

			if err != nil {
				log.Printf("Error getting pending builds by path: %v\n", err)
				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Error getting pending builds by path",
				}
				return
			}

			if len(pendingBuildsByPath) == 0 {
				log.Println("Tell plan: no pending builds")
				return
			}

			for _, pendingBuilds := range pendingBuildsByPath {
				queueBuilds(client, auth.OrgId, auth.User.Id, planId, branch, pendingBuilds)
			}
		}()
	}

	UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
		ap.CurrentStreamingReplyId = replyId
		ap.CurrentReplyDoneCh = make(chan bool, 1)
	})

	storeAssistantReply := func() (*db.ConvoMessage, []string, string, error) {
		num := len(convo) + 1
		if iteration == 0 && missingFileResponse == "" {
			num++
		}

		log.Printf("Storing assistant reply num %d\n", num)

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

		UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
			ap.MessageNum = num
		})

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
					if err.Error() == "context canceled" {
						log.Println("Tell: stream context canceled")
						return
					}

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
					log.Println("Model stream finished")

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
						go summarizeConvo(client, summarizeConvoParams{
							planId:        planId,
							branch:        branch,
							convo:         convo,
							summaries:     summaries,
							promptMessage: promptMessage,
							currentOrgId:  currentOrgId,
						})
					}

					log.Println("Locking repo for assistant reply and description")

					repoLockId, err := db.LockRepo(
						db.LockRepoParams{
							OrgId:    currentOrgId,
							UserId:   currentUserId,
							PlanId:   planId,
							Branch:   branch,
							Scope:    db.LockScopeWrite,
							Ctx:      active.Ctx,
							CancelFn: active.CancelFn,
						},
					)

					if err != nil {
						log.Printf("Error locking repo: %v\n", err)
						active.StreamDoneCh <- &shared.ApiError{
							Type:   shared.ApiErrorTypeOther,
							Status: http.StatusInternalServerError,
							Msg:    "Error locking repo",
						}
						return
					}

					log.Println("Locked repo for assistant reply and description")

					var shouldContinue bool
					err = func() error {
						defer func() {
							if err != nil {
								log.Printf("Error storing reply and description: %v\n", err)
								err = db.GitClearUncommittedChanges(auth.OrgId, planId)
								if err != nil {
									log.Printf("Error clearing uncommitted changes: %v\n", err)
								}
							}

							log.Println("Unlocking repo for assistant reply and description")

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
								description, err = genPlanDescription(client, planId, branch, active.Ctx)
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
							shouldContinue, err = ExecStatusShouldContinue(client, assistantMsg.Message, active.Ctx)
							if err != nil {
								onError(fmt.Errorf("failed to get exec status: %v", err), false, assistantMsg.Id, convoCommitMsg)
								errCh <- err
								return
							}

							log.Printf("Should continue: %v\n", shouldContinue)

							errCh <- nil
						}()

						for i := 0; i < 2; i++ {
							err = <-errCh
							if err != nil {
								return err
							}
						}

						log.Println("Comitting reply message and description")

						err = db.GitAddAndCommit(currentOrgId, planId, branch, convoCommitMsg)
						if err != nil {
							onError(fmt.Errorf("failed to commit: %v", err), false, assistantMsg.Id, convoCommitMsg)
							return err
						}

						log.Println("Assistant reply and description committed")

						return nil
					}()

					if err != nil {
						return
					}

					log.Println("Sending active.CurrentReplyDoneCh <- true")

					active.CurrentReplyDoneCh <- true

					log.Println("Resetting active.CurrentReplyDoneCh")

					UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
						ap.CurrentStreamingReplyId = ""
						ap.CurrentReplyDoneCh = nil
					})

					if req.AutoContinue && shouldContinue {
						log.Println("Auto continue plan")
						// continue plan
						execTellPlan(client, plan, branch, auth, req, iteration+1, "", false)
					} else {
						var buildFinished bool
						UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
							buildFinished = ap.BuildFinished()
							ap.RepliesFinished = true
						})

						log.Printf("Won't continue plan. Build finished: %v\n", buildFinished)

						if buildFinished {
							active.Stream(shared.StreamMessage{
								Type: shared.StreamMessageFinished,
							})
						} else {
							log.Println("Sending RepliesFinished stream message")
							active.Stream(shared.StreamMessage{
								Type: shared.StreamMessageRepliesFinished,
							})
						}
					}

					return
				}

				chunksReceived++
				delta := choice.Delta
				content := delta.Content

				if missingFileResponse != "" {
					if chunksReceived < 3 && strings.Contains(content, "```") {
						// received closing triple backticks in first 3 chunks after missing file response
						// means this is a redundant start of a new file block, so just ignore it

						maybeRedundantBacktickContent += content
						continue
					} else if maybeRedundantBacktickContent != "" {
						if strings.Contains(content, "\n") {
							maybeRedundantBacktickContent = ""
						} else {
							maybeRedundantBacktickContent += content
							continue
						}
					}
				}

				UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
					ap.CurrentReplyContent += content
					ap.NumTokens++
				})

				// log.Printf("Sending stream msg: %s", content)
				active.Stream(shared.StreamMessage{
					Type:       shared.StreamMessageReply,
					ReplyChunk: content,
				})

				// log.Printf("content: %s\n", content)

				replyParser.AddChunk(content, true)
				res := replyParser.Read()
				files := res.Files
				fileContents := res.FileContents
				replyNumTokens = res.TotalTokens
				currentFile := res.CurrentFilePath

				// log.Printf("currentFile: %s\n", currentFile)
				// log.Println("files:")
				// spew.Dump(files)

				// Handle file that is present in project paths but not in context
				// Prompt user for what to do on the client side, stop the stream, and wait for user response before proceeding
				if currentFile != "" &&
					active.ContextsByPath[currentFile] == nil &&
					req.ProjectPaths[currentFile] && !active.AllowOverwritePaths[currentFile] {
					log.Printf("Attempting to overwrite a file that isn't in context: %s\n", currentFile)

					// attempting to overwrite a file that isn't in context
					// we will stop the stream and ask the user what to do
					err := db.SetPlanStatus(planId, branch, shared.PlanStatusPrompting, "")

					if err != nil {
						log.Printf("Error setting plan %s status to prompting: %v\n", planId, err)
						active.StreamDoneCh <- &shared.ApiError{
							Type:   shared.ApiErrorTypeOther,
							Status: http.StatusInternalServerError,
							Msg:    "Error setting plan status to prompting",
						}
						return
					}

					log.Printf("Prompting user for missing file: %s\n", currentFile)

					active.Stream(shared.StreamMessage{
						Type:            shared.StreamMessagePromptMissingFile,
						MissingFilePath: currentFile,
					})

					log.Printf("Stopping stream for missing file: %s\n", currentFile)

					// log.Printf("Current reply content: %s\n", active.CurrentReplyContent)

					// stop stream for now
					active.CancelModelStreamFn()

					log.Printf("Stopped stream for missing file: %s\n", currentFile)

					// wait for user response to come in
					userChoice := <-active.MissingFileResponseCh

					log.Printf("User choice for missing file: %s\n", userChoice)

					active.ResetModelCtx()

					log.Println("Continuing stream")

					// continue plan
					execTellPlan(
						client,
						plan,
						branch,
						auth,
						req,
						iteration, // keep the same iteration
						userChoice,
						false,
					)
					return
				}

				// log.Println("Content:", content)
				// log.Println("Current reply content:", active.CurrentReplyContent)
				// log.Println("Current file:", currentFile)
				// log.Println("files:")
				// spew.Dump(files)
				// log.Println("replyFiles:")
				// spew.Dump(replyFiles)

				if len(files) > len(replyFiles) {
					log.Printf("%d new files\n", len(files)-len(replyFiles))

					for i, file := range files {
						if i < len(replyFiles) {
							continue
						}

						log.Printf("New file: %s\n", file)
						if req.BuildMode == shared.BuildModeAuto {
							log.Printf("Queuing build for %s\n", file)
							queueBuilds(client, currentOrgId, currentUserId, planId, branch, []*types.ActiveBuild{{
								ReplyId:      replyId,
								ReplyContent: active.CurrentReplyContent,
								FileContent:  fileContents[i],
								Path:         file,
							}})
						}
						replyFiles = append(replyFiles, file)
						UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
							ap.Files = append(ap.Files, file)
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

func summarizeConvo(client *openai.Client, params summarizeConvoParams) error {
	log.Printf("summarizeConvo: Called for plan ID %s on branch %s\n", params.planId, params.branch)
	log.Printf("summarizeConvo: Starting summarizeConvo for planId: %s\n", params.planId)
	planId := params.planId
	branch := params.branch
	convo := params.convo
	summaries := params.summaries
	promptMessage := params.promptMessage
	currentOrgId := params.currentOrgId

	log.Println("Generating plan summary for planId:", planId)

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

	summary, err := model.PlanSummary(client, model.PlanSummaryParams{
		Conversation:                summaryMessages,
		LatestConvoMessageId:        latestMessageId,
		LatestConvoMessageCreatedAt: latestMessageSummarizedAt,
		NumMessages:                 numMessagesSummarized + 1,
		OrgId:                       currentOrgId,
		PlanId:                      planId,
	})

	if err != nil {
		log.Printf("summarizeConvo: Error generating plan summary for plan %s: %v\n", params.planId, err)
		return err
	}

	log.Printf("summarizeConvo: Summary generated and stored for plan %s\n", params.planId)

	err = db.StoreSummary(summary)

	if err != nil {
		log.Printf("Error storing plan summary for plan %s: %v\n", planId, err)
		return err
	}

	return nil
}
