package plan

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/model"
	"plandex-server/model/prompts"
	"plandex-server/notify"
	"plandex-server/types"
	"time"

	shared "plandex-shared"

	"github.com/davecgh/go-spew/spew"
	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) addConversationMessages() bool {
	summaries := state.summaries
	tokensBeforeConvo := state.tokensBeforeConvo
	active := GetActivePlan(state.plan.Id, state.branch)

	convo := []*db.ConvoMessage{}
	for _, msg := range state.convo {
		if state.skipConvoMessages != nil && state.skipConvoMessages[msg.Id] {
			continue
		}
		convo = append(convo, msg)
	}

	if active == nil {
		log.Println("summarizeMessagesIfNeeded - Active plan not found")
		return false
	}

	conversationTokens := 0
	tokensUpToTimestamp := make(map[int64]int)
	convoMessagesById := make(map[string]*db.ConvoMessage)
	for _, convoMessage := range convo {
		conversationTokens += convoMessage.Tokens + model.TokensPerMessage + model.TokensPerName
		timestamp := convoMessage.CreatedAt.UnixNano() / int64(time.Millisecond)
		tokensUpToTimestamp[timestamp] = conversationTokens
		convoMessagesById[convoMessage.Id] = convoMessage
		// log.Printf("Timestamp: %s | Tokens: %d | Total: %d | conversationTokens\n", convoMessage.Timestamp, convoMessage.Tokens, conversationTokens)
	}

	log.Printf("Conversation tokens: %d\n", conversationTokens)
	log.Printf("Max conversation tokens: %d\n", state.settings.GetPlannerMaxConvoTokens())

	// log.Println("Tokens up to timestamp:")
	// spew.Dump(tokensUpToTimestamp)

	log.Printf("Total tokens: %d\n", tokensBeforeConvo+conversationTokens)
	log.Printf("Max tokens: %d\n", state.settings.GetPlannerEffectiveMaxTokens())

	var summary *db.ConvoSummary
	if (tokensBeforeConvo+conversationTokens) > state.settings.GetPlannerEffectiveMaxTokens() ||
		conversationTokens > state.settings.GetPlannerMaxConvoTokens() {
		log.Println("Token limit exceeded. Attempting to reduce via conversation summary.")

		// log.Printf("(tokensBeforeConvo+conversationTokens) > state.settings.GetPlannerEffectiveMaxTokens(): %v\n", (tokensBeforeConvo+conversationTokens) > state.settings.GetPlannerEffectiveMaxTokens())
		// log.Printf("conversationTokens > state.settings.GetPlannerMaxConvoTokens(): %v\n", conversationTokens > state.settings.GetPlannerMaxConvoTokens())

		log.Printf("Num summaries: %d\n", len(summaries))

		// token limit exceeded after adding conversation
		// get summary for as much as the conversation as necessary to stay under the token limit
		for _, s := range summaries {
			timestamp := s.LatestConvoMessageCreatedAt.UnixNano() / int64(time.Millisecond)

			tokens, ok := tokensUpToTimestamp[timestamp]

			log.Printf("Last message timestamp: %d | found: %v\n", timestamp, ok)
			log.Printf("Tokens up to timestamp: %d\n", tokens)

			if !ok {
				// try a fallback by id instead of timestamp, in case timestamp rounding caused it to be missing
				convoMessage, ok := convoMessagesById[s.LatestConvoMessageId]

				if ok {
					timestamp = convoMessage.CreatedAt.UnixNano() / int64(time.Millisecond)
					tokens, ok = tokensUpToTimestamp[timestamp]
				}

				if !ok {
					// instead of erroring here as we did previously, we'll just log and continue
					// if no summary is found, we still handle it as an error below
					// but this way we don't error out completely for  a single detached summary

					log.Println("conversation summary timestamp not found in conversation")
					log.Println("timestamp:", timestamp)

					// log.Println("Conversation summary:")
					// spew.Dump(s)

					log.Println("tokensUpToTimestamp:")
					log.Println(spew.Sdump(tokensUpToTimestamp))

					go notify.NotifyErr(notify.SeverityInfo, fmt.Errorf("conversation summary timestamp not found in conversation"))

					continue
				}
			}

			updatedConversationTokens := (conversationTokens - tokens) + s.Tokens
			savedTokens := conversationTokens - updatedConversationTokens

			log.Printf("Conversation summary tokens: %d\n", tokens)
			log.Printf("Updated conversation tokens: %d\n", updatedConversationTokens)
			log.Printf("Saved tokens: %d\n", savedTokens)

			if updatedConversationTokens <= state.settings.GetPlannerMaxConvoTokens() &&
				(tokensBeforeConvo+updatedConversationTokens) <= state.settings.GetPlannerEffectiveMaxTokens() {
				log.Printf("Summarizing up to %s | saving %d tokens\n", s.LatestConvoMessageCreatedAt.Format(time.RFC3339), savedTokens)
				summary = s
				conversationTokens = updatedConversationTokens
				break
			}
		}

		if summary == nil && tokensBeforeConvo+conversationTokens > state.settings.GetPlannerEffectiveMaxTokens() {
			err := errors.New("couldn't get under token limit with conversation summary")
			log.Printf("Error: %v\n", err)
			go notify.NotifyErr(notify.SeverityInfo, fmt.Errorf("couldn't get under token limit with conversation summary"))

			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Exceeded token limit",
			}
			return false
		}
	}

	var latestSummary *db.ConvoSummary
	if len(summaries) > 0 {
		latestSummary = summaries[len(summaries)-1]
	}

	if summary == nil {
		for _, convoMessage := range convo {
			// this gets added later in tell_exec.go
			if state.promptConvoMessage != nil && convoMessage.Id == state.promptConvoMessage.Id {
				continue
			}

			state.messages = append(state.messages, types.ExtendedChatMessage{
				Role: openai.ChatMessageRoleUser,
				Content: []types.ExtendedChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: convoMessage.Message,
					},
				},
			})

			// add the latest summary as a conversation message if this is the last message summarized, in order to reinforce the current state of the plan to the model
			if latestSummary != nil && convoMessage.Id == latestSummary.LatestConvoMessageId {
				state.messages = append(state.messages, types.ExtendedChatMessage{
					Role: openai.ChatMessageRoleAssistant,
					Content: []types.ExtendedChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeText,
							Text: latestSummary.Summary,
						},
					},
				})
			}
		}
	} else {
		if (tokensBeforeConvo + conversationTokens) > state.settings.GetPlannerEffectiveMaxTokens() {
			go notify.NotifyErr(notify.SeverityError, fmt.Errorf("token limit still exceeded after summarizing conversation"))

			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Token limit still exceeded after summarizing conversation",
			}
			return false
		}
		state.summarizedToMessageId = summary.LatestConvoMessageId
		state.messages = append(state.messages, types.ExtendedChatMessage{
			Role: openai.ChatMessageRoleAssistant,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: summary.Summary,
				},
			},
		})

		// add messages after the last message in the summary
		for _, convoMessage := range convo {
			// this gets added later in tell_exec.go
			if state.promptConvoMessage != nil && convoMessage.Id == state.promptConvoMessage.Id {
				continue
			}

			if convoMessage.CreatedAt.After(summary.LatestConvoMessageCreatedAt) {
				state.messages = append(state.messages, types.ExtendedChatMessage{
					Role: openai.ChatMessageRoleUser,
					Content: []types.ExtendedChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeText,
							Text: convoMessage.Message,
						},
					},
				})

				// add the latest summary as a conversation message if this is the last message summarized, in order to reinforce the current state of the plan to the model
				if latestSummary != nil && convoMessage.Id == latestSummary.LatestConvoMessageId {
					state.messages = append(state.messages, types.ExtendedChatMessage{
						Role: openai.ChatMessageRoleAssistant,
						Content: []types.ExtendedChatMessagePart{
							{
								Type: openai.ChatMessagePartTypeText,
								Text: latestSummary.Summary,
							},
						},
					})
				}
			}
		}
	}

	return true
}

type summarizeConvoParams struct {
	auth                  *types.ServerAuth
	plan                  *db.Plan
	branch                string
	convo                 []*db.ConvoMessage
	summaries             []*db.ConvoSummary
	userPrompt            string
	currentReply          string
	currentReplyNumTokens int
	currentOrgId          string
	modelPackName         string
}

func summarizeConvo(clients map[string]model.ClientInfo, config shared.ModelRoleConfig, params summarizeConvoParams, ctx context.Context) *shared.ApiError {
	plan := params.plan
	planId := plan.Id
	log.Printf("summarizeConvo: Called for plan ID %s on branch %s\n", planId, params.branch)
	log.Printf("summarizeConvo: Starting summarizeConvo for planId: %s\n", planId)

	branch := params.branch
	convo := params.convo
	summaries := params.summaries
	userPrompt := params.userPrompt
	currentReply := params.currentReply
	active := GetActivePlan(planId, branch)

	if active == nil {
		log.Printf("Active plan not found for plan ID %s and branch %s\n", planId, branch)

		return &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    fmt.Sprintf("active plan not found for plan ID %s and branch %s", planId, branch),
		}
	}

	log.Println("Generating plan summary for planId:", planId)

	// log.Printf("planId: %s\n", planId)
	// log.Printf("convo: ")
	// spew.Dump(convo)
	// log.Printf("summaries: ")
	// spew.Dump(summaries)
	// log.Printf("promptMessage: ")
	// spew.Dump(promptMessage)
	// log.Printf("currentOrgId: %s\n", currentOrgId)

	var summaryMessages []*types.ExtendedChatMessage
	var latestSummary *db.ConvoSummary
	var numMessagesSummarized int = 0
	var latestMessageSummarizedAt time.Time
	var latestMessageId string
	if len(summaries) > 0 {
		latestSummary = summaries[len(summaries)-1]
		numMessagesSummarized = latestSummary.NumMessages
	}

	// log.Println("Generating plan summary - latest summary:")
	// spew.Dump(latestSummary)

	// log.Println("Generating plan summary - convo:")
	// spew.Dump(convo)

	numTokens := 0

	if latestSummary == nil {
		for _, convoMessage := range convo {
			summaryMessages = append(summaryMessages, &types.ExtendedChatMessage{
				Role: openai.ChatMessageRoleUser,
				Content: []types.ExtendedChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: convoMessage.Message,
					},
				},
			})
			latestMessageId = convoMessage.Id
			latestMessageSummarizedAt = convoMessage.CreatedAt
			numMessagesSummarized++
			numTokens += convoMessage.Tokens + model.TokensPerMessage + model.TokensPerName
		}
	} else {
		summaryMessages = append(summaryMessages, &types.ExtendedChatMessage{
			Role: openai.ChatMessageRoleAssistant,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: latestSummary.Summary,
				},
			},
		})

		numTokens += latestSummary.Tokens + model.TokensPerMessage + model.TokensPerName

		var found bool
		for _, convoMessage := range convo {
			if convoMessage.Id == latestSummary.LatestConvoMessageId {
				found = true
				continue
			}
			if found {
				summaryMessages = append(summaryMessages, &types.ExtendedChatMessage{
					Role: openai.ChatMessageRoleUser,
					Content: []types.ExtendedChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeText,
							Text: convoMessage.Message,
						},
					},
				})
				numMessagesSummarized++
				numTokens += convoMessage.Tokens + model.TokensPerMessage + model.TokensPerName
			}
		}

		latestConvoMessage := convo[len(convo)-1]
		latestMessageId = latestConvoMessage.Id
		latestMessageSummarizedAt = latestConvoMessage.CreatedAt
	}

	log.Println("generating summary - latestMessageId:", latestMessageId)
	log.Println("generating summary - latestMessageSummarizedAt:", latestMessageSummarizedAt)

	if userPrompt != "" {
		if userPrompt != prompts.UserContinuePrompt && userPrompt != prompts.AutoContinuePlanningPrompt && userPrompt != prompts.AutoContinueImplementationPrompt {
			summaryMessages = append(summaryMessages, &types.ExtendedChatMessage{
				Role: openai.ChatMessageRoleUser,
				Content: []types.ExtendedChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: userPrompt,
					},
				},
			})

			tokens := shared.GetNumTokensEstimate(userPrompt)
			numTokens += tokens + model.TokensPerMessage + model.TokensPerName
		}
	}

	if currentReply != "" {
		summaryMessages = append(summaryMessages, &types.ExtendedChatMessage{
			Role: openai.ChatMessageRoleAssistant,
			Content: []types.ExtendedChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: currentReply,
				},
			},
		})

		numTokens += params.currentReplyNumTokens + model.TokensPerMessage + model.TokensPerName
	}

	log.Printf("Calling model for plan summary. Summarizing %d messages\n", len(summaryMessages))

	// log.Println("Generating summary - summary messages:")
	// spew.Dump(summaryMessages)

	// latestSummaryCh := make(chan *db.ConvoSummary, 1)
	// active.LatestSummaryCh = latestSummaryCh

	summary, apiErr := model.PlanSummary(clients, config, model.PlanSummaryParams{
		Conversation:                summaryMessages,
		ConversationNumTokens:       numTokens,
		LatestConvoMessageId:        latestMessageId,
		LatestConvoMessageCreatedAt: latestMessageSummarizedAt,
		NumMessages:                 numMessagesSummarized,
		Auth:                        params.auth,
		Plan:                        plan,
		ModelPackName:               params.modelPackName,
		ModelStreamId:               active.ModelStreamId,
		SessionId:                   active.SessionId,
	}, ctx)

	if apiErr != nil {
		log.Printf("summarizeConvo: Error generating plan summary for plan %s: %v\n", planId, apiErr)
		return apiErr
	}

	log.Printf("summarizeConvo: Summary generated and stored for plan %s\n", planId)

	// log.Println("Generated summary:")
	// spew.Dump(summary)

	err := db.StoreSummary(summary)

	if err != nil {
		log.Printf("Error storing plan summary for plan %s: %v\n", planId, err)
		return &shared.ApiError{
			Type:   shared.ApiErrorTypeOther,
			Status: http.StatusInternalServerError,
			Msg:    fmt.Sprintf("error storing plan summary for plan %s: %v", planId, err),
		}
	}

	// latestSummaryCh <- summary

	return nil
}
