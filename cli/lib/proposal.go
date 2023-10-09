package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"plandex/types"
	"time"

	"github.com/looplab/fsm"
	"github.com/plandex/plandex/shared"

	"github.com/briandowns/spinner"
)

type key struct {
	value rune
}

func Propose(prompt string) error {
	fmt.Println("Sending prompt... ")
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Start()

	time.Sleep(500 * time.Millisecond)

	timestamp := StringTs()
	reply := ""
	done := make(chan struct{})

	termState := ""

	replyUpdateTimer := time.NewTimer(100 * time.Millisecond)
	defer replyUpdateTimer.Stop()

	var proposalId string
	var replyStarted bool
	var terminalHasPendingUpdate bool
	var endedReply bool

	var descJson string
	var desc *shared.PlanDescription
	var state *fsm.FSM
	var streamFinished bool
	var filesFinished bool
	finishedByFile := make(map[string]bool)

	jsonBuffers := make(map[string]string)
	contextTokensByFile := make(map[string]int)

	replyTokenCounter := shared.NewReplyInfo(true)
	var tokensAddedByFile map[string]int

	printReply := func() {
		clearScreen()
		moveCursorToTopLeft()
		mdFull, _ := GetMarkdown(reply)
		fmt.Println(mdFull)
		fmt.Printf(displayHotkeys())
		termState = mdFull
	}

	go func() {
		for range replyUpdateTimer.C {
			if terminalHasPendingUpdate {
				printReply()
				terminalHasPendingUpdate = false
			}
			replyUpdateTimer.Reset(100 * time.Millisecond)
		}
	}()

	keyChan := make(chan *key, 1)
	ctx, cancelKeywatch := context.WithCancel(context.Background())
	errChn := make(chan error, 1)

	endReply := func() {
		replyUpdateTimer.Stop()

		backToMain()
		fmt.Print(termState)
		err := appendConversation(timestamp, prompt, reply)
		if err != nil {
			fmt.Printf("failed to append conversation: %s\n", err)
		}
		_, tokensAddedByFile = replyTokenCounter.FinishAndRead()
		endedReply = true
	}

	contextByFilePath := make(map[string]shared.ModelContextPart)

	running := false
	queue := make(chan types.OnStreamPlanParams, 1)

	var handleStream types.OnStreamPlan
	handleStream = func(params types.OnStreamPlanParams) {
		if running {
			queue <- params
			return
		}

		defer func() {
			if len(queue) > 0 {
				params := <-queue
				handleStream(params)
			} else {
				running = false
			}
		}()

		state = params.State
		err := params.Err
		content := params.Content

		onError := func(err error) {
			backToMain()
			fmt.Fprintln(os.Stderr, "Error:", err)
			cancelKeywatch()
			close(done)
		}

		if err != nil {
			onError(err)
			return
		}

		if proposalId == "" {
			if content == "" {
				onError(fmt.Errorf("proposal id not sent in first chunk"))
				return
			} else {
				proposalId = content
				return
			}
		} else if !replyStarted {
			replyStarted = true
			s.Stop()
			alternateScreen()
		}

		switch state.Current() {
		case shared.STATE_REPLYING, shared.STATE_REVISING:
			reply += content
			replyTokenCounter.AddChunk(content)
			terminalHasPendingUpdate = true

		case shared.STATE_FINISHED:
			if !endedReply {
				endReply()
			}
			streamFinished = true

			if filesFinished {
				close(done)
			}
			return

		case shared.STATE_DESCRIBING:
			if content == shared.STREAM_DESCRIPTION_PHASE {
				endReply()

			} else {
				descJson = content
				err := json.Unmarshal([]byte(descJson), &desc)
				if err != nil {
					onError(fmt.Errorf("error parsing plan description: %v", err))
					return
				}

				if desc.MadePlan && (len(desc.Files) > 0 /*|| desc.HasExec*/) {
					fmt.Println("Writing plan draft:")
					for _, filePath := range desc.Files {
						fmt.Printf("- %s\n", filePath)
					}
					// if desc.HasExec {
					// 	fmt.Printf("- %s\n", "exec.sh")
					// }
				} else {
					filesFinished = true
				}

			}

		case shared.STATE_BUILDING:
			if content == shared.STREAM_BUILD_PHASE {
				// plan build mode started

			} else {
				wroteFile, err := receiveFileChunk(content, desc, jsonBuffers, contextTokensByFile, finishedByFile)

				if err != nil {
					onError(err)
					return
				}

				// files := make([]string, len(desc.Files))
				// copy(files, desc.Files)
				// if desc.HasExec {
				// 	files = append(files, "exec.sh")
				// }

				files := desc.Files

				// Clear previous lines
				moveUpLines(len(files))

				for _, filePath := range files {
					contextPart, foundContext := contextByFilePath[filePath]
					contextTokens := contextTokensByFile[filePath]
					added := tokensAddedByFile[filePath]

					fmtStr := "- %s | %d tokens"
					fmtArgs := []interface{}{filePath, contextTokens}

					finished := finishedByFile[filePath]

					if finished {
						fmtStr += " | done ✅"
					} else {
						if foundContext {
							fmtStr += " / %d estimated (%d base + ~%d changes)"
							contextTotal := int(contextPart.NumTokens)
							total := contextTotal + added

							fmtArgs = append(fmtArgs, total, contextTotal, added)
						} else if added > 0 {
							fmtStr += " / %d estimated"
							fmtArgs = append(fmtArgs, added)
						}
					}

					clearCurrentLine()
					fmt.Printf(fmtStr+"\n", fmtArgs...)
				}

				if wroteFile {
					if len(finishedByFile) == len(files) {
						filesFinished = true

						if streamFinished {
							close(done)
						}
					}

				}

			}
		}

	}

	apiReq, err := Api.Propose(prompt, handleStream)
	if err != nil {
		backToMain()
		return fmt.Errorf("failed to send prompt to server: %s\n", err)
	}
	for _, part := range apiReq.ModelContext {
		contextByFilePath[part.FilePath] = part
	}

	go func(ctx context.Context, errChn chan error) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				k, err := getUserInput()
				if err != nil {
					errChn <- err
					return
				}
				keyChan <- &key{k}
			}
		}
	}(ctx, errChn)

	handleKey := func(k *key) error {
		return handleKeyPress(k.value, proposalId)
	}

Loop:
	for {
		select {
		case k := <-keyChan:
			if err := handleKey(k); err != nil {
				cancelKeywatch()
				return err
			}
		case <-done: // Evidence of operation completion
			cancelKeywatch()
			break Loop
		case err := <-errChn: // Listening for errors
			cancelKeywatch()
			return err
		}
	}

	return nil
}

func Abort(proposalId string) error {
	err := Api.Abort(proposalId)
	return err
}
