package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex/format"
	"plandex/term"
	"plandex/types"
	"strconv"
	"strings"
	"sync"
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
			fmt.Fprintf(os.Stderr, "Failed to check outdated context: %v\n", err)
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

	// fmt.Println("Checked current plan is draft")

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
	var buildMu sync.Mutex

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

	// fmt.Println("Initialized locals")

	promptNumTokens, err := shared.GetNumTokens(prompt)
	if err != nil {
		return fmt.Errorf("failed to get number of tokens in prompt: %s", err)
	}

	// fmt.Println("Got prompt num tokens")

	err = appendConversation(types.AppendConversationParams{
		Timestamp: timestamp,
		PlanState: planState,
		PromptParams: &types.AppendConversationPromptParams{
			Prompt:       prompt,
			PromptTokens: promptNumTokens,
		},
	})

	// fmt.Println("Appended conversation")

	if err != nil {
		return fmt.Errorf("failed to append prompt to conversation: %s", err)
	}

	printReply := func() {
		term.ClearScreen()
		term.MoveCursorToTopLeft()
		mdFull, _ := term.GetMarkdown(reply)
		fmt.Println(mdFull)
		fmt.Println(displayHotkeys())
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

	// fmt.Println("Starting proposal")

	keyChan := make(chan *key, 1)
	ctx, cancelKeywatch := context.WithCancel(context.Background())
	errChn := make(chan error, 1)

	endReply := func() {
		replyUpdateTimer.Stop()
		printReply()
		term.BackToMain()
		fmt.Print(termState)
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

		if len(files) == 0 {
			return
		}

		// Clear previous lines
		numLines := len(files) + 4

		term.MoveUpLines(numLines)

		for _, filePath := range files {
			numStreamedTokens := numStreamedTokensByPath[filePath]

			fmtStr := "  📄 %s | %d 🪙"
			fmtArgs := []interface{}{filePath, numStreamedTokens}

			_, finished := finishedByPath[filePath]

			if finished {
				fmtStr += " | done ✅"
			}

			term.ClearCurrentLine()

			fmt.Printf(fmtStr+"\n", fmtArgs...)
		}

		if !(filesFinished && streamFinished) {
			fmt.Println()
			fmt.Printf(displayHotkeys() + "\n")
		}
	}

	contextByFilePath := make(map[string]*shared.ModelContextPart)

	running := false
	queue := make(chan types.OnStreamPlanParams, 1)

	var apiReq *shared.PromptRequest

	var handleStream types.OnStreamPlan

	// fmt.Println("Starting handleStream")

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
			term.BackToMain()
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
			term.ClearCurrentLine()
			term.AlternateScreen()

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
			buildMu.Lock()
			defer buildMu.Unlock()
			streamFinished = true
			if filesFinished && !closedDone {
				close(done)
				closedDone = true
				writeFileProgress()
			}

		case shared.STATE_DESCRIBING:
			if content == shared.STREAM_DESCRIPTION_PHASE {
				return
			}

			describeStart := time.Now()

			bytes := []byte(content)

			err := json.Unmarshal(bytes, &desc)
			if err != nil {
				onError(fmt.Errorf("error parsing plan description: %v", err))
				return
			}

			if len(desc.Files) > 0 {
				s = spinner.New(spinner.CharSets[33], 100*time.Millisecond)
				s.Prefix = "  "
				s.Start()
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

			// fmt.Println("appending convo reply")

			err = appendConvoReply()
			if err != nil {
				onError(fmt.Errorf("failed to append reply to conversation: %s", err))
				return
			}

			// fmt.Println("appended convo reply")

			// wait a bit if necessary to avoid jarring output of conversation as soon as stream finishes
			elapsed := time.Since(describeStart)
			if elapsed < 1500*time.Millisecond {
				time.Sleep(1500*time.Millisecond - elapsed)
			}

			// fmt.Println("waited elapsed")

			endReply()

			// fmt.Println("ended reply")
			s.Stop()

			if desc.MadePlan && (len(desc.Files) > 0) {
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
				buildMu.Lock()
				defer buildMu.Unlock()
				filesFinished = true

				err = GitCommitPlanUpdate(desc.CommitMsg)
				if err != nil {
					onError(fmt.Errorf("failed to commit root update: %s", err))
					return
				}

				if streamFinished && !closedDone {
					close(done)
					closedDone = true
					writeFileProgress()
				}
			}
		}

	}

	apiReq, err = Api.Propose(prompt, parentProposalId, rootId, handleStream)
	if err != nil {
		term.BackToMain()
		return fmt.Errorf("failed to send prompt to server: %s", err)
	}

	// fmt.Println("Sent prompt to server")

	for _, part := range apiReq.ModelContext {
		contextByFilePath[part.FilePath] = part
	}

	// fmt.Println("Got context")

	go func(ctx context.Context, errChn chan error) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				k, err := term.GetUserInput()
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

	// fmt.Println("Started key loop")

	if desc != nil {
		if desc.MadePlan && len(desc.Files) > 0 {
			fmt.Println()
			for _, cmd := range []string{"apply", "diffs", "preview"} {
				term.ClearCurrentLine()
				term.PrintCmds("  ", cmd)
			}
		}

		term.ClearCurrentLine()
		term.PrintCustomCmd("  ", "tell", "t", "update the plan, give more info, or chat")

		term.ClearCurrentLine()
		term.PrintCmds("  ", "continue")

		term.ClearCurrentLine()
		term.PrintCmds("  ", "rewind")
	}

	// fmt.Println("Finished proposal")

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
		term.ClearCurrentLine()
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

	confirmed, canceled, err := term.ConfirmYesNoCancel("Update context now?")

	if err != nil {
		return false, fmt.Errorf("failed to get user input: %s", err)
	}

	if confirmed {
		MustUpdateContextWithOuput()
	}

	shouldContinue := !canceled

	return shouldContinue, nil
}

func handleAbortKey(proposalId string) error {
	return Abort(proposalId)
}

func handleKeyPress(input rune, proposalId string) error {
	switch input {
	case 's':
		return handleAbortKey(proposalId)
	default:
		return fmt.Errorf("invalid key pressed: %s", string(input))
	}
}

func displayHotkeys() string {
	divisionLine := term.GetDivisionLine()

	return divisionLine + "\n" +
		"  \x1b[1m(s)\x1b[0m" + `top  
` + divisionLine
}
