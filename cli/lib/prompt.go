package lib

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex/types"
	"regexp"
	"strconv"
	"time"

	"github.com/looplab/fsm"
	"github.com/plandex/plandex/shared"

	"github.com/briandowns/spinner"
)

type key struct {
	value rune
}

type openFileState struct {
	file         *os.File
	buffer       string
	jsonBuffer   string
	wrotePrefix  bool
	setExecPerms bool
	numTokens    int
	isNew        bool
	finished     bool
}

func Propose(prompt string) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond) // Choose spinner style
	s.Start()

	timestamp := StringTs()
	reply := ""
	done := make(chan struct{})

	termState := ""

	updateTimer := time.NewTimer(100 * time.Millisecond)
	defer updateTimer.Stop()

	var proposalId string
	var terminalHasPendingUpdate bool
	var endedReply bool

	var descJson string
	var desc *shared.PlanDescription
	var state *fsm.FSM

	fileStates := make(map[string]*openFileState)

	defer clearFileStates(fileStates)

	go func() {
		for range updateTimer.C {
			if terminalHasPendingUpdate {
				// Clear screen
				fmt.Print("\x1b[2J")
				// Move cursor to top-left
				fmt.Print("\x1b[H")
				mdFull, _ := GetMarkdown(reply)
				fmt.Println(mdFull)
				fmt.Printf(displayHotkeys())
				termState = mdFull
				terminalHasPendingUpdate = false
			}
			updateTimer.Reset(100 * time.Millisecond)
		}
	}()

	keyChan := make(chan *key, 1)
	ctx, cancelKeywatch := context.WithCancel(context.Background())
	errChn := make(chan error, 1)

	endReply := func() {
		time.Sleep(100 * time.Millisecond)
		backToMain()
		fmt.Print(termState)
		err := appendConversation(timestamp, prompt, reply)
		if err != nil {
			fmt.Printf("failed to append conversation: %s\n", err)
		}
		endedReply = true
	}

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
				s.Stop()
				// Switch to alternate screen and hide the cursor
				fmt.Print("\x1b[?1049h\x1b[?25l")

				return
			}
		}

		switch state.Current() {
		case shared.STATE_REPLYING, shared.STATE_REVISING:
			reply += content
			terminalHasPendingUpdate = true

		case shared.STATE_FINISHED:
			if !endedReply {
				endReply()
			}

			fmt.Println("Done")

			close(done)
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

				if desc.MadePlan && (len(desc.Files) > 0 || desc.HasExec) {
					fmt.Println("Writing files:")
					for _, filePath := range desc.Files {
						fmt.Println(filePath)
					}
					if desc.HasExec {
						fmt.Println("exec.sh")
					}
				}
			}

		case shared.STATE_BUILDING:
			if content == shared.STREAM_BUILD_PHASE {
				// plan build mode started

			} else {
				var chunk shared.PlanChunk
				err := json.Unmarshal([]byte(content), &chunk)
				if err != nil {
					onError(fmt.Errorf("error parsing plan chunk: %v", err))
					return
				}

				err = writeChunkToFile(fileStates, chunk)
				if err != nil {
					onError(fmt.Errorf("error writing plan chunk to file. chunk: %v, err: %v", chunk, err))
					return
				}
			}
		}

	}

	err := Api.Propose(prompt, handleStream)
	if err != nil {
		backToMain()
		return fmt.Errorf("failed to send prompt to server: %s\n", err)
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

// Function for 'a' key action
func handleAbortKey(proposalId string) error {
	return Abort(proposalId)
}

// Function for 'r' key action
func handleReviseKey(proposalId string) error {
	// Terminate current operation
	err := Api.Abort(proposalId)
	if err != nil {
		return err
	}

	// Prompt the user for new message
	fmt.Println(">\"")
	reader := bufio.NewReader(os.Stdin)
	newMessage, _ := reader.ReadString('"')

	// Propose the new message
	err = Propose(newMessage)
	if err != nil {
		return err
	}

	fmt.Println("Revision proposed.")
	return nil
}

func handleKeyPress(input rune, proposalId string) error {
	switch input {
	case 'a':
		return handleAbortKey(proposalId)
	case 'r':
		return handleReviseKey(proposalId)
	default:
		return fmt.Errorf("invalid key pressed: %s", string(input))
	}
}

func displayHotkeys() string {
	return `                                                                  
            ────────────────────────                                          
            ` + "\x1b[1m(a)\x1b[0m" + `bort   ` + "\x1b[1m(r)\x1b[0m" + `evise
            ────────────────────────                                          
            `
}

func backToMain() {
	// Switch back to main screen and show the cursor on exit
	fmt.Print("\x1b[?1049l\x1b[?25h")
}

func gitCommit(commitMsg string) error {

	err := GitAddAndCommit(ConversationSubdir, commitMsg)
	if err != nil {
		return fmt.Errorf("failed to commit files to conversation dir: %s\n", err)
	}

	err = GitAddAndCommit(PlanSubdir, commitMsg)
	if err != nil {
		return fmt.Errorf("failed to commit files to plan dir: %s\n", err)
	}

	// Stage changes in the submodules in the root repo
	err = GitAdd(CurrentPlanRootDir, ConversationSubdir, true)
	if err != nil {
		return fmt.Errorf("failed to stage submodule changes in conversation dir: %s\n", err)
	}

	err = GitAdd(CurrentPlanRootDir, PlanSubdir, true)
	if err != nil {
		return fmt.Errorf("failed to stage submodule changes in plan dir: %s\n", err)
	}

	// Commit these staged submodule changes in the root repo
	err = GitCommit(CurrentPlanRootDir, commitMsg, true)
	if err != nil {
		return fmt.Errorf("failed to commit submodule updates in root dir: %s\n", err)
	}

	return nil
}

func appendConversation(timestamp, prompt, reply string) error {
	// Create or append to conversation file
	responseTimestamp := StringTs()
	conversationFilePath := filepath.Join(ConversationSubdir, fmt.Sprintf("%s.md", timestamp))
	userHeader := fmt.Sprintf("@@@!>user|%s\n\n", timestamp)
	responseHeader := fmt.Sprintf("@@@!>response|%s\n\n", responseTimestamp)

	// TODO: store both summary and full response in conversation file for different use cases/context needs
	conversationFileContents := fmt.Sprintf("%s%s\n\n%s%s", userHeader, prompt, responseHeader, reply)
	conversationFile, err := os.OpenFile(conversationFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to write conversation file: %s\n", err)
	}
	defer conversationFile.Close()

	_, err = conversationFile.WriteString(conversationFileContents)
	if err != nil {
		return fmt.Errorf("failed to write conversation file: %s\n", err)
	}

	return nil
}

var jsonStartRegex = regexp.MustCompile(`\{\s*?"content"\s*?:\s*"`)

func writeChunkToFile(
	fileStates map[string]*openFileState,
	chunk shared.PlanChunk) error {
	var filePath string
	if chunk.IsExec {
		filePath = filepath.Join(PlanSubdir, "exec.sh")
	} else {
		filePath = filepath.Join(PlanFilesDir, chunk.FilePath)
	}

	state, ok := fileStates[filePath]
	if !ok {
		state = &openFileState{isNew: true}
		fileStates[filePath] = state
	}

	if state.finished {
		return nil
	}

	err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory: %s\n", err)
	}

	file := state.file
	if file == nil {
		var openFlags int
		if state.isNew {
			openFlags = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
		} else {
			openFlags = os.O_APPEND | os.O_CREATE | os.O_WRONLY
		}

		file, err = os.OpenFile(filePath, openFlags,
			0644)
		if err != nil {
			return fmt.Errorf("failed to open plan file '%s': %v\n", filePath,
				err)
		}

		// don't cache the file handle on first chunk because we need to first clear the file
		// the second chunk will instantiate a new appending file handle and cache it
		// subsequent chunks will used the cached appending file handle
		if state.isNew {
			state.isNew = false
			defer file.Close()
		} else {
			state.file = file
		}
	}

	state.jsonBuffer += chunk.Content

	if state.wrotePrefix {
		var streamed shared.StreamedFile
		err = json.Unmarshal([]byte(state.jsonBuffer), &streamed)

		if err == nil {
			// get diff between full content and current buffer
			diff := streamed.Content[len(state.buffer):]
			cleaned := cleanJsonString(diff)

			state.finished = true

			_, err = file.WriteString(cleaned)
			if err != nil {
				return fmt.Errorf("failed to write to plan file '%s': %v\n", filePath, err)
			}
		} else {
			cleaned := cleanJsonString(chunk.Content)

			state.buffer += cleaned

			_, err := file.WriteString(cleaned)
			if err != nil {
				return fmt.Errorf("failed to write to plan file '%s': %v\n", filePath, err)
			}
		}

	} else {
		// Check if buffer matches start of JSON syntax
		if jsonStartRegex.MatchString(state.jsonBuffer) {
			// replace the json start regex in buffer
			s := jsonStartRegex.ReplaceAllString(state.jsonBuffer, "")
			cleaned := cleanJsonString(s)

			_, err := file.WriteString(cleaned)
			if err != nil {
				return fmt.Errorf("failed to write to plan file '%s': %v\n", filePath, err)
			}

			state.buffer = cleaned
			state.wrotePrefix = true
		}
	}

	if chunk.IsExec && !state.setExecPerms {
		err = os.Chmod(filePath, 0755)
		if err != nil {
			return fmt.Errorf("failed to make exec file executable: %s\n", err)
		}
		state.setExecPerms = true
	}

	state.numTokens += 1
	fileStates[filePath] = state

	return nil
}

func cleanJsonString(s string) string {
	// replace escaped quotes with unescaped quotes
	s = regexp.MustCompile(`\\\"`).ReplaceAllString(s, `"`)

	// replace escaped backslashes with unescaped backslashes
	s = regexp.MustCompile(`\\\\`).ReplaceAllString(s, `\`)

	// replace escaped newlines with unescaped newlines
	s = regexp.MustCompile(`\\n`).ReplaceAllString(s, "\n")

	// replace escaped tabs with unescaped tabs
	s = regexp.MustCompile(`\\t`).ReplaceAllString(s, "\t")

	// replace escaped carriage returns with unescaped carriage returns
	s = regexp.MustCompile(`\\r`).ReplaceAllString(s, "\r")

	// replace escaped backspaces with unescaped backspaces
	s = regexp.MustCompile(`\\b`).ReplaceAllString(s, "\b")

	// replace escaped form feeds with unescaped form feeds
	s = regexp.MustCompile(`\\f`).ReplaceAllString(s, "\f")

	// replace escaped forward slashes with unescaped forward slashes
	s = regexp.MustCompile(`\\\/`).ReplaceAllString(s, "/")

	// replace escaped unicode characters with unescaped unicode characters
	s = regexp.MustCompile(`\\u([0-9a-fA-F]{4})`).ReplaceAllStringFunc(s, func(match string) string {
		unicode, _ := strconv.ParseInt(match[2:], 16, 32)
		return string(rune(unicode))
	})

	return s
}

func clearFileStates(fileStates map[string]*openFileState) {
	for path, state := range fileStates {
		state.file.Close()
		fmt.Println("closed file", path)
		delete(fileStates, path)
	}
}
