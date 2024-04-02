package plan

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/model"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func (state *activeTellStreamState) summarizeMessagesIfNeeded() bool {
	convo := state.convo
	summaries := state.summaries
	tokensBeforeConvo := state.tokensBeforeConvo

	active := GetActivePlan(state.plan.Id, state.branch)

	conversationTokens := 0
	tokensUpToTimestamp := make(map[int64]int)
	for _, convoMessage := range convo {
		conversationTokens += convoMessage.Tokens
		timestamp := convoMessage.CreatedAt.UnixNano() / int64(time.Millisecond)
		tokensUpToTimestamp[timestamp] = conversationTokens
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
				err := fmt.Errorf("conversation summary timestamp not found in conversation")
				log.Printf("Error: %v\n", err)

				log.Println("timestamp:", timestamp)

				// log.Println("Conversation summary:")
				// spew.Dump(s)

				log.Println("tokensUpToTimestamp:")
				log.Println(spew.Sdump(tokensUpToTimestamp))

				active.StreamDoneCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    "Conversation summary timestamp not found in conversation",
				}
				return false
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
			return false
		}
	}

	if summary == nil {
		for _, convoMessage := range convo {
			state.messages = append(state.messages, openai.ChatCompletionMessage{
				Role:    convoMessage.Role,
				Content: convoMessage.Message,
			})
		}
	} else {
		if (tokensBeforeConvo + summary.Tokens) > state.settings.GetPlannerEffectiveMaxTokens() {
			active.StreamDoneCh <- &shared.ApiError{
				Type:   shared.ApiErrorTypeOther,
				Status: http.StatusInternalServerError,
				Msg:    "Token limit still exceeded after summarizing conversation",
			}
			return false
		}
		state.summarizedToMessageId = summary.LatestConvoMessageId
		state.messages = append(state.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: summary.Summary,
		})

		// add messages after the last message in the summary
		for _, convoMessage := range convo {
			if convoMessage.CreatedAt.After(summary.LatestConvoMessageCreatedAt) {
				state.messages = append(state.messages, openai.ChatCompletionMessage{
					Role:    convoMessage.Role,
					Content: convoMessage.Message,
				})
			}
		}
	}

	return true
}

type summarizeConvoParams struct {
	planId        string
	branch        string
	convo         []*db.ConvoMessage
	summaries     []*db.ConvoSummary
	promptMessage *openai.ChatCompletionMessage
	currentOrgId  string
}

func summarizeConvo(client *openai.Client, config shared.ModelRoleConfig, params summarizeConvoParams, ctx context.Context) error {
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

	log.Printf("Calling model for plan summary. Summarizing %d messages\n", len(summaryMessages))

	summary, err := model.PlanSummary(client, config, model.PlanSummaryParams{
		Conversation:                summaryMessages,
		LatestConvoMessageId:        latestMessageId,
		LatestConvoMessageCreatedAt: latestMessageSummarizedAt,
		NumMessages:                 numMessagesSummarized + 1,
		OrgId:                       currentOrgId,
		PlanId:                      planId,
	}, ctx)

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
