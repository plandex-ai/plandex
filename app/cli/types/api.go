package types

import (
	"github.com/plandex/plandex/shared"
)

type OnStreamPlanParams struct {
	Msg *shared.StreamMessage
	Err error
}

type OnStreamPlan func(params OnStreamPlanParams)

type ApiClient interface {
	StartTrial() (*shared.StartTrialResponse, *shared.ApiError)
	ConvertTrial(req shared.ConvertTrialRequest) (*shared.SessionResponse, *shared.ApiError)

	CreateEmailVerification(email, customHost, userId string) (*shared.CreateEmailVerificationResponse, *shared.ApiError)

	CreateAccount(req shared.CreateAccountRequest, customHost string) (*shared.SessionResponse, *shared.ApiError)
	SignIn(req shared.SignInRequest, customHost string) (*shared.SessionResponse, *shared.ApiError)
	SignOut() *shared.ApiError

	GetOrgSession() *shared.ApiError
	ListOrgs() ([]*shared.Org, *shared.ApiError)
	CreateOrg(req shared.CreateOrgRequest) (*shared.CreateOrgResponse, *shared.ApiError)

	ListUsers() (*shared.ListUsersResponse, *shared.ApiError)
	DeleteUser(userId string) *shared.ApiError

	ListOrgRoles() ([]*shared.OrgRole, *shared.ApiError)

	InviteUser(req shared.InviteRequest) *shared.ApiError
	ListPendingInvites() ([]*shared.Invite, *shared.ApiError)
	ListAcceptedInvites() ([]*shared.Invite, *shared.ApiError)
	ListAllInvites() ([]*shared.Invite, *shared.ApiError)
	DeleteInvite(inviteId string) *shared.ApiError

	CreateProject(req shared.CreateProjectRequest) (*shared.CreateProjectResponse, *shared.ApiError)
	ListProjects() ([]*shared.Project, *shared.ApiError)
	SetProjectPlan(projectId string, req shared.SetProjectPlanRequest) *shared.ApiError
	RenameProject(projectId string, req shared.RenameProjectRequest) *shared.ApiError

	ListPlans(projectIds []string) ([]*shared.Plan, *shared.ApiError)
	ListArchivedPlans(projectIds []string) ([]*shared.Plan, *shared.ApiError)
	ListPlansRunning(projectIds []string, includeRecent bool) (*shared.ListPlansRunningResponse, *shared.ApiError)

	GetCurrentBranchByPlanId(projectId string, req shared.GetCurrentBranchByPlanIdRequest) (map[string]*shared.Branch, *shared.ApiError)

	GetPlan(planId string) (*shared.Plan, *shared.ApiError)
	CreatePlan(projectId string, req shared.CreatePlanRequest) (*shared.CreatePlanResponse, *shared.ApiError)

	TellPlan(planId, branch string, req shared.TellPlanRequest, onStreamPlan OnStreamPlan) *shared.ApiError
	BuildPlan(planId, branch string, req shared.BuildPlanRequest, onStreamPlan OnStreamPlan) *shared.ApiError
	RespondMissingFile(planId, branch string, req shared.RespondMissingFileRequest) *shared.ApiError

	DeletePlan(planId string) *shared.ApiError
	DeleteAllPlans(projectId string) *shared.ApiError
	ConnectPlan(planId, branch string, onStreamPlan OnStreamPlan) *shared.ApiError
	StopPlan(planId, branch string) *shared.ApiError

	ArchivePlan(planId string) *shared.ApiError

	GetCurrentPlanState(planId, branch string) (*shared.CurrentPlanState, *shared.ApiError)
	ApplyPlan(planId, branch string) *shared.ApiError
	RejectAllChanges(planId, branch string) *shared.ApiError
	RejectFile(planId, branch, filePath string) *shared.ApiError

	LoadContext(planId, branch string, req shared.LoadContextRequest) (*shared.LoadContextResponse, *shared.ApiError)
	UpdateContext(planId, branch string, req shared.UpdateContextRequest) (*shared.UpdateContextResponse, *shared.ApiError)
	DeleteContext(planId, branch string, req shared.DeleteContextRequest) (*shared.DeleteContextResponse, *shared.ApiError)
	ListContext(planId, branch string) ([]*shared.Context, *shared.ApiError)

	ListConvo(planId, branch string) ([]*shared.ConvoMessage, *shared.ApiError)
	ListLogs(planId, branch string) (*shared.LogResponse, *shared.ApiError)
	RewindPlan(planId, branch string, req shared.RewindPlanRequest) (*shared.RewindPlanResponse, *shared.ApiError)

	ListBranches(planId string) ([]*shared.Branch, *shared.ApiError)
	DeleteBranch(planId, branch string) *shared.ApiError
	CreateBranch(planId, branch string, req shared.CreateBranchRequest) *shared.ApiError

	GetSettings(planId, branch string) (*shared.PlanSettings, *shared.ApiError)
	UpdateSettings(planId, branch string, req shared.UpdateSettingsRequest) (*shared.UpdateSettingsResponse, *shared.ApiError)
}
