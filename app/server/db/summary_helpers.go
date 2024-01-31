package db

import (
	"fmt"
	"time"

	"github.com/lib/pq"
)

func GetPlanSummaries(planId string, convoMessageIds []string) ([]*ConvoSummary, error) {
	var summaries []*ConvoSummary

	err := Conn.Select(&summaries, "SELECT * FROM convo_summaries WHERE plan_id = $1 AND latest_convo_message_id = ANY($2) ORDER BY created_at", planId, pq.Array(convoMessageIds))

	if err != nil {
		return nil, fmt.Errorf("error getting plan summaries: %v", err)
	}
	return summaries, nil
}

func StoreSummary(summary *ConvoSummary) error {
	query := "INSERT INTO convo_summaries (org_id, plan_id, latest_convo_message_id, latest_convo_message_created_at, summary, tokens, num_messages) VALUES (:org_id, :plan_id, :latest_convo_message_id, :latest_convo_message_created_at, :summary, :tokens, :num_messages) RETURNING id, created_at"

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
