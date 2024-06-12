package db

import (
	"time"

	"github.com/plandex/plandex/shared"
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
	IsTrial   bool       `db:"is_trial"`
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
		Id:   org.Id,
		Name: org.Name,
	}
}

type User struct {
	Id               string    `db:"id"`
	Name             string    `db:"name"`
	Email            string    `db:"email"`
	Domain           string    `db:"domain"`
	NumNonDraftPlans int       `db:"num_non_draft_plans"`
	IsTrial          bool      `db:"is_trial"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

func (user *User) ToApi() *shared.User {
	return &shared.User{
		Id:               user.Id,
		Name:             user.Name,
		Email:            user.Email,
		NumNonDraftPlans: user.NumNonDraftPlans,
		IsTrial:          user.IsTrial,
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
	Id        string    `db:"id"`
	OrgId     string    `db:"org_id"`
	OrgRoleId string    `db:"org_role_id"`
	UserId    string    `db:"user_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (orgUser *OrgUser) ToApi() *shared.OrgUser {
	return &shared.OrgUser{
		OrgId:     orgUser.OrgId,
		OrgRoleId: orgUser.OrgRoleId,
		UserId:    orgUser.UserId,
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
	Id              string     `db:"id"`
	OrgId           string     `db:"org_id"`
	OwnerId         string     `db:"owner_id"`
	ProjectId       string     `db:"project_id"`
	Name            string     `db:"name"`
	SharedWithOrgAt *time.Time `db:"shared_with_org_at,omitempty"`
	TotalReplies    int        `db:"total_replies"`
	ActiveBranches  int        `db:"active_branches"`
	ArchivedAt      *time.Time `db:"archived_at,omitempty"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
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
	UserId          string    `db:"user_id"`
	PlanId          string    `db:"plan_id"`
	Scope           LockScope `db:"scope"`
	Branch          *string   `db:"branch"`
	PlanBuildId     *string   `db:"plan_build_id"`
	LastHeartbeatAt time.Time `db:"last_heartbeat_at"`
	CreatedAt       time.Time `db:"created_at"`
}

type ModelPack struct {
	Id          string                   `db:"id"`
	OrgId       string                   `db:"org_id"`
	Name        string                   `db:"name"`
	Description string                   `db:"description"`
	Planner     shared.PlannerRoleConfig `db:"planner"`
	PlanSummary shared.ModelRoleConfig   `db:"plan_summary"`
	Builder     shared.ModelRoleConfig   `db:"builder"`
	Namer       shared.ModelRoleConfig   `db:"namer"`
	CommitMsg   shared.ModelRoleConfig   `db:"commit_msg"`
	ExecStatus  shared.ModelRoleConfig   `db:"exec_status"`
	CreatedAt   time.Time                `db:"created_at"`
}

func (modelPack *ModelPack) ToApi() *shared.ModelPack {
	return &shared.ModelPack{
		Id:          modelPack.Id,
		Name:        modelPack.Name,
		Description: modelPack.Description,
		Planner:     modelPack.Planner,
		PlanSummary: modelPack.PlanSummary,
		Builder:     modelPack.Builder,
		Namer:       modelPack.Namer,
		CommitMsg:   modelPack.CommitMsg,
		ExecStatus:  modelPack.ExecStatus,
	}
}

type AvailableModel struct {
	Id                          string               `db:"id"`
	OrgId                       string               `db:"org_id"`
	Provider                    shared.ModelProvider `db:"provider"`
	CustomProvider              *string              `db:"custom_provider"`
	BaseUrl                     string               `db:"base_url"`
	ModelName                   string               `db:"model_name"`
	Description                 string               `db:"description"`
	MaxTokens                   int                  `db:"max_tokens"`
	ApiKeyEnvVar                string               `db:"api_key_env_var"`
	IsOpenAICompatible          bool                 `db:"is_openai_compatible"`
	HasJsonResponseMode         bool                 `db:"has_json_mode"`
	HasStreaming                bool                 `db:"has_streaming"`
	HasFunctionCalling          bool                 `db:"has_function_calling"`
	HasStreamingFunctionCalls   bool                 `db:"has_streaming_function_calls"`
	DefaultMaxConvoTokens       int                  `db:"default_max_convo_tokens"`
	DefaultReservedOutputTokens int                  `db:"default_reserved_output_tokens"`
	CreatedAt                   time.Time            `db:"created_at"`
	UpdatedAt                   time.Time            `db:"updated_at"`
}

func (model *AvailableModel) ToApi() *shared.AvailableModel {
	return &shared.AvailableModel{
		Id: model.Id,
		BaseModelConfig: shared.BaseModelConfig{
			Provider:       model.Provider,
			CustomProvider: model.CustomProvider,
			BaseUrl:        model.BaseUrl,
			ModelName:      model.ModelName,
			MaxTokens:      model.MaxTokens,
			ApiKeyEnvVar:   model.ApiKeyEnvVar,
			ModelCompatibility: shared.ModelCompatibility{
				IsOpenAICompatible:        model.IsOpenAICompatible,
				HasJsonResponseMode:       model.HasJsonResponseMode,
				HasStreaming:              model.HasStreaming,
				HasFunctionCalling:        model.HasFunctionCalling,
				HasStreamingFunctionCalls: model.HasStreamingFunctionCalls,
			}},
		Description:                 model.Description,
		DefaultMaxConvoTokens:       model.DefaultMaxConvoTokens,
		DefaultReservedOutputTokens: model.DefaultReservedOutputTokens,
		CreatedAt:                   model.CreatedAt,
		UpdatedAt:                   model.UpdatedAt,
	}
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
	PlanId          string                `json:"planId"`
	ContextType     shared.ContextType    `json:"contextType"`
	Name            string                `json:"name"`
	Url             string                `json:"url"`
	FilePath        string                `json:"filePath"`
	Sha             string                `json:"sha"`
	NumTokens       int                   `json:"numTokens"`
	Body            string                `json:"body,omitempty"`
	ForceSkipIgnore bool                  `json:"forceSkipIgnore"`
	ImageDetail     openai.ImageURLDetail `json:"imageDetail,omitempty"`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
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
		ForceSkipIgnore: context.ForceSkipIgnore,
		CreatedAt:       context.CreatedAt,
		UpdatedAt:       context.UpdatedAt,
	}
}

type ConvoMessage struct {
	Id        string    `json:"id"`
	OrgId     string    `json:"orgId"`
	PlanId    string    `json:"planId"`
	UserId    string    `json:"userId"`
	Role      string    `json:"role"`
	Tokens    int       `json:"tokens"`
	Num       int       `json:"num"`
	Message   string    `json:"message"`
	Stopped   bool      `json:"stopped"`
	CreatedAt time.Time `json:"createdAt"`
}

func (msg *ConvoMessage) ToApi() *shared.ConvoMessage {
	return &shared.ConvoMessage{
		Id:        msg.Id,
		UserId:    msg.UserId,
		Role:      msg.Role,
		Tokens:    msg.Tokens,
		Num:       msg.Num,
		Message:   msg.Message,
		Stopped:   msg.Stopped,
		CreatedAt: msg.CreatedAt,
	}
}

type ConvoMessageDescription struct {
	Id                    string          `json:"id"`
	OrgId                 string          `json:"orgId"`
	PlanId                string          `json:"planId"`
	ConvoMessageId        string          `json:"convoMessageId"`
	SummarizedToMessageId string          `json:"summarizedToMessageId"`
	MadePlan              bool            `json:"madePlan"`
	CommitMsg             string          `json:"commitMsg"`
	Files                 []string        `json:"files"`
	Error                 string          `json:"error"`
	DidBuild              bool            `json:"didBuild"`
	BuildPathsInvalidated map[string]bool `json:"buildPathsInvalidated"`
	AppliedAt             *time.Time      `json:"appliedAt,omitempty"`
	CreatedAt             time.Time       `json:"createdAt"`
	UpdatedAt             time.Time       `json:"updatedAt"`
}

func (desc *ConvoMessageDescription) ToApi() *shared.ConvoMessageDescription {
	return &shared.ConvoMessageDescription{
		Id:                    desc.Id,
		ConvoMessageId:        desc.ConvoMessageId,
		SummarizedToMessageId: desc.SummarizedToMessageId,
		MadePlan:              desc.MadePlan,
		CommitMsg:             desc.CommitMsg,
		Files:                 desc.Files,
		DidBuild:              desc.DidBuild,
		BuildPathsInvalidated: desc.BuildPathsInvalidated,
		AppliedAt:             desc.AppliedAt,
		Error:                 desc.Error,
		CreatedAt:             desc.CreatedAt,
		UpdatedAt:             desc.UpdatedAt,
	}
}

type PlanFileResult struct {
	Id                  string                `json:"id"`
	TypeVersion         int                   `json:"typeVersion"`
	ReplaceWithLineNums bool                  `json:"replaceWithLineNums"`
	OrgId               string                `json:"orgId"`
	PlanId              string                `json:"planId"`
	ConvoMessageId      string                `json:"convoMessageId"`
	PlanBuildId         string                `json:"planBuildId"`
	Path                string                `json:"path"`
	Content             string                `json:"content,omitempty"`
	Replacements        []*shared.Replacement `json:"replacements"`
	AnyFailed           bool                  `json:"anyFailed"`
	Error               string                `json:"error"`

	CanVerify    bool       `json:"canVerify"`
	RanVerifyAt  *time.Time `json:"ranVerifyAt,omitempty"`
	VerifyPassed bool       `json:"verifyPassed"`

	WillCheckSyntax bool     `json:"willCheckSyntax"`
	SyntaxValid     bool     `json:"syntaxValid"`
	SyntaxErrors    []string `json:"syntaxErrors"`

	IsFix       bool `json:"isFix"`
	IsSyntaxFix bool `json:"isSyntaxFix"`
	IsOtherFix  bool `json:"isOtherFix"`
	FixEpoch    int  `json:"fixEpoch"`

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
		CanVerify:           res.CanVerify,
		RanVerifyAt:         res.RanVerifyAt,
		VerifyPassed:        res.VerifyPassed,
		IsFix:               res.IsFix,
		IsSyntaxFix:         res.IsSyntaxFix,
		IsOtherFix:          res.IsOtherFix,
		CreatedAt:           res.CreatedAt,
		UpdatedAt:           res.UpdatedAt,
	}
}
