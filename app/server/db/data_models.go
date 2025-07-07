package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	shared "plandex-shared"

	"github.com/sashabaranov/go-openai"
)

// The models below should only be used server-side.
// Many of them have corresponding models in shared/api for client-side use.
// This adds some duplication, but helps ensure that server-only data doesn't leak to the client.
// Models used client-side have a ToApi() method to convert it to the corresponding client-side model.

type AuthToken struct {
	Id        string     `db:"id"`
	UserId    string     `db:"user_id"`
	TokenHash string     `db:"token_hash"`
	CreatedAt time.Time  `db:"created_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

type Org struct {
	Id                 string  `db:"id"`
	Name               string  `db:"name"`
	Domain             *string `db:"domain"`
	AutoAddDomainUsers bool    `db:"auto_add_domain_users"`
	OwnerId            string  `db:"owner_id"`
	IsTrial            bool    `db:"is_trial"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (org *Org) ToApi() *shared.Org {
	return &shared.Org{
		Id:                 org.Id,
		Name:               org.Name,
		AutoAddDomainUsers: org.AutoAddDomainUsers,
		IsTrial:            org.IsTrial,
	}
}

type User struct {
	Id                string             `db:"id"`
	Name              string             `db:"name"`
	Email             string             `db:"email"`
	Domain            string             `db:"domain"`
	NumNonDraftPlans  int                `db:"num_non_draft_plans"`
	DefaultPlanConfig *shared.PlanConfig `db:"default_plan_config"`
	CreatedAt         time.Time          `db:"created_at"`
	UpdatedAt         time.Time          `db:"updated_at"`
}

func (user *User) ToApi() *shared.User {
	return &shared.User{
		Id:                user.Id,
		Name:              user.Name,
		Email:             user.Email,
		NumNonDraftPlans:  user.NumNonDraftPlans,
		IsTrial:           false, // legacy field
		DefaultPlanConfig: user.DefaultPlanConfig,
	}
}

type Invite struct {
	Id         string     `db:"id"`
	OrgId      string     `db:"org_id"`
	Email      string     `db:"email"`
	Name       string     `db:"name"`
	InviterId  string     `db:"inviter_id"`
	InviteeId  *string    `db:"invitee_id"`
	OrgRoleId  string     `db:"org_role_id"`
	AcceptedAt *time.Time `db:"accepted_at"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
}

func (invite *Invite) ToApi() *shared.Invite {
	return &shared.Invite{
		Id:         invite.Id,
		OrgId:      invite.OrgId,
		Email:      invite.Email,
		Name:       invite.Name,
		InviterId:  invite.InviterId,
		InviteeId:  invite.InviteeId,
		OrgRoleId:  invite.OrgRoleId,
		AcceptedAt: invite.AcceptedAt,
		CreatedAt:  invite.CreatedAt,
	}
}

type OrgUser struct {
	Id        string                `db:"id"`
	OrgId     string                `db:"org_id"`
	OrgRoleId string                `db:"org_role_id"`
	UserId    string                `db:"user_id"`
	Config    *shared.OrgUserConfig `db:"config"`
	CreatedAt time.Time             `db:"created_at"`
	UpdatedAt time.Time             `db:"updated_at"`
}

func (orgUser *OrgUser) ToApi() *shared.OrgUser {
	return &shared.OrgUser{
		OrgId:     orgUser.OrgId,
		OrgRoleId: orgUser.OrgRoleId,
		UserId:    orgUser.UserId,
		Config:    orgUser.Config,
	}
}

type Project struct {
	Id        string    `db:"id"`
	OrgId     string    `db:"org_id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (project *Project) ToApi() *shared.Project {
	return &shared.Project{
		Id:   project.Id,
		Name: project.Name,
	}
}

type Plan struct {
	Id              string             `db:"id"`
	OrgId           string             `db:"org_id"`
	OwnerId         string             `db:"owner_id"`
	ProjectId       string             `db:"project_id"`
	Name            string             `db:"name"`
	SharedWithOrgAt *time.Time         `db:"shared_with_org_at,omitempty"`
	TotalReplies    int                `db:"total_replies"`
	ActiveBranches  int                `db:"active_branches"`
	PlanConfig      *shared.PlanConfig `db:"plan_config"`
	ArchivedAt      *time.Time         `db:"archived_at,omitempty"`
	CreatedAt       time.Time          `db:"created_at"`
	UpdatedAt       time.Time          `db:"updated_at"`
}

func (plan *Plan) ToApi() *shared.Plan {
	return &shared.Plan{
		Id:              plan.Id,
		OwnerId:         plan.OwnerId,
		ProjectId:       plan.ProjectId,
		Name:            plan.Name,
		SharedWithOrgAt: plan.SharedWithOrgAt,
		TotalReplies:    plan.TotalReplies,
		ActiveBranches:  plan.ActiveBranches,
		PlanConfig:      plan.PlanConfig,
		ArchivedAt:      plan.ArchivedAt,
		CreatedAt:       plan.CreatedAt,
		UpdatedAt:       plan.UpdatedAt,
	}
}

type Branch struct {
	Id              string            `db:"id"`
	OrgId           string            `db:"org_id"`
	OwnerId         string            `db:"owner_id"`
	PlanId          string            `db:"plan_id"`
	ParentBranchId  *string           `db:"parent_branch_id"`
	Name            string            `db:"name"`
	Status          shared.PlanStatus `db:"status"`
	Error           *string           `db:"error"`
	ContextTokens   int               `db:"context_tokens"`
	ConvoTokens     int               `db:"convo_tokens"`
	SharedWithOrgAt *time.Time        `db:"shared_with_org_at,omitempty"`
	ArchivedAt      *time.Time        `db:"archived_at,omitempty"`
	CreatedAt       time.Time         `db:"created_at"`
	UpdatedAt       time.Time         `db:"updated_at"`
	DeletedAt       *time.Time        `db:"deleted_at"`
}

func (branch *Branch) ToApi() *shared.Branch {
	return &shared.Branch{
		Id:              branch.Id,
		PlanId:          branch.PlanId,
		OwnerId:         branch.OwnerId,
		ParentBranchId:  branch.ParentBranchId,
		Name:            branch.Name,
		Status:          branch.Status,
		ContextTokens:   branch.ContextTokens,
		ConvoTokens:     branch.ConvoTokens,
		SharedWithOrgAt: branch.SharedWithOrgAt,
		ArchivedAt:      branch.ArchivedAt,
		CreatedAt:       branch.CreatedAt,
		UpdatedAt:       branch.UpdatedAt,
	}
}

type ConvoSummary struct {
	Id                          string    `db:"id"`
	OrgId                       string    `db:"org_id"`
	PlanId                      string    `db:"plan_id"`
	LatestConvoMessageId        string    `db:"latest_convo_message_id"`
	LatestConvoMessageCreatedAt time.Time `db:"latest_convo_message_created_at"`
	Summary                     string    `db:"summary"`
	Tokens                      int       `db:"tokens"`
	NumMessages                 int       `db:"num_messages"`
	CreatedAt                   time.Time `db:"created_at"`
}

func (summary *ConvoSummary) ToApi() *shared.ConvoSummary {
	return &shared.ConvoSummary{
		Id:                          summary.Id,
		LatestConvoMessageId:        summary.LatestConvoMessageId,
		LatestConvoMessageCreatedAt: summary.LatestConvoMessageCreatedAt,
		Summary:                     summary.Summary,
		Tokens:                      summary.Tokens,
		NumMessages:                 summary.NumMessages,
		CreatedAt:                   summary.CreatedAt,
	}
}

type PlanBuild struct {
	Id             string    `db:"id"`
	OrgId          string    `db:"org_id"`
	PlanId         string    `db:"plan_id"`
	ConvoMessageId string    `db:"convo_message_id"`
	FilePath       string    `db:"file_path"`
	Error          string    `db:"error"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

func (build *PlanBuild) ToApi() *shared.PlanBuild {
	return &shared.PlanBuild{
		Id:             build.Id,
		ConvoMessageId: build.ConvoMessageId,
		Error:          build.Error,
		FilePath:       build.FilePath,
		CreatedAt:      build.CreatedAt,
		UpdatedAt:      build.UpdatedAt,
	}
}

type OrgRole struct {
	Id          string    `db:"id"`
	OrgId       *string   `db:"org_id"`
	Name        string    `db:"name"`
	Label       string    `db:"label"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func (role *OrgRole) ToApi() *shared.OrgRole {
	return &shared.OrgRole{
		Id:          role.Id,
		IsDefault:   role.OrgId == nil,
		Label:       role.Label,
		Description: role.Description,
	}
}

type ModelStream struct {
	Id              string     `db:"id"`
	OrgId           string     `db:"org_id"`
	PlanId          string     `db:"plan_id"`
	InternalIp      string     `db:"internal_ip"`
	Branch          string     `db:"branch"`
	LastHeartbeatAt time.Time  `db:"last_heartbeat_at"`
	CreatedAt       time.Time  `db:"created_at"`
	FinishedAt      *time.Time `db:"finished_at"`
}

// type ModelStreamSubscription struct {
// 	Id            string     `db:"id"`
// 	OrgId         string     `db:"org_id"`
// 	PlanId        string     `db:"plan_id"`
// 	UserId        string     `db:"user_id"`
// 	ModelStreamId string     `db:"model_stream_id"`
// 	UserIp        string     `db:"user_ip"`
// 	CreatedAt     time.Time  `db:"created_at"`
// 	FinishedAt    *time.Time `db:"finished_at"`
// }

type LockScope string

const (
	LockScopeRead  LockScope = "r"
	LockScopeWrite LockScope = "w"
)

type repoLock struct {
	Id              string    `db:"id"`
	OrgId           string    `db:"org_id"`
	UserId          *string   `db:"user_id"`
	PlanId          string    `db:"plan_id"`
	Scope           LockScope `db:"scope"`
	Branch          *string   `db:"branch"`
	PlanBuildId     *string   `db:"plan_build_id"`
	LastHeartbeatAt time.Time `db:"last_heartbeat_at"`
	CreatedAt       time.Time `db:"created_at"`
}

type ModelPack struct {
	Id               string                   `db:"id"`
	OrgId            string                   `db:"org_id"`
	Name             string                   `db:"name"`
	Description      string                   `db:"description"`
	Planner          shared.PlannerRoleConfig `db:"planner"`
	Coder            *shared.ModelRoleConfig  `db:"coder"`
	PlanSummary      shared.ModelRoleConfig   `db:"plan_summary"`
	Builder          shared.ModelRoleConfig   `db:"builder"`
	WholeFileBuilder *shared.ModelRoleConfig  `db:"whole_file_builder"`
	Namer            shared.ModelRoleConfig   `db:"namer"`
	CommitMsg        shared.ModelRoleConfig   `db:"commit_msg"`
	ExecStatus       shared.ModelRoleConfig   `db:"exec_status"`
	Architect        *shared.ModelRoleConfig  `db:"context_loader"`
	CreatedAt        time.Time                `db:"created_at"`
	UpdatedAt        time.Time                `db:"updated_at"`
}

func ModelPackFromApi(apiModelPack *shared.ModelPack) *ModelPack {
	return &ModelPack{
		Name:             apiModelPack.Name,
		Description:      apiModelPack.Description,
		Planner:          apiModelPack.Planner,
		Architect:        apiModelPack.Architect,
		Coder:            apiModelPack.Coder,
		PlanSummary:      apiModelPack.PlanSummary,
		Builder:          apiModelPack.Builder,
		WholeFileBuilder: apiModelPack.WholeFileBuilder,
		Namer:            apiModelPack.Namer,
		CommitMsg:        apiModelPack.CommitMsg,
		ExecStatus:       apiModelPack.ExecStatus,
	}
}

func (modelPack *ModelPack) ToApi() *shared.ModelPack {
	return &shared.ModelPack{
		Id:               modelPack.Id,
		Name:             modelPack.Name,
		Description:      modelPack.Description,
		Planner:          modelPack.Planner,
		Architect:        modelPack.Architect,
		Coder:            modelPack.Coder,
		PlanSummary:      modelPack.PlanSummary,
		Builder:          modelPack.Builder,
		WholeFileBuilder: modelPack.WholeFileBuilder,
		Namer:            modelPack.Namer,
		CommitMsg:        modelPack.CommitMsg,
		ExecStatus:       modelPack.ExecStatus,
	}
}

type CustomModel struct {
	Id                    string                   `db:"id"`
	OrgId                 string                   `db:"org_id"`
	ModelId               shared.ModelId           `db:"model_id"`
	Publisher             shared.ModelPublisher    `db:"publisher"`
	Description           string                   `db:"description"`
	MaxTokens             int                      `db:"max_tokens"`
	DefaultMaxConvoTokens int                      `db:"default_max_convo_tokens"`
	MaxOutputTokens       int                      `db:"max_output_tokens"`
	ReservedOutputTokens  int                      `db:"reserved_output_tokens"`
	HasImageSupport       bool                     `db:"has_image_support"`
	PreferredOutputFormat shared.ModelOutputFormat `db:"preferred_output_format"`

	SystemPromptDisabled   bool                   `db:"system_prompt_disabled"`
	RoleParamsDisabled     bool                   `db:"role_params_disabled"`
	StopDisabled           bool                   `db:"stop_disabled"`
	PredictedOutputEnabled bool                   `db:"predicted_output_enabled"`
	ReasoningEffortEnabled bool                   `db:"reasoning_effort_enabled"`
	ReasoningEffort        shared.ReasoningEffort `db:"reasoning_effort"`
	IncludeReasoning       bool                   `db:"include_reasoning"`
	ReasoningBudget        int                    `db:"reasoning_budget"`
	SupportsCacheControl   bool                   `db:"supports_cache_control"`
	// for anthropic, single message system prompt needs to be flipped to 'user'
	SingleMessageNoSystemPrompt bool `db:"single_message_no_system_prompt"`

	// for anthropic, token estimate padding percentage
	TokenEstimatePaddingPct float64 `db:"token_estimate_padding_pct"`

	Providers CustomModelProviders `db:"providers"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func CustomModelFromApi(apiModel *shared.CustomModel) *CustomModel {
	providers := make(CustomModelProviders, len(apiModel.Providers))
	for i, provider := range apiModel.Providers {
		providers[i] = CustomModelUsesProvider{
			Provider:       provider.Provider,
			CustomProvider: provider.CustomProvider,
			ModelName:      provider.ModelName,
		}
	}
	dbModel := CustomModel{
		Id:                          apiModel.Id,
		ModelId:                     apiModel.ModelId,
		Publisher:                   apiModel.Publisher,
		Description:                 apiModel.Description,
		MaxTokens:                   apiModel.MaxTokens,
		HasImageSupport:             apiModel.ModelCompatibility.HasImageSupport,
		DefaultMaxConvoTokens:       apiModel.DefaultMaxConvoTokens,
		MaxOutputTokens:             apiModel.MaxOutputTokens,
		ReservedOutputTokens:        apiModel.ReservedOutputTokens,
		PreferredOutputFormat:       apiModel.PreferredOutputFormat,
		SystemPromptDisabled:        apiModel.SystemPromptDisabled,
		RoleParamsDisabled:          apiModel.RoleParamsDisabled,
		StopDisabled:                apiModel.StopDisabled,
		PredictedOutputEnabled:      apiModel.PredictedOutputEnabled,
		IncludeReasoning:            apiModel.IncludeReasoning,
		ReasoningEffortEnabled:      apiModel.ReasoningEffortEnabled,
		ReasoningEffort:             apiModel.ReasoningEffort,
		ReasoningBudget:             apiModel.ReasoningBudget,
		SupportsCacheControl:        apiModel.SupportsCacheControl,
		SingleMessageNoSystemPrompt: apiModel.SingleMessageNoSystemPrompt,
		TokenEstimatePaddingPct:     apiModel.TokenEstimatePaddingPct,
		Providers:                   providers,
	}

	return &dbModel
}

func (model *CustomModel) ToApi() *shared.CustomModel {
	providers := make([]shared.BaseModelUsesProvider, len(model.Providers))
	for i, provider := range model.Providers {
		providers[i] = *provider.ToApi()
	}
	return &shared.CustomModel{
		Id:          model.Id,
		ModelId:     model.ModelId,
		Publisher:   model.Publisher,
		Description: model.Description,
		BaseModelShared: shared.BaseModelShared{
			DefaultMaxConvoTokens:       model.DefaultMaxConvoTokens,
			MaxTokens:                   model.MaxTokens,
			MaxOutputTokens:             model.MaxOutputTokens,
			ReservedOutputTokens:        model.ReservedOutputTokens,
			PreferredOutputFormat:       model.PreferredOutputFormat,
			SystemPromptDisabled:        model.SystemPromptDisabled,
			RoleParamsDisabled:          model.RoleParamsDisabled,
			StopDisabled:                model.StopDisabled,
			PredictedOutputEnabled:      model.PredictedOutputEnabled,
			IncludeReasoning:            model.IncludeReasoning,
			ReasoningEffortEnabled:      model.ReasoningEffortEnabled,
			ReasoningEffort:             model.ReasoningEffort,
			ReasoningBudget:             model.ReasoningBudget,
			SupportsCacheControl:        model.SupportsCacheControl,
			SingleMessageNoSystemPrompt: model.SingleMessageNoSystemPrompt,
			TokenEstimatePaddingPct:     model.TokenEstimatePaddingPct,

			ModelCompatibility: shared.ModelCompatibility{
				HasImageSupport: model.HasImageSupport,
			},
		},
		Providers: providers,
		CreatedAt: &model.CreatedAt,
		UpdatedAt: &model.UpdatedAt,
	}
}

type ExtraAuthVars []shared.ModelProviderExtraAuthVars

func (e *ExtraAuthVars) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	switch s := src.(type) {
	case []byte:
		return json.Unmarshal(s, e)
	case string:
		return json.Unmarshal([]byte(s), e)
	default:
		return fmt.Errorf("unsupported data type: %T", src)
	}
}

func (e ExtraAuthVars) Value() (driver.Value, error) {
	return json.Marshal(e)
}

type CustomProvider struct {
	Id            string        `db:"id"`
	OrgId         string        `db:"org_id"`
	Name          string        `db:"name"`
	BaseUrl       string        `db:"base_url"`
	SkipAuth      bool          `db:"skip_auth"`
	ApiKeyEnvVar  string        `db:"api_key_env_var"`
	ExtraAuthVars ExtraAuthVars `db:"extra_auth_vars"`
	CreatedAt     time.Time     `db:"created_at"`
	UpdatedAt     time.Time     `db:"updated_at"`
}

func CustomProviderFromApi(apiProvider *shared.CustomProvider) *CustomProvider {
	return &CustomProvider{
		Id:            apiProvider.Id,
		Name:          apiProvider.Name,
		BaseUrl:       apiProvider.BaseUrl,
		SkipAuth:      apiProvider.SkipAuth,
		ApiKeyEnvVar:  apiProvider.ApiKeyEnvVar,
		ExtraAuthVars: apiProvider.ExtraAuthVars,
	}
}

func (provider *CustomProvider) ToApi() *shared.CustomProvider {
	return &shared.CustomProvider{
		Id:            provider.Id,
		Name:          provider.Name,
		BaseUrl:       provider.BaseUrl,
		SkipAuth:      provider.SkipAuth,
		ApiKeyEnvVar:  provider.ApiKeyEnvVar,
		ExtraAuthVars: provider.ExtraAuthVars,
	}
}

type CustomModelUsesProvider struct {
	Provider       shared.ModelProvider `db:"provider"`
	CustomProvider *string              `db:"custom_provider"`
	ModelName      shared.ModelName     `db:"model_name"`
}

func (usesProvider *CustomModelUsesProvider) ToApi() *shared.BaseModelUsesProvider {
	return &shared.BaseModelUsesProvider{
		Provider:       usesProvider.Provider,
		ModelName:      usesProvider.ModelName,
		CustomProvider: usesProvider.CustomProvider,
	}
}

type CustomModelProviders []CustomModelUsesProvider

func (providers *CustomModelProviders) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	switch s := src.(type) {
	case []byte:
		return json.Unmarshal(s, providers)
	case string:
		return json.Unmarshal([]byte(s), providers)
	}

	return fmt.Errorf("unsupported data type: %T", src)
}

func (providers CustomModelProviders) Value() (driver.Value, error) {
	return json.Marshal(providers)
}

type DefaultPlanSettings struct {
	Id           string              `db:"id"`
	OrgId        string              `db:"org_id"`
	PlanSettings shared.PlanSettings `db:"plan_settings"`
	CreatedAt    time.Time           `db:"created_at"`
	UpdatedAt    time.Time           `db:"updated_at"`
}

// Models below are stored in files, not in the database.
// This allows us to store them in a git repo and use git to manage history.

type Context struct {
	Id              string                `json:"id"`
	OrgId           string                `json:"orgId"`
	OwnerId         string                `json:"ownerId"`
	ProjectId       string                `json:"projectId"`
	PlanId          string                `json:"planId"`
	ContextType     shared.ContextType    `json:"contextType"`
	Name            string                `json:"name"`
	Url             string                `json:"url"`
	FilePath        string                `json:"filePath"`
	Sha             string                `json:"sha"`
	NumTokens       int                   `json:"numTokens"`
	Body            string                `json:"body,omitempty"`
	BodySize        int64                 `json:"bodySize,omitempty"`
	ForceSkipIgnore bool                  `json:"forceSkipIgnore"`
	ImageDetail     openai.ImageURLDetail `json:"imageDetail,omitempty"`
	MapParts        shared.FileMapBodies  `json:"mapParts,omitempty"`
	MapShas         map[string]string     `json:"mapShas,omitempty"`
	MapTokens       map[string]int        `json:"mapTokens,omitempty"`
	MapSizes        map[string]int64      `json:"mapSizes,omitempty"`
	AutoLoaded      bool                  `json:"autoLoaded"`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
}

func (context *Context) ToMeta() *Context {
	// everything except body and mapParts
	return &Context{
		Id:              context.Id,
		OrgId:           context.OrgId,
		OwnerId:         context.OwnerId,
		ProjectId:       context.ProjectId,
		PlanId:          context.PlanId,
		ContextType:     context.ContextType,
		Name:            context.Name,
		Url:             context.Url,
		FilePath:        context.FilePath,
		Sha:             context.Sha,
		NumTokens:       context.NumTokens,
		BodySize:        context.BodySize,
		ForceSkipIgnore: context.ForceSkipIgnore,
		AutoLoaded:      context.AutoLoaded,
		ImageDetail:     context.ImageDetail,
		MapShas:         context.MapShas,
		MapTokens:       context.MapTokens,
		MapSizes:        context.MapSizes,
		CreatedAt:       context.CreatedAt,
		UpdatedAt:       context.UpdatedAt,
	}
}

func (context *Context) ToApi() *shared.Context {
	return &shared.Context{
		Id:              context.Id,
		OwnerId:         context.OwnerId,
		ContextType:     context.ContextType,
		Name:            context.Name,
		Url:             context.Url,
		FilePath:        context.FilePath,
		Sha:             context.Sha,
		NumTokens:       context.NumTokens,
		Body:            context.Body,
		BodySize:        context.BodySize,
		ForceSkipIgnore: context.ForceSkipIgnore,
		AutoLoaded:      context.AutoLoaded,
		ImageDetail:     context.ImageDetail,
		MapParts:        context.MapParts,
		MapShas:         context.MapShas,
		MapTokens:       context.MapTokens,
		MapSizes:        context.MapSizes,
		CreatedAt:       context.CreatedAt,
		UpdatedAt:       context.UpdatedAt,
	}
}

type ConvoMessage struct {
	Id                    string                   `json:"id"`
	OrgId                 string                   `json:"orgId"`
	PlanId                string                   `json:"planId"`
	UserId                string                   `json:"userId"`
	Role                  string                   `json:"role"`
	Tokens                int                      `json:"tokens"`
	Num                   int                      `json:"num"`
	Message               string                   `json:"message"`
	Stopped               bool                     `json:"stopped"`
	Subtask               *Subtask                 `json:"subtask,omitempty"`
	AddedSubtasks         []*Subtask               `json:"addedSubtasks,omitempty"`
	RemovedSubtasks       []string                 `json:"removedSubtasks,omitempty"`
	Flags                 shared.ConvoMessageFlags `json:"flags"`
	ActivatedPaths        map[string]bool          `json:"activatePaths,omitempty"`
	ActivatedPathsOrdered []string                 `json:"activatePathsOrdered,omitempty"`
	CreatedAt             time.Time                `json:"createdAt"`
}

func (msg *ConvoMessage) ToApi() *shared.ConvoMessage {
	addedSubtasks := make([]*shared.Subtask, len(msg.AddedSubtasks))
	for i, subtask := range msg.AddedSubtasks {
		addedSubtasks[i] = subtask.ToApi()
	}
	return &shared.ConvoMessage{
		Id:              msg.Id,
		UserId:          msg.UserId,
		Role:            msg.Role,
		Tokens:          msg.Tokens,
		Num:             msg.Num,
		Message:         msg.Message,
		Stopped:         msg.Stopped,
		Flags:           msg.Flags,
		Subtask:         msg.Subtask.ToApi(),
		AddedSubtasks:   addedSubtasks,
		RemovedSubtasks: msg.RemovedSubtasks,
		CreatedAt:       msg.CreatedAt,
	}
}

type ConvoMessageDescription struct {
	Id                    string `json:"id"`
	OrgId                 string `json:"orgId"`
	PlanId                string `json:"planId"`
	ConvoMessageId        string `json:"convoMessageId"`
	SummarizedToMessageId string `json:"summarizedToMessageId"`
	WroteFiles            bool   `json:"wroteFiles"`
	CommitMsg             string `json:"commitMsg"`
	// Files                 []string        `json:"files"`
	Operations            []*shared.Operation `json:"operations"`
	Error                 string              `json:"error"`
	DidBuild              bool                `json:"didBuild"`
	BuildPathsInvalidated map[string]bool     `json:"buildPathsInvalidated"`
	AppliedAt             *time.Time          `json:"appliedAt,omitempty"`
	CreatedAt             time.Time           `json:"createdAt"`
	UpdatedAt             time.Time           `json:"updatedAt"`
}

func (desc *ConvoMessageDescription) ToApi() *shared.ConvoMessageDescription {
	return &shared.ConvoMessageDescription{
		Id:                    desc.Id,
		ConvoMessageId:        desc.ConvoMessageId,
		SummarizedToMessageId: desc.SummarizedToMessageId,
		WroteFiles:            desc.WroteFiles,
		CommitMsg:             desc.CommitMsg,
		// Files:                 desc.Files,
		Operations:            desc.Operations,
		DidBuild:              desc.DidBuild,
		BuildPathsInvalidated: desc.BuildPathsInvalidated,
		AppliedAt:             desc.AppliedAt,
		Error:                 desc.Error,
		CreatedAt:             desc.CreatedAt,
		UpdatedAt:             desc.UpdatedAt,
	}
}

type PlanFileResult struct {
	Id                  string `json:"id"`
	TypeVersion         int    `json:"typeVersion"`
	ReplaceWithLineNums bool   `json:"replaceWithLineNums"`
	OrgId               string `json:"orgId"`
	PlanId              string `json:"planId"`
	ConvoMessageId      string `json:"convoMessageId"`
	PlanBuildId         string `json:"planBuildId"`
	Path                string `json:"path"`
	Content             string `json:"content,omitempty"`

	Replacements []*shared.Replacement `json:"replacements"`

	RemovedFile bool `json:"removedFile"`

	AnyFailed bool   `json:"anyFailed"`
	Error     string `json:"error"`

	SyntaxErrors []string `json:"syntaxErrors"`

	AppliedAt  *time.Time `json:"appliedAt,omitempty"`
	RejectedAt *time.Time `json:"rejectedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

func (res *PlanFileResult) ToApi() *shared.PlanFileResult {
	return &shared.PlanFileResult{
		Id:                  res.Id,
		TypeVersion:         res.TypeVersion,
		ReplaceWithLineNums: res.ReplaceWithLineNums,
		PlanBuildId:         res.PlanBuildId,
		ConvoMessageId:      res.ConvoMessageId,
		Path:                res.Path,
		Content:             res.Content,
		AnyFailed:           res.AnyFailed,
		AppliedAt:           res.AppliedAt,
		RejectedAt:          res.RejectedAt,
		Replacements:        res.Replacements,
		RemovedFile:         res.RemovedFile,
		CreatedAt:           res.CreatedAt,
		UpdatedAt:           res.UpdatedAt,
	}
}

type PlanApply struct {
	Id                         string    `json:"id"`
	OrgId                      string    `json:"orgId"`
	PlanId                     string    `json:"planId"`
	UserId                     string    `json:"userId"`
	ConvoMessageIds            []string  `json:"convoMessageIds"`
	ConvoMessageDescriptionIds []string  `json:"convoMessageDescriptionIds"`
	PlanFileResultIds          []string  `json:"planFileResultIds"`
	CommitMsg                  string    `json:"commitMsg"`
	CreatedAt                  time.Time `json:"createdAt"`
}

func (apply *PlanApply) ToApi() *shared.PlanApply {
	return &shared.PlanApply{
		Id:                         apply.Id,
		UserId:                     apply.UserId,
		ConvoMessageIds:            apply.ConvoMessageIds,
		ConvoMessageDescriptionIds: apply.ConvoMessageDescriptionIds,
		PlanFileResultIds:          apply.PlanFileResultIds,
		CommitMsg:                  apply.CommitMsg,
		CreatedAt:                  apply.CreatedAt,
	}
}

type Subtask struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	UsesFiles   []string `json:"usesFiles"`
	IsFinished  bool     `json:"isFinished"`
	NumTries    int      `json:"numTries"`
}

func (subtask *Subtask) ToApi() *shared.Subtask {
	if subtask == nil {
		return nil
	}
	return &shared.Subtask{
		Title:       subtask.Title,
		Description: subtask.Description,
		UsesFiles:   subtask.UsesFiles,
		IsFinished:  subtask.IsFinished,
	}
}
