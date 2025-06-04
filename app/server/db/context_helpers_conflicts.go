package db

import (
	"fmt"
	"log"
	shared "plandex-shared"
	"runtime"
	"runtime/debug"
)

type invalidateConflictedResultsParams struct {
	orgId         string
	planId        string
	filesToUpdate map[string]string
	descriptions  []*ConvoMessageDescription
	currentPlan   *shared.CurrentPlanState
}

func invalidateConflictedResults(params invalidateConflictedResultsParams) error {
	orgId := params.orgId
	planId := params.planId
	filesToUpdate := params.filesToUpdate

	var descriptions []*ConvoMessageDescription

	if params.descriptions == nil {
		var err error
		descriptions, err = GetConvoMessageDescriptions(orgId, planId)
		if err != nil {
			return fmt.Errorf("error getting pending build descriptions: %v", err)
		}
	} else {
		descriptions = params.descriptions
	}

	var currentPlan *shared.CurrentPlanState

	if params.currentPlan == nil {
		var err error
		currentPlan, err = GetCurrentPlanState(CurrentPlanStateParams{
			OrgId:                    orgId,
			PlanId:                   planId,
			ConvoMessageDescriptions: descriptions,
		})
		if err != nil {
			return fmt.Errorf("error getting current plan state: %v", err)
		}
	} else {
		currentPlan = params.currentPlan
	}

	conflictPaths := currentPlan.PlanResult.FileResultsByPath.ConflictedPaths(filesToUpdate)

	// log.Println("invalidateConflictedResults - Conflicted paths:", conflictPaths)

	if len(conflictPaths) > 0 {
		toUpdateDescs := []*ConvoMessageDescription{}

		for _, desc := range descriptions {
			if !desc.DidBuild || desc.AppliedAt != nil {
				continue
			}

			for _, op := range desc.Operations {
				if _, found := conflictPaths[op.Path]; found {
					if desc.BuildPathsInvalidated == nil {
						desc.BuildPathsInvalidated = make(map[string]bool)
					}
					desc.BuildPathsInvalidated[op.Path] = true
					toUpdateDescs = append(toUpdateDescs, desc)
				}
			}
		}

		numRoutines := len(toUpdateDescs) + 1
		errCh := make(chan error, numRoutines)

		for _, desc := range toUpdateDescs {
			go func(desc *ConvoMessageDescription) {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("panic in StoreDescription: %v\n%s", r, debug.Stack())
						errCh <- fmt.Errorf("panic in StoreDescription: %v\n%s", r, debug.Stack())
						runtime.Goexit() // don't allow outer function to continue and double-send to channel
					}
				}()

				err := StoreDescription(desc)

				if err != nil {
					errCh <- fmt.Errorf("error storing description: %v", err)
					return
				}

				errCh <- nil
			}(desc)
		}

		go func() {
			err := DeletePendingResultsForPaths(orgId, planId, conflictPaths)

			if err != nil {
				errCh <- fmt.Errorf("error deleting pending results: %v", err)
				return
			}

			errCh <- nil
		}()

		for i := 0; i < numRoutines; i++ {
			err := <-errCh
			if err != nil {
				return fmt.Errorf("error storing description: %v", err)
			}
		}
	}

	return nil
}
