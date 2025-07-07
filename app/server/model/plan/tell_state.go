package plan

import (
	"plandex-server/db"
	"plandex-server/model"
	"plandex-server/types"
	"time"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

type activeTellStreamState struct {
	activePlan            *types.ActivePlan
	modelStreamId         string
	clients               map[string]model.ClientInfo
	authVars              map[string]string
	req                   *shared.TellPlanRequest
	auth                  *types.ServerAuth
	currentOrgId          string
	currentUserId         string
	orgUserConfig         *shared.OrgUserConfig
	plan                  *db.Plan
	branch                string
	iteration             int
	replyId               string
	modelContext          []*db.Context
	hasContextMap         bool
	contextMapEmpty       bool
	convo                 []*db.ConvoMessage
	promptConvoMessage    *db.ConvoMessage
	currentPlanState      *shared.CurrentPlanState
	missingFileResponse   shared.RespondMissingFileChoice
	summaries             []*db.ConvoSummary
	summarizedToMessageId string
	latestSummaryTokens   int
	userPrompt            string
	promptMessage         *openai.ChatCompletionMessage
	replyParser           *types.ReplyParser
	replyNumTokens        int
	messages              []types.ExtendedChatMessage
	tokensBeforeConvo     int
	totalRequestTokens    int
	settings              *shared.PlanSettings
	subtasks              []*db.Subtask
	currentSubtask        *db.Subtask
	hasAssistantReply     bool
	currentStage          shared.CurrentStage
	chunkProcessor        *chunkProcessor
	generationId          string

	requestStartedAt time.Time
	firstTokenAt     time.Time
	originalReq      *types.ExtendedChatCompletionRequest
	modelConfig      *shared.ModelRoleConfig
	baseModelConfig  *shared.BaseModelConfig
	fallbackRes      shared.FallbackResult

	skipConvoMessages map[string]bool

	manualStop []string

	numErrorRetry       int
	numFallbackRetry    int
	modelErr            *shared.ModelError
	noCacheSupportErr   bool
	didProviderFallback bool
}

type chunkProcessor struct {
	replyOperations                 []*shared.Operation
	chunksReceived                  int
	maybeRedundantOpeningTagContent string
	fileOpen                        bool
	contentBuffer                   string
	awaitingBlockOpeningTag         bool
	awaitingBlockClosingTag         bool
	awaitingOpClosingTag            bool
	awaitingBackticks               bool
}
