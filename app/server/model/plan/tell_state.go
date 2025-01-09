package plan

import (
	"plandex-server/db"
	"plandex-server/types"
	"strings"

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
	hasContextMap          bool
	contextMapEmpty        bool
	convo                  []*db.ConvoMessage
	promptConvoMessage     *db.ConvoMessage
	currentPlanState       *shared.CurrentPlanState
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
	totalRequestTokens     int
	settings               *shared.PlanSettings
	currentReplyNumRetries int
	subtasks               []*db.Subtask
	currentSubtask         *db.Subtask

	isContextStage        bool
	isPlanningStage       bool
	isImplementationStage bool

	chunkProcessor *chunkProcessor
}

type chunkProcessor struct {
	replyOperations                 []*shared.Operation
	chunksReceived                  int
	maybeRedundantOpeningTagContent string
	fileOpen                        bool
	contentBuffer                   *strings.Builder
	awaitingBlockOpeningTag         bool
	awaitingBlockClosingTag         bool
	awaitingOpClosingTag            bool
	awaitingBackticks               bool
}
