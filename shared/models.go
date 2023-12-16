package shared

import "time"

type Org struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type User struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Project struct {
	Id               string `json:"id"`
	Name             string `json:"name"`
	LastActivePlanId string `json:"lastActivePlanId"`
}

type PlanStatus string

const (
	PlanStatusReplying   PlanStatus = "replying"
	PlanStatusDescribing PlanStatus = "describing"
	PlanStatusBuilding   PlanStatus = "building"
	PlanStatusFinished   PlanStatus = "finished"
	PlanStatusStopped    PlanStatus = "stopped"
	PlanStatusError      PlanStatus = "error"
)

type Plan struct {
	Id                 string     `json:"id"`
	CreatorId          string     `json:"creatorId"`
	Name               string     `json:"name"`
	Status             PlanStatus `json:"status"`
	ContextTokens      int        `json:"contextTokens"`
	ConvoTokens        int        `json:"convoTokens"`
	ConvoSummaryTokens int        `json:"convoSummaryTokens"`
	AppliedAt          *time.Time `json:"appliedAt,omitempty"`
	ArchivedAt         *time.Time `json:"archivedAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
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
	Id          string      `json:"id"`
	CreatorId   string      `json:"creatorId"`
	ContextType ContextType `json:"contextType"`
	Name        string      `json:"name"`
	Url         string      `json:"url"`
	FilePath    string      `json:"file_path"`
	Sha         string      `json:"sha"`
	NumTokens   int         `json:"numTokens"`
	Body        string      `json:"body,omitempty"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
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
	Id                    string    `json:"id"`
	ConvoMessageId        string    `json:"convoMessageId"`
	SummarizedToMessageId string    `json:"summarizedToMessageId"`
	MadePlan              bool      `json:"madePlan"`
	CommitMsg             string    `json:"commitMsg"`
	Files                 []string  `json:"files"`
	Error                 string    `json:"error"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type PlanBuild struct {
	Id             string    `json:"id"`
	ConvoMessageId string    `json:"convoMessageId"`
	Error          string    `json:"error"`
	ErrorPath      string    `json:"errorPath"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Replacement struct {
	Id         string     `json:"id"`
	Old        string     `json:"old"`
	New        string     `json:"new"`
	Summary    string     `json:"summary"`
	Failed     bool       `json:"failed"`
	RejectedAt *time.Time `json:"rejectedAt,omitempty"`
}

type PlanFileResult struct {
	Id           string         `json:"id"`
	PlanBuildId  string         `json:"planBuildId"`
	Path         string         `json:"path"`
	ContextSha   string         `json:"contextSha"`
	Content      string         `json:"content"`
	AnyFailed    bool           `json:"anyFailed"`
	AppliedAt    *time.Time     `json:"appliedAt,omitempty"`
	RejectedAt   *time.Time     `json:"rejectedAt,omitempty"`
	Replacements []*Replacement `json:"replacements"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

type CurrentPlanFiles struct {
	Files       map[string]string `json:"files"`
	ContextShas map[string]string `json:"contextShas"`
}

type PlanFileResultsByPath map[string][]*PlanFileResult

type PlanResult struct {
	SortedPaths        []string                  `json:"sortedPaths"`
	FileResultsByPath  PlanFileResultsByPath     `json:"fileResultsByPath"`
	ReplacementsByPath map[string][]*Replacement `json:"replacementsByPath"`
}

type CurrentPlanState struct {
	PlanResult             *PlanResult
	CurrentPlanFiles       *CurrentPlanFiles
	Contexts               []*Context
	ContextsByPath         map[string]*Context
	LatestBuildDescription *ConvoMessageDescription
}
