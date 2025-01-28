package shared

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/shopspring/decimal"
)

type Org struct {
	Id                 string `json:"id"`
	Name               string `json:"name"`
	IsTrial            bool   `json:"isTrial"`
	AutoAddDomainUsers bool   `json:"autoAddDomainUsers"`

	// optional cloud attributes
	IntegratedModelsMode bool                `json:"integratedModelsMode,omitempty"`
	CloudBillingFields   *CloudBillingFields `json:"cloudBillingFields,omitempty"`
}

type User struct {
	Id               string `json:"id"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	IsTrial          bool   `json:"isTrial"`
	NumNonDraftPlans int    `json:"numNonDraftPlans"`

	DefaultPlanConfig *PlanConfig `json:"defaultPlanConfig,omitempty"`
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
	Id              string      `json:"id"`
	OwnerId         string      `json:"ownerId"`
	ProjectId       string      `json:"projectId"`
	Name            string      `json:"name"`
	SharedWithOrgAt *time.Time  `json:"sharedWithOrgAt,omitempty"`
	TotalReplies    int         `json:"totalReplies"`
	ActiveBranches  int         `json:"activeBranches"`
	PlanConfig      *PlanConfig `json:"planConfig,omitempty"`
	ArchivedAt      *time.Time  `json:"archivedAt,omitempty"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
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
	ContextImageType         ContextType = "image"
	ContextMapType           ContextType = "map"
)

type FileMapBodies map[string]string

type Context struct {
	Id              string                `json:"id"`
	OwnerId         string                `json:"ownerId"`
	ContextType     ContextType           `json:"contextType"`
	Name            string                `json:"name"`
	Url             string                `json:"url"`
	FilePath        string                `json:"file_path"`
	Sha             string                `json:"sha"`
	NumTokens       int                   `json:"numTokens"`
	Body            string                `json:"body,omitempty"`
	BodySize        int64                 `json:"bodySize,omitempty"`
	ForceSkipIgnore bool                  `json:"forceSkipIgnore"`
	ImageDetail     openai.ImageURLDetail `json:"imageDetail,omitempty"`
	MapParts        FileMapBodies         `json:"mapParts,omitempty"`
	MapShas         map[string]string     `json:"mapShas,omitempty"`
	MapTokens       map[string]int        `json:"mapTokens,omitempty"`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
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

type OperationType string

const (
	OperationTypeFile   OperationType = "file"
	OperationTypeMove   OperationType = "move"
	OperationTypeRemove OperationType = "remove"
	OperationTypeReset  OperationType = "reset"
)

type Operation struct {
	Type        OperationType
	Path        string
	Destination string
	Content     string
	Description string
	ReplyBefore string
	NumTokens   int
}

func (o *Operation) Name() string {
	res := string(o.Type) + " | " + o.Path
	if o.Destination != "" {
		res += " → " + o.Destination
	}
	return res
}

type ConvoMessageDescription struct {
	Id                    string `json:"id"`
	ConvoMessageId        string `json:"convoMessageId"`
	SummarizedToMessageId string `json:"summarizedToMessageId"`
	MadePlan              bool   `json:"madePlan"`
	CommitMsg             string `json:"commitMsg"`
	// Files                 []string        `json:"files"`
	Operations            []*Operation    `json:"operations"`
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
	Id                    string                             `json:"id"`
	Old                   string                             `json:"old"`
	Summary               string                             `json:"summary"`
	EntireFile            bool                               `json:"entireFile"`
	New                   string                             `json:"new"`
	Failed                bool                               `json:"failed"`
	RejectedAt            *time.Time                         `json:"rejectedAt,omitempty"`
	StreamedChange        *StreamedChangeWithLineNums        `json:"streamedChange"`
	StreamedChangeUpdated *StreamedChangeWithLineNumsUpdated `json:"streamedChangeUpdated"`
}

func (r *Replacement) GetSummary() string {
	if r.Summary != "" {
		return r.Summary
	}
	if r.StreamedChange != nil {
		return r.StreamedChange.Summary
	}
	return ""
}

type PlanFileResult struct {
	Id                  string         `json:"id"`
	TypeVersion         int            `json:"typeVersion"`
	ReplaceWithLineNums bool           `json:"replaceWithLineNums"`
	ConvoMessageId      string         `json:"convoMessageId"`
	PlanBuildId         string         `json:"planBuildId"`
	Path                string         `json:"path"`
	Content             string         `json:"content"`
	AnyFailed           bool           `json:"anyFailed"`
	AppliedAt           *time.Time     `json:"appliedAt,omitempty"`
	RejectedAt          *time.Time     `json:"rejectedAt,omitempty"`
	Replacements        []*Replacement `json:"replacements"`

	RemovedFile bool `json:"removedFile"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CurrentPlanFiles struct {
	Files           map[string]string    `json:"files"`
	Removed         map[string]bool      `json:"removedByPath"`
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

type ModelCompatibility struct {
	HasImageSupport bool `json:"hasImageSupport"`
}

type ModelOutputFormat string

const (
	ModelOutputFormatToolCallJson ModelOutputFormat = "tool-call-json"
	ModelOutputFormatXml          ModelOutputFormat = "xml"
)

type BaseModelConfig struct {
	Provider                   ModelProvider     `json:"provider"`
	CustomProvider             *string           `json:"customProvider,omitempty"`
	BaseUrl                    string            `json:"baseUrl"`
	ModelName                  string            `json:"modelName"`
	MaxTokens                  int               `json:"maxTokens"`
	ApiKeyEnvVar               string            `json:"apiKeyEnvVar"`
	PreferredModelOutputFormat ModelOutputFormat `json:"preferredModelOutputFormat"`

	SystemPromptDisabled bool `json:"systemPromptDisabled"`
	RoleParamsDisabled   bool `json:"roleParamsDisabled"`

	ModelCompatibility
}

type AvailableModel struct {
	Id string `json:"id"`
	BaseModelConfig
	Description                 string    `json:"description"`
	DefaultMaxConvoTokens       int       `json:"defaultMaxConvoTokens"`
	DefaultReservedOutputTokens int       `json:"defaultReservedOutputTokens"`
	CreatedAt                   time.Time `json:"createdAt"`
	UpdatedAt                   time.Time `json:"updatedAt"`
}

type PlannerModelConfig struct {
	MaxConvoTokens int `json:"maxConvoTokens"`
}

type ModelRoleConfig struct {
	Role                 ModelRole       `json:"role"`
	BaseModelConfig      BaseModelConfig `json:"baseModelConfig"`
	Temperature          float32         `json:"temperature"`
	TopP                 float32         `json:"topP"`
	ReservedOutputTokens int             `json:"reservedOutputTokens"`
}

func (m *ModelRoleConfig) GetReservedOutputTokens() int {
	if m.ReservedOutputTokens > 0 {
		return m.ReservedOutputTokens
	}
	return AvailableModelsByName[m.BaseModelConfig.ModelName].DefaultReservedOutputTokens
}

func (m *ModelRoleConfig) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch s := src.(type) {
	case []byte:
		return json.Unmarshal(s, m)
	case string:
		return json.Unmarshal([]byte(s), m)
	default:
		return fmt.Errorf("unsupported data type: %T", src)
	}
}

func (m ModelRoleConfig) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type PlannerRoleConfig struct {
	ModelRoleConfig
	PlannerModelConfig
}

func (p *PlannerRoleConfig) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch s := src.(type) {
	case []byte:
		return json.Unmarshal(s, p)
	case string:
		return json.Unmarshal([]byte(s), p)
	default:
		return fmt.Errorf("unsupported data type: %T", src)
	}
}

func (p PlannerRoleConfig) Value() (driver.Value, error) {
	return json.Marshal(p)
}

type ModelPack struct {
	Id               string            `json:"id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Planner          PlannerRoleConfig `json:"planner"`
	Coder            *ModelRoleConfig  `json:"coder"`
	PlanSummary      ModelRoleConfig   `json:"planSummary"`
	Builder          ModelRoleConfig   `json:"builder"`
	WholeFileBuilder *ModelRoleConfig  `json:"wholeFileBuilder"` // optional, defaults to builder model — access via GetWholeFileBuilder()
	Namer            ModelRoleConfig   `json:"namer"`
	CommitMsg        ModelRoleConfig   `json:"commitMsg"`
	ExecStatus       ModelRoleConfig   `json:"execStatus"`
	ContextLoader    *ModelRoleConfig  `json:"contextLoader"`
}

func (m *ModelPack) GetCoder() ModelRoleConfig {
	if m.Coder == nil {
		return m.Planner.ModelRoleConfig
	}
	return *m.Coder
}

func (m *ModelPack) GetWholeFileBuilder() ModelRoleConfig {
	if m.WholeFileBuilder == nil {
		return m.Builder
	}
	return *m.WholeFileBuilder
}

func (m *ModelPack) GetContextLoader() ModelRoleConfig {
	if m.ContextLoader == nil {
		return m.Planner.ModelRoleConfig
	}
	return *m.ContextLoader
}

type ModelOverrides struct {
	MaxConvoTokens       *int `json:"maxConvoTokens"`
	MaxTokens            *int `json:"maxContextTokens"`
	ReservedOutputTokens *int `json:"maxOutputTokens"`
}

type PlanSettings struct {
	ModelOverrides ModelOverrides `json:"modelOverrides"`
	ModelPack      *ModelPack     `json:"modelPack"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

func (p *PlanSettings) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch s := src.(type) {
	case []byte:
		return json.Unmarshal(s, p)
	case string:
		return json.Unmarshal([]byte(s), p)
	default:
		return fmt.Errorf("unsupported data type: %T", src)
	}
}

func (p PlanSettings) Value() (driver.Value, error) {
	return json.Marshal(p)
}

type CloudBillingFields struct {
	CreditsBalance        decimal.Decimal `json:"creditsBalance"`
	MonthlyGrant          decimal.Decimal `json:"monthlyGrant"`
	AutoRebuyEnabled      bool            `json:"autoRebuyEnabled"`
	AutoRebuyMinThreshold decimal.Decimal `json:"autoRebuyMinThreshold"`
	AutoRebuyToBalance    decimal.Decimal `json:"autoRebuyToBalance"`
	NotifyThreshold       decimal.Decimal `json:"notifyThreshold"`
	MaxThresholdPerMonth  decimal.Decimal `json:"maxThresholdPerMonth"`
	BillingCycleStartedAt time.Time       `json:"billingCycleStartedAt"`

	ChangedBillingMode bool `json:"changedBillingMode"`
	TrialPaid          bool `json:"trialPaid"`

	StripeSubscriptionId *string    `json:"stripeSubscriptionId"`
	SubscriptionStatus   *string    `json:"subscriptionStatus"`
	SubscriptionPausedAt *time.Time `json:"subscriptionPausedAt"`
	StripePaymentMethod  *string    `json:"stripePaymentMethod"`
}

type CreditsTransactionType string

const (
	CreditsTransactionTypeCredit CreditsTransactionType = "credit"
	CreditsTransactionTypeDebit  CreditsTransactionType = "debit"
)

type CreditType string

const (
	CreditTypeTrial      CreditType = "trial"
	CreditTypeGrant      CreditType = "grant"
	CreditTypeAdminGrant CreditType = "admin_grant"
	CreditTypePurchase   CreditType = "purchase"
	CreditTypeSwitch     CreditType = "switch"
)

type CreditsTransaction struct {
	Id              string                 `json:"id"`
	OrgId           string                 `json:"orgId"`
	OrgName         string                 `json:"orgName"`
	UserId          *string                `json:"userId"`
	UserEmail       *string                `json:"userEmail"`
	UserName        *string                `json:"userName"`
	TransactionType CreditsTransactionType `json:"transactionType"`
	Amount          decimal.Decimal        `json:"amount"`
	StartBalance    decimal.Decimal        `json:"startBalance"`
	EndBalance      decimal.Decimal        `json:"endBalance"`

	CreditType                  *CreditType      `json:"creditType,omitempty"`
	CreditIsAutoRebuy           bool             `json:"creditIsAutoRebuy"`
	CreditAutoRebuyMinThreshold *decimal.Decimal `json:"creditAutoRebuyMinThreshold,omitempty"`
	CreditAutoRebuyToBalance    *decimal.Decimal `json:"creditAutoRebuyToBalance,omitempty"`

	DebitInputTokens              *int             `json:"debitInputTokens,omitempty"`
	DebitOutputTokens             *int             `json:"debitOutputTokens,omitempty"`
	DebitModelInputPricePerToken  *decimal.Decimal `json:"debitModelInputPricePerToken,omitempty"`
	DebitModelOutputPricePerToken *decimal.Decimal `json:"debitModelOutputPricePerToken,omitempty"`

	DebitBaseAmount *decimal.Decimal `json:"debitBaseAmount,omitempty"`
	DebitSurcharge  *decimal.Decimal `json:"debitSurcharge,omitempty"`

	DebitModelProvider *ModelProvider `json:"debitModelProvider,omitempty"`
	DebitModelName     *string        `json:"debitModelName,omitempty"`
	DebitModelPackName *string        `json:"debitModelPackName,omitempty"`
	DebitModelRole     *ModelRole     `json:"debitModelRole,omitempty"`

	DebitPurpose  *string `json:"debitPurpose,omitempty"`
	DebitPlanId   *string `json:"debitPlanId,omitempty"`
	DebitPlanName *string `json:"debitPlanName,omitempty"`
	DebitId       *string `json:"debitId,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}
