package plan

import (
	"log"
	"math/rand"
	"plandex-server/types"
	"time"

	"github.com/plandex/plandex/shared"
)

func (fileState *activeBuildStreamFileState) onVerifyResult(res types.VerifyResult) {

	filePath := fileState.filePath
	planId := fileState.plan.Id
	branch := fileState.branch

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		log.Printf("listenStreamVerifyOutput - Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return
	}

	if res.IsCorrect() {
		buildInfo := &shared.BuildInfo{
			Path:      filePath,
			NumTokens: 0,
			Finished:  true,
		}
		activePlan.Stream(shared.StreamMessage{
			Type:      shared.StreamMessageBuildInfo,
			BuildInfo: buildInfo,
		})
		log.Println("build verify - res.IsCorrect")

		time.Sleep(50 * time.Millisecond)

		fileState.onFinishBuildFile(nil, "")
	} else {

		log.Printf("listenStreamVerifyOutput - File %s: Streamed verify result is incorrect\n", filePath)

		fileState.verificationErrors = res.GetReasoning()
		fileState.isFixingOther = true

		// log.Println("Verification errors:")
		// log.Println(fileState.verificationErrors)

		select {
		case <-activePlan.Ctx.Done():
			log.Println("listenStreamVerifyOutput - Context canceled. Exiting.")
			return
		case <-time.After(time.Duration(rand.Intn(1001)) * time.Millisecond):
			break
		}
		fileState.fixFileLineNums()
	}

}

func (fileState *activeBuildStreamFileState) verifyRetryOrAbort(err error) {
	if fileState.verifyFileNumRetry < MaxBuildStreamErrorRetries {
		fileState.verifyFileNumRetry++
		fileState.activeBuild.VerifyBuffer = ""
		fileState.activeBuild.VerifyBufferTokens = 0
		log.Printf("Retrying verify file '%s' due to error: %v\n", fileState.filePath, err)

		// Exponential backoff
		time.Sleep(time.Duration((fileState.verifyFileNumRetry*fileState.verifyFileNumRetry)/2)*200*time.Millisecond + time.Duration(rand.Intn(500))*time.Millisecond)

		fileState.verifyFileBuild()
	} else {
		log.Printf("Aborting verify file '%s' due to error: %v\n", fileState.filePath, err)

		fileState.onFinishBuildFile(nil, "")
	}
}
