package types

import (
	"fmt"
	"os"
)

type ApplyRollbackOption string

const (
	ApplyRollbackOptionKeep     ApplyRollbackOption = "Still apply other changes"
	ApplyRollbackOptionRollback ApplyRollbackOption = "Roll back all changes"
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

func (r *ApplyRollbackPlan) Rollback(msg bool) error {
	errCh := make(chan error, len(r.ToRevert)+len(r.ToRemove))

	for path, revert := range r.ToRevert {
		go func(path string, revert ApplyReversion) {
			err := os.WriteFile(path, []byte(revert.Content), revert.Mode)
			if err != nil {
				errCh <- fmt.Errorf("failed to write %s: %s", path, err.Error())
				return
			}
			errCh <- nil
		}(path, revert)
	}

	for _, path := range r.ToRemove {
		go func(path string) {
			err := os.RemoveAll(path)
			if err != nil {
				errCh <- fmt.Errorf("failed to remove %s: %s", path, err.Error())
				return
			}
			errCh <- nil
		}(path)
	}

	errs := []error{}

	for i := 0; i < len(r.ToRevert)+len(r.ToRemove); i++ {
		err := <-errCh
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to rollback: %s", errs)
	}

	if msg {
		fmt.Println("🚫 Rolled back all changes")
	}

	return nil
}
