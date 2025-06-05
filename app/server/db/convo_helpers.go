package db

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	shared "plandex-shared"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

func GetPlanConvo(orgId, planId string) ([]*ConvoMessage, error) {
	var convo []*ConvoMessage
	convoDir := getPlanConversationDir(orgId, planId)

	files, err := os.ReadDir(convoDir)
	if err != nil {
		if os.IsNotExist(err) {
			return convo, nil
		}

		return nil, fmt.Errorf("error reading convo dir: %v", err)
	}

	errCh := make(chan error, len(files))
	convoCh := make(chan *ConvoMessage, len(files))

	for _, file := range files {
		go func(file os.DirEntry) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in GetPlanConvo: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in GetPlanConvo: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
			bytes, err := os.ReadFile(filepath.Join(convoDir, file.Name()))

			if err != nil {
				errCh <- fmt.Errorf("error reading convo file: %v", err)
				return
			}

			var convoMessage ConvoMessage
			err = json.Unmarshal(bytes, &convoMessage)

			if err != nil {
				errCh <- fmt.Errorf("error unmarshalling convo file: %v", err)
				return
			}

			convoCh <- &convoMessage

		}(file)
	}

	for i := 0; i < len(files); i++ {
		select {
		case err := <-errCh:
			return nil, fmt.Errorf("error reading convo files: %v", err)
		case convoMessage := <-convoCh:
			convo = append(convo, convoMessage)
		}
	}

	sort.Slice(convo, func(i, j int) bool {
		return convo[i].CreatedAt.Before(convo[j].CreatedAt)
	})

	return convo, nil
}

func GetConvoMessage(orgId, planId, messageId string) (*ConvoMessage, error) {
	convoDir := getPlanConversationDir(orgId, planId)

	filePath := filepath.Join(convoDir, messageId+".json")

	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading convo message: %v", err)
	}

	var convoMessage ConvoMessage
	err = json.Unmarshal(bytes, &convoMessage)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling convo message: %v", err)
	}

	return &convoMessage, nil
}

func StoreConvoMessage(repo *GitRepo, message *ConvoMessage, currentUserId, branch string, commit bool) (string, error) {
	convoDir := getPlanConversationDir(message.OrgId, message.PlanId)

	ts := time.Now().UTC()

	if message.Id == "" {
		message.Id = uuid.New().String()
	}

	message.CreatedAt = ts

	bytes, err := json.Marshal(message)

	if err != nil {
		return "", fmt.Errorf("error marshalling convo message: %v", err)
	}

	err = os.MkdirAll(convoDir, os.ModePerm)

	if err != nil {
		return "", fmt.Errorf("error creating convo dir: %v", err)
	}

	err = os.WriteFile(filepath.Join(convoDir, message.Id+".json"), bytes, os.ModePerm)

	if err != nil {
		return "", fmt.Errorf("error writing convo message: %v", err)
	}

	err = AddPlanConvoMessage(message, branch)

	if err != nil {
		return "", fmt.Errorf("error adding convo tokens: %v", err)
	}

	var desc string
	if message.Role == openai.ChatMessageRoleUser {
		desc = "💬 User prompt"
		// TODO: add user name
	} else {
		desc = "🤖 Plandex reply"
		if message.Stopped {
			desc += " | 🛑 " + color.New(color.FgHiRed).Sprint("stopped")
		}
	}

	replyTags := message.Flags.GetReplyTags()

	var msg string
	if len(replyTags) > 0 {
		msg = fmt.Sprintf("Message #%d | %s | %s | %d 🪙", message.Num, desc, strings.Join(replyTags, " | "), message.Tokens)
	} else {
		msg = fmt.Sprintf("Message #%d | %s | %d 🪙", message.Num, desc, message.Tokens)
	}

	if len(message.AddedSubtasks) > 0 {
		msg += "\n\n"
		for _, subtask := range message.AddedSubtasks {
			msg += "\n• " + subtask.Title
		}
	}

	if len(message.RemovedSubtasks) > 0 {
		msg += "\n\n"
		msg += "Removed Tasks"
		for _, subtask := range message.RemovedSubtasks {
			msg += "\n• " + subtask
		}
	}

	log.Println("StoreConvoMessage - message.Flags.CurrentStage.TellStage:", message.Flags.CurrentStage.TellStage)
	log.Println("StoreConvoMessage - message.Subtask:", message.Subtask)

	if message.Flags.CurrentStage.TellStage == shared.TellStageImplementation && message.Subtask != nil {
		msg += "\n\n" + "📋 " + message.Subtask.Title
		if len(message.Subtask.UsesFiles) > 0 {
			for _, file := range message.Subtask.UsesFiles {
				msg += "\n • 📄 " + file
			}
		}
	}

	if message.Flags.DidCompletePlan {
		msg += "\n\n" + "🏁 Completed Plan"
	}

	// Cleaner without the cut off message - maybe need a separate command to show both the log and full messages?
	// cutoff := 140
	// if len(message.Message) > cutoff {
	// 	msg += "\n\n" + message.Message[:cutoff] + "..."
	// } else {
	// 	msg += "\n\n" + message.Message
	// }

	if commit {
		log.Printf("[Git] StoreConvoMessage - committing convo message: %s, branch: %s", msg, branch)
		err = repo.GitAddAndCommit(branch, msg)
		if err != nil {
			return "", fmt.Errorf("error committing convo message: %v", err)
		}
	}

	return msg, nil
}
