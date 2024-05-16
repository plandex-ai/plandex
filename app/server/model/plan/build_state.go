package plan

import (
	"plandex-server/db"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

const MaxBuildStreamErrorRetries = 3 // uses semi-exponential backoff so be careful with this

const FixSyntaxRetries = 3
const FixSyntaxEpochs = 3

type activeBuildStreamState struct {
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
	filePath           string
	convoMessageId     string
	build              *db.PlanBuild
	currentPlanState   *shared.CurrentPlanState
	activeBuild        *types.ActiveBuild
	preBuildState      string
	lineNumsNumRetry   int
	verifyFileNumRetry int
	fixFileNumRetry    int

	syntaxNumRetry int
	syntaxNumEpoch int

	isFixingSyntax bool
	isFixingOther  bool

	streamedChangesWithLineNums []*shared.StreamedChangeWithLineNums
	updated                     string

	verificationErrors string
	syntaxErrors       []string

	isNewFile bool
}
