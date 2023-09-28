package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/briandowns/spinner"
)

func Prompt(prompt string, chatOnly bool) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond) // Choose spinner style
	defer s.Stop()
	s.Start()
	fmt.Fprintln(os.Stderr, "Sending prompt...")

	timestamp := StringTs()
	response, err := ApiPrompt(prompt, chatOnly)
	if err != nil {
		return fmt.Errorf("failed to send prompt to server: %s\n", err)
	}

	fmt.Fprintln(os.Stderr, response)

	// Create or append to conversation file
	responseTimestamp := StringTs()
	conversationFilePath := filepath.Join(ConversationSubdir, fmt.Sprintf("%s.md", timestamp))
	userHeader := fmt.Sprintf("@@@!>user|%s\n\n", timestamp)
	responseHeader := fmt.Sprintf("@@@!>response|%s\n\n", responseTimestamp)
	conversationFileContents := fmt.Sprintf("%s%s\n\n%s%s", userHeader, prompt, responseHeader, response.Reply)
	conversationFile, err := os.OpenFile(conversationFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to write conversation file: %s\n", err)
	}
	defer conversationFile.Close()

	_, err = conversationFile.WriteString(conversationFileContents)
	if err != nil {
		return fmt.Errorf("failed to write conversation file: %s\n", err)
	}

	if len(response.Files) > 0 {
		filesDir := filepath.Join(PlanSubdir, "files")
		err = os.MkdirAll(filesDir, os.ModePerm)

		if err != nil {
			return fmt.Errorf("failed to create files directory: %s\n", err)
		}

		for path, contents := range response.Files {
			filePath := filepath.Join(filesDir, path)
			dir := filepath.Dir(filePath)

			err = os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return fmt.Errorf("failed to create directory: %s\n", err)
			}

			err = os.WriteFile(filePath, []byte(contents), 0644)
			if err != nil {
				return fmt.Errorf("failed to write file: %s\n", err)
			}
		}
	}

	if response.Exec != "" {
		// write to exec.sh file
		execFilePath := filepath.Join(PlanSubdir, "exec.sh")
		execFile, err := os.OpenFile(execFilePath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to write exec file: %s\n", err)
		}
		defer execFile.Close()

		_, err = execFile.WriteString(response.Exec)
		if err != nil {
			return fmt.Errorf("failed to write exec file: %s\n", err)
		}

		err = os.Chmod(execFilePath, 0755)
		if err != nil {
			return fmt.Errorf("failed to make exec file executable: %s\n", err)
		}
	}

	err = GitAddAndCommit(ConversationSubdir, response.CommitMsg)
	if err != nil {
		return fmt.Errorf("failed to commit files to conversation dir: %s\n", err)
	}

	err = GitAddAndCommit(PlanSubdir, response.CommitMsg)
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
	err = GitCommit(CurrentPlanRootDir, response.CommitMsg, true)
	if err != nil {
		return fmt.Errorf("failed to commit submodule updates in root dir: %s\n", err)
	}

	return nil
}
