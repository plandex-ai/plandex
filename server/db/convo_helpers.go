package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

func GetPlanConvo(orgId, planId string) ([]*ConvoMessage, error) {
	var convo []*ConvoMessage
	convoDir := getPlanConversationDir(orgId, planId)

	files, err := os.ReadDir(convoDir)
	if err != nil {
		return nil, fmt.Errorf("error reading convo dir: %v", err)
	}

	errCh := make(chan error, len(files))
	convoCh := make(chan *ConvoMessage, len(files))

	for _, file := range files {
		go func(file os.DirEntry) {
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

func StoreConvoMessage(message *ConvoMessage, commit bool) (string, error) {
	convoDir := getPlanConversationDir(message.OrgId, message.PlanId)

	id := uuid.New().String()
	ts := time.Now().UTC()

	message.Id = id
	message.CreatedAt = ts

	bytes, err := json.Marshal(message)

	if err != nil {
		return "", fmt.Errorf("error marshalling convo message: %v", err)
	}

	err = os.WriteFile(filepath.Join(convoDir, message.Id+".json"), bytes, os.ModePerm)

	if err != nil {
		return "", fmt.Errorf("error writing convo message: %v", err)
	}

	err = AddPlanConvoMessage(message.PlanId, message.Tokens)

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
			desc += " | 🛑 stopped early"
		}
	}

	msg := fmt.Sprintf("Message #%d | %s | %d 🪙", message.Num, desc, message.Tokens)

	if commit {
		err = GitAddAndCommit(message.OrgId, message.PlanId, msg)
		if err != nil {
			return "", fmt.Errorf("error committing convo message: %v", err)
		}
	}

	return msg, nil
}

func GetPlanSummaries(planId string) ([]*ConvoSummary, error) {
	var summaries []*ConvoSummary

	err := Conn.Select(&summaries, "SELECT * FROM convo_summaries WHERE plan_id = $1 ORDER BY created_at", planId)

	if err != nil {
		return nil, fmt.Errorf("error getting plan summaries: %v", err)
	}
	return summaries, nil
}

func StoreSummary(summary *ConvoSummary) error {
	query := "INSERT INTO convo_summaries (org_id, plan_id, latest_convo_message_id, latest_convo_message_created_at, summary, tokens, num_messages, created_at) VALUES (:org_id, :plan_id, :latest_convo_message_id, :latest_convo_message_created_at, :summary, :tokens, :num_messages, :created_at) RETURNING id, created_at"

	row, err := Conn.NamedQuery(query, summary)

	if err != nil {
		return fmt.Errorf("error storing summary: %v", err)
	}

	defer row.Close()

	if row.Next() {
		var createdAt time.Time
		var id string
		if err := row.Scan(&id, &createdAt); err != nil {
			return fmt.Errorf("error storing summary: %v", err)
		}

		summary.Id = id
		summary.CreatedAt = createdAt
	}

	return nil
}
