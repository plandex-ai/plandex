package api

import (
	"github.com/looplab/fsm"
	"github.com/plandex/plandex/shared"
)

type OnStreamPlanParams struct {
	Content string
	State   *fsm.FSM
	Err     error
}

type OnStreamPlan func(params OnStreamPlanParams)

type Api struct{}

type ApiClient interface {
	CreateProject(req shared.CreateProjectRequest) (*shared.CreateProjectResponse, error)
	ListProjects() ([]*shared.Project, error)
	SetProjectPlan(projectId string, req shared.SetProjectPlanRequest) error
	RenameProject(projectId string, req shared.RenameProjectRequest) error

	ListPlans(projectId string) ([]*shared.Plan, error)
	ListArchivedPlans(projectId string) ([]*shared.Plan, error)
	ListPlansRunning(projectId string) ([]*shared.Plan, error)
	GetPlan(planId string) (*shared.Plan, error)
	CreatePlan(projectId string, req shared.CreatePlanRequest) (*shared.CreatePlanResponse, error)

	TellPlan(planId string, req shared.TellPlanRequest, onStreamPlan OnStreamPlan) error
	DeletePlan(planId string) error
	DeleteAllPlans(projectId string) error
	ConnectPlan(planId string, onStreamPlan OnStreamPlan) error
	StopPlan(planId string) error

	ArchivePlan(planId string) error

	GetCurrentPlanState(planId string) (*shared.CurrentPlanState, error)
	ApplyPlan(planId string) error
	RejectAllChanges(planId string) error
	RejectResult(planId, resultId string) error
	RejectReplacement(planId, resultId, replacementId string) error

	LoadContext(planId string, req shared.LoadContextRequest) (*shared.LoadContextResponse, error)
	UpdateContext(planId string, req shared.UpdateContextRequest) (*shared.UpdateContextResponse, error)
	DeleteContext(planId string, req shared.DeleteContextRequest) (*shared.DeleteContextResponse, error)
	ListContext(planId string) ([]*shared.Context, error)

	ListConvo(planId string) ([]*shared.ConvoMessage, error)
	ListLogs(planId string) (*shared.LogResponse, error)
	RewindPlan(planId string, req shared.RewindPlanRequest) (*shared.RewindPlanResponse, error)
}
