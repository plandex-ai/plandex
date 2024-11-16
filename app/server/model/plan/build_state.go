package plan

import (
	"plandex-server/db"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
	sitter "github.com/smacker/go-tree-sitter"
)

const MaxBuildErrorRetries = 3 // uses semi-exponential backoff so be careful with this

const FixSyntaxRetries = 2
const FixSyntaxEpochs = 2

type activeBuildStreamState struct {
	tellState     *activeTellStreamState
	clients       map[string]*openai.Client
	auth          *types.ServerAuth
	currentOrgId  string
	currentUserId string
	plan          *db.Plan
	branch        string
	settings      *shared.PlanSettings
	modelContext  []*db.Context
	convo         []*db.ConvoMessage
}

type activeBuildStreamFileState struct {
	*activeBuildStreamState
	filePath                   string
	convoMessageId             string
	build                      *db.PlanBuild
	currentPlanState           *shared.CurrentPlanState
	activeBuild                *types.ActiveBuild
	preBuildState              string
	parser                     *sitter.Parser
	language                   string
	preBuildStateSyntaxInvalid bool

	structuredEditNumRetry int
	expandRefsNumRetry     int
	lineNumsNumRetry       int
	verifyFileNumRetry     int
	fixFileNumRetry        int

	syntaxNumRetry int
	syntaxNumEpoch int

	isFixingSyntax bool
	isFixingOther  bool

	streamedChangesWithLineNums []*shared.StreamedChangeWithLineNums
	updated                     string

	verificationErrors string
	syntaxErrors       []string

	isNewFile bool

	inputTokens int
}
