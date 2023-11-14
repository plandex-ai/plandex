package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex/format"
	"plandex/types"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/looplab/fsm"
	"github.com/plandex/plandex/shared"

	"github.com/briandowns/spinner"
)

const replyStreamThrottle = 70 * time.Millisecond

type key struct {
	value rune
}

func Propose(prompt string) error {
	var err error

	planState, err := GetPlanState()
	if err != nil {
		return fmt.Errorf("failed to get plan state: %s", err)
	}

	s := spinner.New(spinner.CharSets[33], 100*time.Millisecond)
	if planState.ContextUpdatableTokens > 0 {
		shouldContinue, err := checkOutdatedContext(s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to check outdated context: %v", err)
			return err
		}

		if !shouldContinue {
			return nil
		}
	}

	start := time.Now()
	s.Prefix = "💬 Sending prompt... "
	s.Start()

	if CurrentPlanIsDraft() {
		fileNameResp, err := Api.FileName(prompt)
		if err != nil {
			fmt.Fprintln(os.Stderr, "\nError getting file name for prompt:", err)
			return err
		}
		err = RenameCurrentDraftPlan(planState, format.GetFileNameWithoutExt(fileNameResp.FileName))
		if err != nil {
			fmt.Fprintln(os.Stderr, "\nError renaming draft plan:", err)
			return err
		}
	}

	timestamp := shared.StringTs()
	reply := ""
	done := make(chan struct{})
	closedDone := false

	termState := ""

	replyUpdateTimer := time.NewTimer(150 * time.Millisecond)
	defer replyUpdateTimer.Stop()

	var proposalId string
	var replyStarted bool
	var terminalHasPendingUpdate bool
	var desc *shared.PlanDescription
	var state *fsm.FSM
	var streamFinished bool
	var filesFinished bool
	var lastReplyTokenAdded time.Time
	finishedByPath := make(map[string]bool)

	numStreamedTokensByPath := make(map[string]int)

	replyTokenCounter := shared.NewReplyInfo()

	var parentProposalId string
	var rootId string

	parentProposalId = planState.ProposalId
	rootId = planState.RootId

	if rootId != "" {
		err = saveLatestConvoSummary(rootId)
		if err != nil {
			return fmt.Errorf("failed to save latest convo summary: %s", err)
		}
	}

	promptNumTokens := shared.GetNumTokens(prompt)

	err = appendConversation(types.AppendConversationParams{
		Timestamp: timestamp,
		PlanState: planState,
		PromptParams: &types.AppendConversationPromptParams{
			Prompt:       prompt,
			PromptTokens: promptNumTokens,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to append prompt to conversation: %s", err)
	}

	printReply := func() {
		ClearScreen()
		MoveCursorToTopLeft()
		mdFull, _ := GetMarkdown(reply)
		fmt.Println(mdFull)
		fmt.Print(displayHotkeys())
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
		BackToMain()
		fmt.Print(termState)
		s = spinner.New(spinner.CharSets[33], 100*time.Millisecond)
		s.Prefix = "  "
		s.Start()
	}

	appendConvoReply := func() error {
		var totalTokens int
		_, _, _, totalTokens = replyTokenCounter.FinishAndRead()

		return appendConversation(types.AppendConversationParams{
			Timestamp: timestamp,
			PlanState: planState,
			ReplyParams: &types.AppendConversationReplyParams{
				ResponseTimestamp: desc.ResponseTimestamp,
				Reply:             reply,
				ReplyTokens:       totalTokens,
			},
		})
	}

	writeFileProgress := func() {
		files := desc.Files
		// Clear previous lines
		if filesFinished {
			MoveUpLines(len(files))
		} else {
			MoveUpLines(len(files) + 4)
		}

		for _, filePath := range files {
			numStreamedTokens := numStreamedTokensByPath[filePath]

			fmtStr := "  📄 %s | %d 🪙"
			fmtArgs := []interface{}{filePath, numStreamedTokens}

			_, finished := finishedByPath[filePath]

			if finished {
				fmtStr += " | done ✅"
			}

			ClearCurrentLine()

			fmt.Printf(fmtStr+"\n", fmtArgs...)
		}

		if !filesFinished {
			fmt.Println()
			fmt.Printf(displayHotkeys() + "\n")
		}
	}

	contextByFilePath := make(map[string]*shared.ModelContextPart)

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
			BackToMain()
			fmt.Fprintln(os.Stderr, "Error:", err)
			cancelKeywatch()
			if !closedDone {
				close(done)
				closedDone = true
			}
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
				if rootId == "" {
					rootId = proposalId
				}

				// Save proposal id to [CurrentPlanRootDir]/plan.json
				planState.ProposalId = proposalId
				planState.RootId = rootId
				err = SetPlanState(planState, shared.StringTs())
				if err != nil {
					onError(fmt.Errorf("failed to update plan state: %s", err))
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
			ClearCurrentLine()
			alternateScreen()

			replyStarted = true
			lastReplyTokenAdded = time.Now()
		}

		switch state.Current() {
		case shared.STATE_REPLYING:
			elapsed := time.Since(lastReplyTokenAdded)

			if elapsed < replyStreamThrottle {
				time.Sleep(replyStreamThrottle - elapsed)
			}

			reply += content
			replyTokenCounter.AddToken(content, true)
			terminalHasPendingUpdate = true
			lastReplyTokenAdded = time.Now()

		case shared.STATE_FINISHED:
			s.Stop()
			streamFinished = true

			if filesFinished {
				close(done)
				closedDone = true
			}

		case shared.STATE_DESCRIBING:
			if content == shared.STREAM_DESCRIPTION_PHASE {
				endReply()
				return
			}
			bytes := []byte(content)

			err := json.Unmarshal(bytes, &desc)
			if err != nil {
				onError(fmt.Errorf("error parsing plan description: %v", err))
				return
			}

			err = os.MkdirAll(DescriptionsSubdir, os.ModePerm)
			if err != nil {
				onError(fmt.Errorf("failed to create plan descriptions directory: %s", err))
				return
			}
			descriptionsPath := filepath.Join(DescriptionsSubdir, desc.ResponseTimestamp+".json")
			err = os.WriteFile(descriptionsPath, bytes, 0644)
			if err != nil {
				onError(fmt.Errorf("failed to write plan description to file: %s", err))
				return
			}

			err = appendConvoReply()
			if err != nil {
				onError(fmt.Errorf("failed to append reply to conversation: %s", err))
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

		case shared.STATE_BUILDING:
			if content == shared.STREAM_BUILD_PHASE {
				return

			}
			err := updateTokenCounts(content, numStreamedTokensByPath, finishedByPath)

			if err != nil {
				onError(err)
				return
			}

			writeFileProgress()

		case shared.STATE_WRITING:
			if content == shared.STREAM_WRITE_PHASE {
				// write phase started
				return
			}

			err := writePlanRes(content)

			if err != nil {
				onError(fmt.Errorf("failed to write plan file: %s", err))
				return
			}

			// fmt.Printf("Wrote %d / %d files", len(finishedByPath), len(files))
			if len(finishedByPath) == len(desc.Files) {
				filesFinished = true

				err = GitCommitPlanUpdate(desc.CommitMsg)
				if err != nil {
					onError(fmt.Errorf("failed to commit root update: %s", err))
					return
				}

				if streamFinished {
					close(done)
					closedDone = true
				}
			}
		}

	}

	apiReq, err = Api.Propose(prompt, parentProposalId, rootId, handleStream)
	if err != nil {
		BackToMain()
		return fmt.Errorf("failed to send prompt to server: %s", err)
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

	if desc != nil {
		if desc.MadePlan && len(desc.Files) > 0 {
			fmt.Println()
			for _, cmd := range []string{"apply", "diffs", "preview"} {
				ClearCurrentLine()
				PrintCmds("  ", cmd)
			}
		}

		ClearCurrentLine()
		PrintCustomCmd("  ", "tell", "t", "update the plan, give more info, or chat")

		ClearCurrentLine()
		PrintCmds("  ", "continue")

		ClearCurrentLine()
		PrintCmds("  ", "rewind")
	}

	return nil
}

func Abort(proposalId string) error {
	err := Api.Abort(proposalId)
	return err
}

func checkOutdatedContext(s *spinner.Spinner) (bool, error) {
	start := time.Now()
	s.Prefix = "🔬 Checking context... "
	s.Start()

	stopSpinner := func() {
		s.Stop()
		ClearCurrentLine()
	}

	outdatedRes, err := CheckOutdatedContext()
	if err != nil {
		stopSpinner()
		return false, fmt.Errorf("failed to check outdated context: %s", err)
	}

	elapsed := time.Since(start)
	if elapsed < 700*time.Millisecond {
		time.Sleep(700*time.Millisecond - elapsed)
	}

	stopSpinner()

	if len(outdatedRes.UpdatedParts) == 0 {
		fmt.Println("✅ Context is up to date")
		return true, nil
	}
	types := []string{}
	if outdatedRes.NumFiles > 0 {
		lbl := "file"
		if outdatedRes.NumFiles > 1 {
			lbl = "files"
		}
		lbl = strconv.Itoa(outdatedRes.NumFiles) + " " + lbl
		types = append(types, lbl)
	}
	if outdatedRes.NumUrls > 0 {
		lbl := "url"
		if outdatedRes.NumUrls > 1 {
			lbl = "urls"
		}
		lbl = strconv.Itoa(outdatedRes.NumUrls) + " " + lbl
		types = append(types, lbl)
	}
	if outdatedRes.NumTrees > 0 {
		lbl := "directory tree"
		if outdatedRes.NumTrees > 1 {
			lbl = "directory trees"
		}
		lbl = strconv.Itoa(outdatedRes.NumTrees) + " " + lbl
		types = append(types, lbl)
	}

	var msg string
	if len(types) <= 2 {
		msg += strings.Join(types, " and ")
	} else {
		for i, add := range types {
			if i == len(types)-1 {
				msg += ", and " + add
			} else {
				msg += ", " + add
			}
		}
	}

	phrase := "have been"
	if len(outdatedRes.UpdatedParts) == 1 {
		phrase = "has been"
	}
	color.New(color.FgHiCyan, color.Bold).Printf("%s in context %s modified 👇\n\n", msg, phrase)

	tableString := TableForContextUpdateRes(outdatedRes)
	fmt.Println(tableString)

	fmt.Println()

	var input func() (bool, error)
	input = func() (bool, error) {
		color.New(color.FgHiGreen, color.Bold).Println("Update context now? (y)es | (n)o | (c)ancel")
		color.New(color.FgHiGreen, color.Bold).Print("> ")

		char, err := getUserInput()
		if err != nil {
			return false, fmt.Errorf("failed to get user input: %s", err)
		}

		fmt.Println(string(char))
		if char == 'y' || char == 'Y' {
			MustUpdateContextWithOuput()
			return true, nil
		} else if char == 'n' || char == 'N' {
			return true, nil
		} else if char == 'c' || char == 'C' {
			return false, nil
		} else {
			fmt.Println()
			color.New(color.FgHiRed, color.Bold).Print("Invalid input.\nEnter 'y' for yes, 'n' for no, or 'c' to cancel.\n\n")
			return input()
		}
	}

	shouldContinue, err := input()
	if err != nil {
		return false, fmt.Errorf("failed to get user input: %s", err)
	}

	return shouldContinue, nil
}
