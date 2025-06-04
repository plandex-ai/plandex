package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	shared "plandex-shared"
	"runtime"
	"runtime/debug"
)

func ContextRemove(orgId, planId string, contexts []*Context) error {
	return contextRemove(contextRemoveParams{
		orgId:    orgId,
		planId:   planId,
		contexts: contexts,
	})
}

type contextRemoveParams struct {
	orgId        string
	planId       string
	contexts     []*Context
	descriptions []*ConvoMessageDescription
	currentPlan  *shared.CurrentPlanState
}

func contextRemove(params contextRemoveParams) error {
	orgId := params.orgId
	planId := params.planId
	contexts := params.contexts

	// remove files
	numFiles := 0

	filesToUpdate := make(map[string]string)

	errCh := make(chan error, numFiles)
	for _, context := range contexts {
		filesToUpdate[context.FilePath] = ""
		contextDir := getPlanContextDir(orgId, planId)
		for _, ext := range []string{".meta", ".body", ".map-parts"} {
			numFiles++
			go func(context *Context, dir, ext string) {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("panic in contextRemove: %v\n%s", r, debug.Stack())
						errCh <- fmt.Errorf("panic in contextRemove: %v\n%s", r, debug.Stack())
						runtime.Goexit() // don't allow outer function to continue and double-send to channel
					}
				}()
				errCh <- os.Remove(filepath.Join(dir, context.Id+ext))
			}(context, contextDir, ext)
		}
	}

	for i := 0; i < numFiles; i++ {
		err := <-errCh
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error removing context file: %v", err)
		}
	}

	err := invalidateConflictedResults(invalidateConflictedResultsParams{
		orgId:         orgId,
		planId:        planId,
		filesToUpdate: filesToUpdate,
		descriptions:  params.descriptions,
		currentPlan:   params.currentPlan,
	})
	if err != nil {
		return fmt.Errorf("error invalidating conflicted results: %v", err)
	}

	return nil
}

type ClearContextParams struct {
	OrgId       string
	PlanId      string
	SkipMaps    bool
	SkipPending bool
}

func ClearContext(params ClearContextParams) error {
	orgId := params.OrgId
	planId := params.PlanId
	skipMaps := params.SkipMaps
	skipPending := params.SkipPending

	contexts, err := GetPlanContexts(orgId, planId, false, false)
	if err != nil {
		return fmt.Errorf("error getting plan contexts: %v", err)
	}

	var descriptions []*ConvoMessageDescription
	var currentPlan *shared.CurrentPlanState

	if skipPending {
		var err error
		descriptions, err = GetConvoMessageDescriptions(orgId, planId)
		if err != nil {
			return fmt.Errorf("error getting pending build descriptions: %v", err)
		}

		currentPlan, err = GetCurrentPlanState(CurrentPlanStateParams{
			OrgId:                    orgId,
			PlanId:                   planId,
			ConvoMessageDescriptions: descriptions,
		})

		if err != nil {
			return fmt.Errorf("error getting current plan state: %v", err)
		}
	}

	toRemove := []*Context{}

	for _, context := range contexts {
		shouldSkip := false

		if !(skipMaps && context.ContextType == shared.ContextMapType) {
			shouldSkip = true
		}

		if skipPending && currentPlan.CurrentPlanFiles.Files[context.FilePath] != "" {
			shouldSkip = true
		}

		if !shouldSkip {
			toRemove = append(toRemove, context)
		}
	}

	if len(toRemove) > 0 {
		err := contextRemove(contextRemoveParams{
			orgId:        orgId,
			planId:       planId,
			contexts:     toRemove,
			descriptions: descriptions,
			currentPlan:  currentPlan,
		})
		if err != nil {
			return fmt.Errorf("error removing non-map contexts: %v", err)
		}
	}

	return nil
}
