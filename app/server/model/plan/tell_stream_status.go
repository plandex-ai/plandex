package plan

import (
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	shared "plandex-shared"
	"runtime/debug"
)

type handleDescAndExecStatusResult struct {
	handleStreamFinishedResult
	subtaskFinished      bool
	generatedDescription *db.ConvoMessageDescription
}

func (state *activeTellStreamState) handleDescAndExecStatus() handleDescAndExecStatusResult {
	currentOrgId := state.currentOrgId
	summarizedToMessageId := state.summarizedToMessageId
	planId := state.plan.Id
	branch := state.branch
	replyOperations := state.chunkProcessor.replyOperations

	active := GetActivePlan(planId, branch)
	if active == nil {
		state.onActivePlanMissingError()
		return handleDescAndExecStatusResult{
			handleStreamFinishedResult: handleStreamFinishedResult{
				shouldContinueMainLoop: true,
				shouldReturn:           false,
			},
		}
	}

	var generatedDescription *db.ConvoMessageDescription
	var subtaskFinished bool

	var errCh = make(chan *shared.ApiError, 2)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in genPlanDescription: %v\n%s", r, debug.Stack())
				errCh <- &shared.ApiError{
					Type:   shared.ApiErrorTypeOther,
					Status: http.StatusInternalServerError,
					Msg:    fmt.Sprintf("Error generating plan description: %v", r),
				}
			}
		}()

		if len(replyOperations) > 0 {
			log.Println("Generating plan description")

			res, err := state.genPlanDescription()
			if err != nil {
				errCh <- err
				return
			}

			generatedDescription = res
			generatedDescription.OrgId = currentOrgId
			generatedDescription.SummarizedToMessageId = summarizedToMessageId
			generatedDescription.WroteFiles = true
			generatedDescription.Operations = replyOperations

			log.Println("Generated plan description.")
		}
		errCh <- nil
	}()

	if state.currentStage.TellStage == shared.TellStageImplementation {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in execStatusShouldContinue: %v\n%s", r, debug.Stack())
					errCh <- &shared.ApiError{
						Type:   shared.ApiErrorTypeOther,
						Status: http.StatusInternalServerError,
						Msg:    fmt.Sprintf("Error getting exec status: %v", r),
					}
				}
			}()

			log.Println("Getting exec status")
			var err *shared.ApiError
			res, err := state.execStatusShouldContinue(active.CurrentReplyContent, active.SessionId, active.Ctx)
			if err != nil {
				errCh <- err
				return
			}

			subtaskFinished = res.subtaskFinished

			log.Printf("subtaskFinished: %v\n", subtaskFinished)

			errCh <- nil
		}()

	} else {
		errCh <- nil
	}

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			res := state.onError(onErrorParams{
				streamApiErr: err,
				storeDesc:    true,
			})
			return handleDescAndExecStatusResult{
				handleStreamFinishedResult: handleStreamFinishedResult{
					shouldContinueMainLoop: res.shouldContinueMainLoop,
					shouldReturn:           res.shouldReturn,
				},
				subtaskFinished:      subtaskFinished,
				generatedDescription: generatedDescription,
			}
		}
	}

	return handleDescAndExecStatusResult{
		handleStreamFinishedResult: handleStreamFinishedResult{},
		subtaskFinished:            subtaskFinished,
		generatedDescription:       generatedDescription,
	}
}

type willContinuePlanParams struct {
	hasNewSubtasks      bool
	removedSubtasks     bool
	allSubtasksFinished bool
	activatePaths       map[string]bool
	hasExplicitPaths    bool
}

func (state *activeTellStreamState) willContinuePlan(params willContinuePlanParams) bool {
	hasNewSubtasks := params.hasNewSubtasks
	removedSubtasks := params.removedSubtasks
	allSubtasksFinished := params.allSubtasksFinished
	activatePaths := params.activatePaths
	currentSubtask := state.currentSubtask

	log.Printf("[willContinuePlan] currentStage: %v", state.currentStage)

	log.Printf("[willContinuePlan] Initial state - hasNewSubtasks: %v, allSubtasksFinished: %v, tellStage: %v, planningPhase: %v, iteration: %d, autoContinue: %v",
		hasNewSubtasks, allSubtasksFinished, state.currentStage.TellStage, state.currentStage.PlanningPhase, state.iteration, state.req.AutoContinue)

	if state.currentStage.TellStage == shared.TellStagePlanning {
		log.Println("[willContinuePlan] In planning stage")

		// always continue to response or planning phase after context phase
		if state.currentStage.PlanningPhase == shared.PlanningPhaseContext {

			// if it's the context stage but it's chat mode and no files were loaded, don't continue
			if state.req.IsChatOnly && len(activatePaths) == 0 {
				log.Println("[willContinuePlan] Chat only - no files loaded - stopping")
				return false
			}

			// if no files were listed explicitly in a ### Files section, don't continue if it's chat mode
			if state.req.IsChatOnly && !params.hasExplicitPaths {
				log.Println("[willContinuePlan] Chat only - no files loaded - stopping")
				return false
			}

			log.Println("[willContinuePlan] In context phase - continuing to planning phase")
			return true
		}

		if state.req.IsChatOnly {
			log.Println("[willContinuePlan] Chat only - stopping")
			return false
		}

		// otherwise, if auto-continue is disabled, never continue
		if !state.req.AutoContinue {
			log.Println("[willContinuePlan] Auto-continue disabled - stopping")
			return false
		}

		// if there are new subtasks, continue
		if hasNewSubtasks {
			log.Println("[willContinuePlan] Has new subtasks - continuing")
			return true
		}

		if removedSubtasks && !allSubtasksFinished {
			log.Println("[willContinuePlan] Removed subtasks - continuing")
			return true
		}

		// if all subtasks are finished, don't continue
		log.Printf("[willContinuePlan] Checking subtasks finished - allSubtasksFinished: %v, will continue: %v",
			allSubtasksFinished, !allSubtasksFinished)

		log.Printf("[willContinuePlan] currentSubtask: %v", currentSubtask)

		return !allSubtasksFinished && currentSubtask != nil

	} else if state.currentStage.TellStage == shared.TellStageImplementation {
		log.Println("[willContinuePlan] In implementation stage")

		// if all subtasks are finished, don't continue
		if allSubtasksFinished {
			log.Println("[willContinuePlan] All subtasks finished - stopping")
			return false
		}

		// if we've automatically continued too many times, don't continue
		if state.iteration >= MaxAutoContinueIterations {
			log.Printf("[willContinuePlan] Reached max iterations (%d) - stopping", MaxAutoContinueIterations)
			return false
		}

		// otherwise, continue with implementation
		log.Println("[willContinuePlan] Continuing implementation")
		return true
	}

	log.Printf("[willContinuePlan] Unknown tell stage: %v - won't continue", state.currentStage.TellStage)
	return false
}
