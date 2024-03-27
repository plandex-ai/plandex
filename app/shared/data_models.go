package shared

import (
	"time"

	"github.com/sashabaranov/go-openai"
)

type Org struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	IsPending bool   `json:"isPending"`
}

type User struct {
	Id               string `json:"id"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	IsTrial          bool   `json:"isTrial"`
	NumNonDraftPlans int    `json:"numNonDraftPlans"`
}

type OrgUser struct {
	OrgId     string `json:"orgId"`
	UserId    string `json:"userId"`
	OrgRoleId string `json:"orgRoleId"`
}

type Invite struct {
	Id         string     `json:"id"`
	OrgId      string     `json:"orgId"`
	Email      string     `json:"email"`
	Name       string     `json:"name"`
	OrgRoleId  string     `json:"orgRoleId"`
	InviterId  string     `json:"inviterId"`
	InviteeId  *string    `json:"inviteeId"`
	AcceptedAt *time.Time `json:"acceptedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type Project struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Plan struct {
	Id              string     `json:"id"`
	OwnerId         string     `json:"ownerId"`
	ProjectId       string     `json:"projectId"`
	Name            string     `json:"name"`
	SharedWithOrgAt *time.Time `json:"sharedWithOrgAt,omitempty"`
	TotalReplies    int        `json:"totalReplies"`
	ActiveBranches  int        `json:"activeBranches"`
	ArchivedAt      *time.Time `json:"archivedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type Branch struct {
	Id              string     `json:"id"`
	PlanId          string     `json:"planId"`
	OwnerId         string     `json:"ownerId"`
	ParentBranchId  *string    `json:"parentBranchId"`
	Name            string     `json:"name"`
	Status          PlanStatus `json:"status"`
	ContextTokens   int        `json:"contextTokens"`
	ConvoTokens     int        `json:"convoTokens"`
	SharedWithOrgAt *time.Time `json:"sharedWithOrgAt,omitempty"`
	ArchivedAt      *time.Time `json:"archivedAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type ContextType string

const (
	ContextFileType          ContextType = "file"
	ContextURLType           ContextType = "url"
	ContextNoteType          ContextType = "note"
	ContextDirectoryTreeType ContextType = "directory tree"
	ContextPipedDataType     ContextType = "piped data"
)

type Context struct {
	Id              string      `json:"id"`
	OwnerId         string      `json:"ownerId"`
	ContextType     ContextType `json:"contextType"`
	Name            string      `json:"name"`
	Url             string      `json:"url"`
	FilePath        string      `json:"file_path"`
	Sha             string      `json:"sha"`
	NumTokens       int         `json:"numTokens"`
	Body            string      `json:"body,omitempty"`
	ForceSkipIgnore bool        `json:"forceSkipIgnore"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
}

type ConvoMessage struct {
	Id        string    `json:"id"`
	UserId    string    `json:"userId"`
	Role      string    `json:"role"`
	Tokens    int       `json:"tokens"`
	Num       int       `json:"num"`
	Message   string    `json:"message"`
	Stopped   bool      `json:"stopped"`
	CreatedAt time.Time `json:"createdAt"`
}

type ConvoSummary struct {
	Id                          string    `json:"id"`
	LatestConvoMessageCreatedAt time.Time `json:"latestConvoMessageCreatedAt"`
	LatestConvoMessageId        string    `json:"lastestConvoMessageId"`
	Summary                     string    `json:"summary"`
	Tokens                      int       `json:"tokens"`
	NumMessages                 int       `json:"numMessages"`
	CreatedAt                   time.Time `json:"createdAt"`
}

type ConvoMessageDescription struct {
	Id                    string          `json:"id"`
	ConvoMessageId        string          `json:"convoMessageId"`
	SummarizedToMessageId string          `json:"summarizedToMessageId"`
	MadePlan              bool            `json:"madePlan"`
	CommitMsg             string          `json:"commitMsg"`
	Files                 []string        `json:"files"`
	DidBuild              bool            `json:"didBuild"`
	BuildPathsInvalidated map[string]bool `json:"buildPathsInvalidated"`
	Error                 string          `json:"error"`
	AppliedAt             *time.Time      `json:"appliedAt,omitempty"`
	CreatedAt             time.Time       `json:"createdAt"`
	UpdatedAt             time.Time       `json:"updatedAt"`
}

type PlanBuild struct {
	Id             string    `json:"id"`
	ConvoMessageId string    `json:"convoMessageId"`
	FilePath       string    `json:"filePath"`
	Error          string    `json:"error"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Replacement struct {
	Id             string          `json:"id"`
	Old            string          `json:"old"`
	New            string          `json:"new"`
	Failed         bool            `json:"failed"`
	RejectedAt     *time.Time      `json:"rejectedAt,omitempty"`
	StreamedChange *StreamedChange `json:"streamedChange"`
}

type PlanFileResult struct {
	Id             string         `json:"id"`
	ConvoMessageId string         `json:"convoMessageId"`
	PlanBuildId    string         `json:"planBuildId"`
	Path           string         `json:"path"`
	Content        string         `json:"content"`
	AnyFailed      bool           `json:"anyFailed"`
	AppliedAt      *time.Time     `json:"appliedAt,omitempty"`
	RejectedAt     *time.Time     `json:"rejectedAt,omitempty"`
	Replacements   []*Replacement `json:"replacements"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type CurrentPlanFiles struct {
	Files           map[string]string    `json:"files"`
	UpdatedAtByPath map[string]time.Time `json:"updatedAtByPath"`
}

type PlanFileResultsByPath map[string][]*PlanFileResult

type PlanResult struct {
	SortedPaths        []string                  `json:"sortedPaths"`
	FileResultsByPath  PlanFileResultsByPath     `json:"fileResultsByPath"`
	Results            []*PlanFileResult         `json:"results"`
	ReplacementsByPath map[string][]*Replacement `json:"replacementsByPath"`
}

type CurrentPlanState struct {
	PlanResult               *PlanResult                `json:"planResult"`
	CurrentPlanFiles         *CurrentPlanFiles          `json:"currentPlanFiles"`
	ConvoMessageDescriptions []*ConvoMessageDescription `json:"convoMessageDescriptions"`
	ContextsByPath           map[string]*Context        `json:"contextsByPath"`
}

type OrgRole struct {
	Id          string `json:"id"`
	IsDefault   bool   `json:"isDefault"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type BaseModelConfig struct {
	Provider  ModelProvider `json:"provider"`
	BaseUrl   string        `json:"baseUrl"`
	ModelName string        `json:"modelName"`
	MaxTokens int           `json:"maxTokens"`
}

type PlannerModelConfig struct {
	MaxConvoTokens       int `json:"maxConvoTokens"`
	ReservedOutputTokens int `json:"maxOutputTokens"`
}

type TaskModelConfig struct {
	OpenAIResponseFormat *openai.ChatCompletionResponseFormat `json:"openAIResponseFormat"`
}

type ModelRoleConfig struct {
	Role            ModelRole       `json:"role"`
	BaseModelConfig BaseModelConfig `json:"baseModelConfig"`
	Temperature     float32         `json:"temperature"`
	TopP            float32         `json:"topP"`
}

type PlannerRoleConfig struct {
	ModelRoleConfig
	PlannerModelConfig
}

type TaskRoleConfig struct {
	ModelRoleConfig
	TaskModelConfig
}

type ModelSet struct {
	Planner     PlannerRoleConfig `json:"planner"`
	PlanSummary ModelRoleConfig   `json:"planSummary"`
	Builder     TaskRoleConfig    `json:"builder"`
	Namer       TaskRoleConfig    `json:"namer"`
	CommitMsg   TaskRoleConfig    `json:"commitMsg"`
	ExecStatus  TaskRoleConfig    `json:"execStatus"`
}

type ModelOverrides struct {
	MaxConvoTokens       *int `json:"maxConvoTokens"`
	MaxTokens            *int `json:"maxContextTokens"`
	ReservedOutputTokens *int `json:"maxOutputTokens"`
}

type PlanSettings struct {
	ModelOverrides ModelOverrides `json:"modelOverrides"`
	ModelSet       *ModelSet      `json:"modelSet"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}
