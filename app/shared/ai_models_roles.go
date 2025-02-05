package shared

type ModelRole string

const (
	ModelRolePlanner          ModelRole = "planner"
	ModelRoleCoder            ModelRole = "coder"
	ModelRoleContextLoader    ModelRole = "context-loader"
	ModelRolePlanSummary      ModelRole = "summarizer"
	ModelRoleBuilder          ModelRole = "builder"
	ModelRoleWholeFileBuilder ModelRole = "whole-file-builder"
	ModelRoleName             ModelRole = "names"
	ModelRoleCommitMsg        ModelRole = "commit-messages"
	ModelRoleExecStatus       ModelRole = "auto-continue"
)

var AllModelRoles = []ModelRole{ModelRolePlanner, ModelRoleCoder, ModelRoleContextLoader, ModelRolePlanSummary, ModelRoleBuilder, ModelRoleWholeFileBuilder, ModelRoleName, ModelRoleCommitMsg, ModelRoleExecStatus}

var ModelRoleDescriptions = map[ModelRole]string{
	ModelRolePlanner:          "replies to prompts and makes plans",
	ModelRoleCoder:            "writes code to implement a plan",
	ModelRolePlanSummary:      "summarizes conversations exceeding max-convo-tokens",
	ModelRoleBuilder:          "builds a plan into file diffs",
	ModelRoleWholeFileBuilder: "builds a plan into file diffs by writing the entire file",
	ModelRoleName:             "names plans",
	ModelRoleCommitMsg:        "writes commit messages",
	ModelRoleExecStatus:       "determines whether to auto-continue",
	ModelRoleContextLoader:    "decides what context to load using codebase map",
}
