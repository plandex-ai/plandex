package plan

import (
	"context"
	"errors"
	"fmt"
	"log"
	"plandex-server/syntax"
	"plandex-server/utils"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

type raceResult struct {
	content string
	valid   bool
}

type buildRaceParams struct {
	updated         string
	proposedContent string
	desc            string
	reasons         []syntax.NeedsVerifyReason
	syntaxErrors    []string

	didCallFastApply bool
	fastApplyCh      chan string

	sessionId string
}

func (fileState *activeBuildStreamFileState) buildRace(
	buildCtx context.Context,
	cancelBuild context.CancelFunc,
	params buildRaceParams,
) (raceResult, error) {
	log.Printf("buildRace - starting race for file")
	defer func() {
		log.Printf("buildRace - canceling build context")
		cancelBuild()
	}()

	originalFile := fileState.preBuildState

	updated := params.updated
	proposedContent := params.proposedContent
	desc := params.desc
	reasons := params.reasons
	syntaxErrors := params.syntaxErrors
	fastApplyCh := params.fastApplyCh
	sessionId := params.sessionId
	log.Printf("buildRace - original file length: %d, updated length: %d", len(originalFile), len(updated))
	log.Printf("buildRace - has %d syntax errors and %d verify reasons", len(syntaxErrors), len(reasons))

	maxErrs := 3

	resCh := make(chan raceResult, 1)
	errCh := make(chan error, maxErrs)

	sendRes := func(res raceResult) {
		select {
		case resCh <- res:
		case <-buildCtx.Done():
			log.Printf("buildRace - context canceled, skipping sendRes")
		}
	}

	sendErr := func(err error) {
		select {
		case errCh <- err:
		case <-buildCtx.Done():
			log.Printf("buildRace - context canceled, skipping sendErr")
		}
	}

	startedFallbacks := false

	startWholeFileBuild := func(comments string) {
		log.Printf("buildRace - starting whole file fallback build")
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in startWholeFileBuild: %v\n%s", r, debug.Stack())
					sendErr(fmt.Errorf("error starting whole file build: %v", r))
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			select {
			case <-buildCtx.Done():
				log.Printf("buildRace - context already canceled, skipping whole file build")
				return
			default:
			}

			content, err := fileState.buildWholeFileFallback(buildCtx, proposedContent, desc, comments, sessionId)

			if err != nil {
				if errors.Is(err, context.Canceled) {
					log.Printf("Context canceled during whole file build")
					return
				}

				log.Printf("buildRace - whole file build failed: %v", err)
				sendErr(fmt.Errorf("error building whole file: %w", err))
			} else {
				log.Printf("buildRace - whole file build succeeded")
				sendRes(raceResult{content: content, valid: true})
			}
		}()
	}

	maybeStartFastApply := func(onFail func()) {
		log.Printf("buildRace - starting fast apply")
		if !params.didCallFastApply {
			log.Printf("buildRace - fast apply isn't defined, skipping")
			sendErr(nil) // no error, just no fast apply
			onFail()
			return
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in maybeStartFastApply: %v\n%s", r, debug.Stack())
					sendErr(fmt.Errorf("error starting fast apply: %v", r))
					onFail()
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			var fastApplyRes string

			select {
			case fastApplyRes = <-fastApplyCh:
			case <-buildCtx.Done():
				log.Printf("buildRace - context canceled, skipping fast apply")
				sendErr(nil) // no error, just no fast apply
				onFail()
				return
			}

			if fastApplyRes == "" {
				log.Printf("buildRace - fast apply isn't defined or failed to run")
				sendErr(nil) // no error, just no fast apply
				onFail()
				return
			}

			// log.Printf("buildRace - fast apply result:\n\n%s", fastApplyRes)

			fastApplySyntaxErrors := fileState.validateSyntax(buildCtx, fastApplyRes)
			fileState.builderRun.FastApplySyntaxErrors = fastApplySyntaxErrors

			if len(fastApplySyntaxErrors) > 0 {
				log.Printf("buildRace - fast apply succeeded, but has %d syntax errors", len(fastApplySyntaxErrors))
				sendErr(fmt.Errorf("fast apply succeeded, but has %d syntax errors", len(fastApplySyntaxErrors)))
				onFail()
				return
			}

			log.Printf("buildRace - fast apply returned, validating...	")
			validateResult, err := fileState.buildValidateLoop(buildCtx, buildValidateLoopParams{
				originalFile:    originalFile,
				updated:         fastApplyRes,
				proposedContent: proposedContent,
				desc:            desc,
				reasons:         reasons,

				// just validate since we're already building replacements in parallel
				maxAttempts:                1,
				validateOnlyOnFinalAttempt: true,
				isInitial:                  false,
				sessionId:                  sessionId,
			})

			if err != nil {
				if errors.Is(err, context.Canceled) {
					log.Printf("Context canceled during fast apply validation")
					return
				}

				log.Printf("buildRace - fast apply validation failed with error: %v", err)
				sendErr(fmt.Errorf("fast apply validation failed: %w", err))
				onFail()
				return
			}

			if validateResult.valid {
				log.Printf("buildRace - fast apply validation succeeded")
				fileState.builderRun.FastApplySuccess = true
				sendRes(raceResult{content: validateResult.updated, valid: validateResult.valid})
			} else {
				log.Printf("buildRace - fast apply validation failed with problem: %s", validateResult.problem)
				fileState.builderRun.FastApplyFailureResponse = validateResult.problem
				sendErr(fmt.Errorf("fast apply validation failed: %s", validateResult.problem))
				onFail()
				return
			}
		}()
	}

	startFallbacks := func(comments string) {
		startedFallbacks = true
		// try fast apply + validation first if it's defined
		// if it's undefined or fails, start the whole file build fallback
		maybeStartFastApply(func() {
			startWholeFileBuild(comments)
		})
	}

	// If we get an incorrect marker, start the whole file build in the background while the validation/replacement loop continues
	onInitialStream := func(chunk string, buffer string) bool {
		if !startedFallbacks && strings.Contains(buffer, "<PlandexIncorrect/>") && strings.Contains(buffer, "<PlandexComments>") {
			log.Printf("buildRace - detected incorrect marker, triggering whole file build")

			comments := utils.GetXMLContent(buffer, "PlandexComments")

			startFallbacks(comments)
		}
		// keep streaming
		return false
	}

	fileState.builderRun.AutoApplyValidationStartedAt = time.Now()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in buildRace validation loop: %v\n%s", r, debug.Stack())
				sendErr(fmt.Errorf("error building validate loop: %v", r))
				runtime.Goexit() // don't allow outer function to continue and double-send to channel
			}
		}()

		log.Printf("buildRace - starting validation loop")
		validateResult, err := fileState.buildValidateLoop(buildCtx, buildValidateLoopParams{
			originalFile:         originalFile,
			updated:              updated,
			proposedContent:      proposedContent,
			desc:                 desc,
			reasons:              reasons,
			syntaxErrors:         syntaxErrors,
			initialPhaseOnStream: onInitialStream,
			isInitial:            true,
			sessionId:            sessionId,
		})

		fileState.builderRun.AutoApplyValidationFinishedAt = time.Now()

		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Printf("Context canceled during buildValidate")
				return
			}

			log.Printf("buildRace - validation loop failed: %v", err)
			sendErr(fmt.Errorf("error building validate loop: %w", err))
		} else {
			log.Printf("buildRace - validation loop finished, valid: %v", validateResult.valid)
			if validateResult.valid {
				log.Printf("buildRace - validation loop succeeded, valid: %v", validateResult.valid)
				sendRes(raceResult{content: validateResult.updated, valid: validateResult.valid})
			} else {
				log.Printf("buildRace - validation loop failed, valid: %v", validateResult.valid)
				sendErr(fmt.Errorf("validation loop failed: %s", validateResult.problem))
			}
		}
	}()

	errs := []error{}
	errChNumReceived := 0

	for {
		select {
		case <-buildCtx.Done():
			log.Printf("buildRace - context canceled")
			return raceResult{}, buildCtx.Err()
		case err := <-errCh:
			errChNumReceived++
			log.Printf("buildRace - error channel received %d: %v\n", errChNumReceived, err)

			if err != nil {
				errs = append(errs, err)
			}

			if errChNumReceived >= maxErrs {
				log.Printf("buildRace - all attempts failed with %d errors", len(errs))
				return raceResult{}, fmt.Errorf("all build attempts failed: %v", errs)
			}

			if !startedFallbacks {
				log.Printf("buildRace - starting build fallbacks")
				startFallbacks("") // since replacements failed, pass an empty string for comments -- this causes whole file build to classify comments first
			}
		case res := <-resCh:
			log.Printf("buildRace - got successful result")
			return res, nil
		}
	}
}
