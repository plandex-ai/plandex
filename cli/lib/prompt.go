package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/briandowns/spinner"
)

func Propose(prompt string, chatOnly bool) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond) // Choose spinner style
	s.Start()
	fmt.Fprintln(os.Stderr, "Sending prompt...")

	timestamp := StringTs()
	reply := ""
	done := make(chan struct{})

	s.Stop()

	// Switch to alternate screen and hide the cursor
	fmt.Print("\x1b[?1049h\x1b[?25l")

	backToMain := func() {
		// Switch back to main screen and show the cursor on exit
		fmt.Print("\x1b[?1049l\x1b[?25h")
	}

	termState := ""

	updateTimer := time.NewTimer(100 * time.Millisecond)
	defer updateTimer.Stop()

	var pendingUpdate bool

	go func() {
		for range updateTimer.C {
			if pendingUpdate {
				// Clear screen
				fmt.Print("\x1b[2J")
				// Move cursor to top-left
				fmt.Print("\x1b[H")
				mdFull, _ := GetMarkdown(reply)
				fmt.Println(mdFull)
				termState = mdFull
				pendingUpdate = false
			}
			updateTimer.Reset(100 * time.Millisecond)
		}
	}()

	err := ApiPropose(prompt, chatOnly, func(content string, isFinished bool, err error) {
		onError := func(err error) {
			backToMain()
			fmt.Fprintln(os.Stderr, "Error:", err)
			close(done)
		}

		if err != nil {
			onError(err)
			return
		}

		if isFinished {
			close(done)
			return
		}

		reply += content
		pendingUpdate = true
	})
	if err != nil {
		backToMain()
		return fmt.Errorf("failed to send prompt to server: %s\n", err)
	}

	// block until streaming is done
	<-done

	backToMain()
	fmt.Print(termState)

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

func Confirm(proposalId string) error {

	// if len(response.Files) > 0 {
	// 	filesDir := filepath.Join(PlanSubdir, "files")
	// 	err = os.MkdirAll(filesDir, os.ModePerm)

	// 	if err != nil {
	// 		return fmt.Errorf("failed to create files directory: %s\n", err)
	// 	}

	// 	for path, contents := range response.Files {
	// 		filePath := filepath.Join(filesDir, path)
	// 		dir := filepath.Dir(filePath)

	// 		err = os.MkdirAll(dir, os.ModePerm)
	// 		if err != nil {
	// 			return fmt.Errorf("failed to create directory: %s\n", err)
	// 		}

	// 		err = os.WriteFile(filePath, []byte(contents), 0644)
	// 		if err != nil {
	// 			return fmt.Errorf("failed to write file: %s\n", err)
	// 		}
	// 	}
	// }

	// if response.Exec != "" {
	// 	// write to exec.sh file
	// 	execFilePath := filepath.Join(PlanSubdir, "exec.sh")
	// 	execFile, err := os.OpenFile(execFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to write exec file: %s\n", err)
	// 	}
	// 	defer execFile.Close()

	// 	_, err = execFile.WriteString(response.Exec)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to write exec file: %s\n", err)
	// 	}

	// 	err = os.Chmod(execFilePath, 0755)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to make exec file executable: %s\n", err)
	// 	}
	// }

	// err = GitAddAndCommit(ConversationSubdir, response.CommitMsg)
	// if err != nil {
	// 	return fmt.Errorf("failed to commit files to conversation dir: %s\n", err)
	// }

	// err = GitAddAndCommit(PlanSubdir, response.CommitMsg)
	// if err != nil {
	// 	return fmt.Errorf("failed to commit files to plan dir: %s\n", err)
	// }

	// // Stage changes in the submodules in the root repo
	// err = GitAdd(CurrentPlanRootDir, ConversationSubdir, true)
	// if err != nil {
	// 	return fmt.Errorf("failed to stage submodule changes in conversation dir: %s\n", err)
	// }

	// err = GitAdd(CurrentPlanRootDir, PlanSubdir, true)
	// if err != nil {
	// 	return fmt.Errorf("failed to stage submodule changes in plan dir: %s\n", err)
	// }

	// // Commit these staged submodule changes in the root repo
	// err = GitCommit(CurrentPlanRootDir, response.CommitMsg, true)
	// if err != nil {
	// 	return fmt.Errorf("failed to commit submodule updates in root dir: %s\n", err)
	// }

	return nil
}

func Abort(proposalId string) error {

	return nil
}
