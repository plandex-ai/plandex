package plan

import (
	"fmt"
	"log"
	"plandex-server/db"
	"plandex-server/types"
	"time"
)

type verifyState struct {
	proposedChanges   string
	preBuildFileState string
}

func (fileState *activeBuildStreamFileState) GetVerifyState() (*verifyState, error) {
	planState := fileState.currentPlanState
	convo := fileState.convo
	path := fileState.filePath

	fileResults := planState.PlanResult.FileResultsByPath[path]

	if fileResults == nil {
		return nil, fmt.Errorf("no file results for path: %s", path)
	}

	proposedChanges := ""
	convoMessageIds := map[string]bool{}
	startingReplacementId := ""

	for _, fileResult := range fileResults {
		if !fileResult.IsPending() {
			continue
		}

		if fileResult.CanVerify && fileResult.RanVerifyAt != nil {
			convoMessageIds = map[string]bool{}
			startingReplacementId = ""
		} else if !fileResult.IsFix {
			convoMessageIds[fileResult.ConvoMessageId] = true
			if len(fileResult.Replacements) > 0 {
				startingReplacementId = fileResult.Replacements[0].Id
			}
		}
	}

	for _, convoMessage := range convo {
		if convoMessageIds[convoMessage.Id] {
			parser := types.NewReplyParser()
			parser.AddChunk(convoMessage.Message, false)
			parserRes := parser.FinishAndRead()

			for i, file := range parserRes.Files {
				if file == path {
					desc := parserRes.FileDescriptions[i]
					fileContents := parserRes.FileContents[i]

					proposedChanges += fmt.Sprintf("%s\n\n```\n%s\n```\n\n", desc, fileContents)
				}
			}

		}
	}

	var preBuildState string

	if startingReplacementId == "" {
		if planState.ContextsByPath[path] != nil {
			preBuildState = planState.ContextsByPath[path].Body
		}
	} else {
		files, err := planState.GetFilesBeforeReplacement(startingReplacementId)

		if err != nil {
			return nil, fmt.Errorf("error getting files before replacement: %v", err)
		}

		if files.Files[path] == "" {
			return nil, fmt.Errorf("no file content before replacement")
		}

		preBuildState = files.Files[path]
	}

	return &verifyState{
		proposedChanges:   proposedChanges,
		preBuildFileState: preBuildState,
	}, nil

}

func (fileState *activeBuildStreamFileState) MarkLatestResultVerified(passed bool) error {
	planId := fileState.plan.Id
	branch := fileState.branch
	currentOrgId := fileState.currentOrgId
	currentUserId := fileState.currentUserId
	build := fileState.build

	activePlan := GetActivePlan(planId, branch)

	repoLockId, err := db.LockRepo(
		db.LockRepoParams{
			OrgId:       currentOrgId,
			UserId:      currentUserId,
			PlanId:      planId,
			Branch:      branch,
			PlanBuildId: build.Id,
			Scope:       db.LockScopeWrite,
			Ctx:         activePlan.Ctx,
			CancelFn:    activePlan.CancelFn,
		},
	)
	if err != nil {
		log.Printf("Error locking repo for build file: %v\n", err)
		return fmt.Errorf("error locking repo for build file: %v", err)
	}

	err = func() error {
		var err error
		defer func() {
			if err != nil {
				log.Printf("Error: %v\n", err)
				err = db.GitClearUncommittedChanges(currentOrgId, planId)
				if err != nil {
					log.Printf("Error clearing uncommitted changes: %v\n", err)
				}
			}

			err := db.UnlockRepo(repoLockId)
			if err != nil {
				log.Printf("Error unlocking repo: %v\n", err)
			}
		}()

		results, err := db.GetPlanFileResults(currentOrgId, planId)

		if err != nil {
			return fmt.Errorf("error getting plan file results: %v", err)
		}

		if len(results) == 0 {
			return fmt.Errorf("no file results for path: %s", fileState.filePath)
		}

		latestPlanRes := results[len(results)-1]
		now := time.Now()
		latestPlanRes.RanVerifyAt = &now
		latestPlanRes.VerifyPassed = passed

		err = db.StorePlanResult(latestPlanRes)
		if err != nil {
			log.Printf("Error storing plan result: %v\n", err)
			return fmt.Errorf("error storing plan result: %v", err)
		}
		return nil
	}()

	if err != nil {
		return err
	}

	return nil
}
