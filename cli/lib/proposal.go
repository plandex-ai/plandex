package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex/types"
	"time"

	"github.com/fatih/color"
	"github.com/looplab/fsm"
	"github.com/plandex/plandex/shared"

	"github.com/briandowns/spinner"
)

type key struct {
	value rune
}

func Propose(prompt string) error {
	var err error

	start := time.Now()

	s := spinner.New(spinner.CharSets[33], 100*time.Millisecond)
	s.Prefix = "💬 Sending prompt..."
	s.Start()

	if CurrentPlanIsDraft() {
		fileNameResp, err := Api.FileName(prompt)
		if err != nil {
			fmt.Fprintln(os.Stderr, "\nError summarizing prompt:", err)
			return err
		}
		RenameCurrentDraftPlan(GetFileNameWithoutExt(fileNameResp.FileName))
	}

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
	finishedByPath := make(map[string]bool)

	jsonBuffers := make(map[string]string)
	numStreamedTokensByPath := make(map[string]int)

	replyTokenCounter := shared.NewReplyInfo()
	// var tokensAddedByFile map[string]int

	// currentPlanTokensByFilePath, err := loadCurrentPlanTokensByFilePath()
	// if err != nil {
	// 	return fmt.Errorf("failed to load token counts: %s\n", err)
	// }

	var parentProposalId string
	var planState types.PlanState
	// get plan state from [CurrentPlanRootDir]/plan.json
	planStatePath := filepath.Join(CurrentPlanRootDir, "plan.json")
	if _, err := os.Stat(planStatePath); os.IsNotExist(err) {
		planState = types.PlanState{}
	} else {
		fileBytes, err := os.ReadFile(planStatePath)
		if err != nil {
			return fmt.Errorf("failed to open plan state file: %s\n", err)
		}
		err = json.Unmarshal(fileBytes, &planState)
		if err != nil {
			return fmt.Errorf("failed to parse plan state json: %s\n", err)
		}
		parentProposalId = planState.ProposalId
	}

	var promptNumTokens int
	go func() {
		promptNumTokens = shared.GetNumTokens(prompt)
	}()

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
			if replyStarted && terminalHasPendingUpdate {
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
		printReply()
		backToMain()
		fmt.Print(termState)
		s = spinner.New(spinner.CharSets[33], 100*time.Millisecond)
		s.Prefix = "  "
		s.Start()
		var totalTokens int
		// _, tokensAddedByFile, totalTokens = replyTokenCounter.FinishAndRead()
		_, _, totalTokens = replyTokenCounter.FinishAndRead()
		err := appendConversation(types.AppendConversationParams{
			Timestamp:    timestamp,
			Prompt:       prompt,
			PromptTokens: promptNumTokens,
			Reply:        reply,
			ReplyTokens:  totalTokens,
		})
		if err != nil {
			fmt.Printf("failed to append conversation: %s\n", err)
		}
		endedReply = true

	}

	showUpdatedPlanCmds := func() {
		fmt.Println()
		for _, cmd := range []string{"diffs", "preview", "apply"} {
			clearCurrentLine()
			PrintCmds("  ", cmd)
		}
		clearCurrentLine()
		PrintCustomCmd("  ", "tell", "t", "update the plan, give more info, or chat")
	}

	contextByFilePath := make(map[string]shared.ModelContextPart)

	running := false
	queue := make(chan types.OnStreamPlanParams, 1)

	var apiReq *shared.PromptRequest

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

				// Save proposal id to [CurrentPlanRootDir]/plan.json
				planState = types.PlanState{
					ProposalId: proposalId,
				}
				planStatePath := filepath.Join(CurrentPlanRootDir, "plan.json")
				planStateBytes, err := json.Marshal(planState)
				if err != nil {
					onError(fmt.Errorf("failed to marshal plan state: %s\n", err))
					return
				}
				err = os.WriteFile(planStatePath, planStateBytes, 0644)
				if err != nil {
					onError(fmt.Errorf("failed to write plan state: %s\n", err))
					return
				}

				return
			}
		} else if !replyStarted {
			elapsed := time.Since(start)
			if elapsed < 700*time.Millisecond {
				time.Sleep(700*time.Millisecond - elapsed)
			}

			s.Stop()
			clearCurrentLine()
			alternateScreen()

			replyStarted = true
		}

		switch state.Current() {
		case shared.STATE_REPLYING, shared.STATE_REVISING:
			reply += content
			replyTokenCounter.AddToken(content, true)
			terminalHasPendingUpdate = true

		case shared.STATE_FINISHED:
			if !endedReply {
				endReply()
			}
			s.Stop()
			streamFinished = true

			if filesFinished {
				if desc.MadePlan && len(desc.Files) > 0 {
					showUpdatedPlanCmds()
				}
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

				if desc.MadePlan && (len(desc.Files) > 0) {
					s.Stop()
					fmt.Println("  " + color.New(color.BgGreen, color.FgHiWhite, color.Bold).Sprint(" 🏗  ") + color.New(color.BgGreen, color.FgHiWhite).Sprint("Building plan "))
					for _, filePath := range desc.Files {
						fmt.Printf("  📄 %s\n", filePath)
					}
					fmt.Println()
					fmt.Printf(displayHotkeys() + "\n")
				} else {
					filesFinished = true
				}

			}

		case shared.STATE_BUILDING:
			if content == shared.STREAM_BUILD_PHASE {
				// plan build mode started

			} else {
				wroteFile, err := receiveFileToken(&receiveFileChunkParams{
					Content:                 content,
					JsonBuffers:             jsonBuffers,
					NumStreamedTokensByPath: numStreamedTokensByPath,
					FinishedByPath:          finishedByPath,
				})

				if err != nil {
					onError(err)
					return
				}

				files := desc.Files

				// Clear previous lines
				if filesFinished {
					moveUpLines(len(files))
				} else {
					moveUpLines(len(files) + 4)
				}

				for _, filePath := range files {
					numStreamedTokens := numStreamedTokensByPath[filePath]

					fmtStr := "  📄 %s | %d 🪙"
					fmtArgs := []interface{}{filePath, numStreamedTokens}

					_, finished := finishedByPath[filePath]

					if finished {
						fmtStr += " | done ✅"
					}

					clearCurrentLine()

					fmt.Printf(fmtStr+"\n", fmtArgs...)
				}

				if wroteFile {
					// fmt.Printf("Wrote %d / %d files", len(finishedByPath), len(files))
					if len(finishedByPath) == len(files) {
						filesFinished = true

						if streamFinished {
							showUpdatedPlanCmds()
							close(done)
						}
					}
				}

				if !filesFinished {
					fmt.Println()
					fmt.Printf(displayHotkeys() + "\n")
				}

			}
		}

	}

	apiReq, err = Api.Propose(prompt, parentProposalId, handleStream)
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
