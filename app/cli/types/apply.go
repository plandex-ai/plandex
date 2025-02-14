package types

import (
	"os"
)

type ApplyFlags struct {
	AutoConfirm bool
	AutoCommit  bool
	NoCommit    bool
	AutoExec    bool
	NoExec      bool
	AutoDebug   int
}

type ApplyRollbackOption string

const (
	ApplyRollbackOptionKeep     ApplyRollbackOption = "Apply file changes"
	ApplyRollbackOptionRollback ApplyRollbackOption = "Roll back file changes"
)

type OnApplyExecFailFn func(status int, output string, attempt int, toRollback *ApplyRollbackPlan, onErr OnErrFn, onSuccess func())

type ApplyReversion struct {
	Content string
	Mode    os.FileMode
}

type ApplyRollbackPlan struct {
	ToRevert             map[string]ApplyReversion
	ToRemove             []string
	PreviousProjectPaths *ProjectPaths
}

func (r *ApplyRollbackPlan) HasChanges() bool {
	return len(r.ToRevert) > 0 || len(r.ToRemove) > 0
}
