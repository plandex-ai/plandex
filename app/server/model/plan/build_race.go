package plan

import (
	"context"
	"fmt"
	"log"
	"plandex-server/syntax"
	"plandex-server/utils"
	"strings"
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
	blocksRemoved   []struct {
		Start   int
		End     int
		Content string
	}
}

func (fileState *activeBuildStreamFileState) buildRace(
	buildCtx context.Context,
	cancelBuild context.CancelFunc,
	params buildRaceParams,
) (raceResult, error) {
	log.Printf("buildRace - starting race for file")
	defer cancelBuild()

	originalFile := fileState.preBuildState

	updated := params.updated
	proposedContent := params.proposedContent
	desc := params.desc
	reasons := params.reasons
	syntaxErrors := params.syntaxErrors

	log.Printf("buildRace - original file length: %d, updated length: %d", len(originalFile), len(updated))
	log.Printf("buildRace - has %d syntax errors and %d verify reasons", len(syntaxErrors), len(reasons))

	resCh := make(chan raceResult, 1)
	errCh := make(chan error, 2)

	startedWholeFileBuild := false

	startWholeFileBuild := func(comments string) {
		log.Printf("buildRace - starting whole file fallback build")
		startedWholeFileBuild = true
		go func() {
			content, err := fileState.buildWholeFileFallback(buildCtx, proposedContent, desc, comments)

			if err != nil {
				log.Printf("buildRace - whole file build failed: %v", err)
				errCh <- fmt.Errorf("error building whole file: %w", err)
			} else {
				log.Printf("buildRace - whole file build succeeded")
				resCh <- raceResult{content: content, valid: true}
			}
		}()
	}

	// If we get an incorrect marker, start the whole file build in the background while the validation/replacement loop continues
	onInitialStream := func(chunk string, buffer string) bool {
		if !startedWholeFileBuild && strings.Contains(buffer, "<PlandexIncorrect/>") && strings.Contains(buffer, "<PlandexComments>") {
			log.Printf("buildRace - detected incorrect marker, triggering whole file build")

			comments := utils.GetXMLContent(buffer, "PlandexComments")

			startWholeFileBuild(comments)
		}
		// keep streaming
		return false
	}

	go func() {
		log.Printf("buildRace - starting validation loop")
		validateResult, err := fileState.buildValidateLoop(buildCtx, buildValidateLoopParams{
			originalFile:         originalFile,
			updated:              updated,
			proposedContent:      proposedContent,
			desc:                 desc,
			reasons:              reasons,
			syntaxErrors:         syntaxErrors,
			initialPhaseOnStream: onInitialStream,
		})

		if err != nil {
			log.Printf("buildRace - validation loop failed: %v", err)
			errCh <- fmt.Errorf("error building validate loop: %w", err)
		} else {
			log.Printf("buildRace - validation loop succeeded, valid: %v", validateResult.valid)
			resCh <- raceResult{content: validateResult.updated, valid: validateResult.valid}
		}
	}()

	errs := []error{}

	for {
		select {
		case err := <-errCh:
			log.Printf("buildRace - error %d: %v\n", len(errs), err)
			errs = append(errs, err)

			if len(errs) > 1 {
				log.Printf("buildRace - all attempts failed with %d errors", len(errs))
				return raceResult{}, fmt.Errorf("all build attempts failed: %v", errs)
			}

			if !startedWholeFileBuild {
				log.Printf("buildRace - starting whole file build")
				startWholeFileBuild("") // since replacements failed, pass an empty string for comments -- this causes whole file build to classify comments first
			}
		case res := <-resCh:
			log.Printf("buildRace - got successful result")
			return res, nil
		}
	}
}
