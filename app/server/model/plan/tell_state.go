package plan

import (
	"plandex-server/db"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

type activeTellStreamState struct {
	clients                map[string]*openai.Client
	req                    *shared.TellPlanRequest
	auth                   *types.ServerAuth
	currentOrgId           string
	currentUserId          string
	plan                   *db.Plan
	branch                 string
	iteration              int
	replyId                string
	modelContext           []*db.Context
	convo                  []*db.ConvoMessage
	missingFileResponse    shared.RespondMissingFileChoice
	summaries              []*db.ConvoSummary
	summarizedToMessageId  string
	latestSummaryTokens    int
	userPrompt             string
	promptMessage          *openai.ChatCompletionMessage
	replyParser            *types.ReplyParser
	replyNumTokens         int
	messages               []openai.ChatCompletionMessage
	tokensBeforeConvo      int
	settings               *shared.PlanSettings
	currentReplyNumRetries int
}
