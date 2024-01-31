package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

func StoreModelStream(stream *ModelStream) error {
	query := `INSERT INTO model_streams (org_id, plan_id, internal_ip, branch) VALUES (:org_id, :plan_id, :internal_ip, :branch) RETURNING id, created_at`

	row, err := Conn.NamedQuery(query, stream)

	if err != nil {
		return fmt.Errorf("error storing model stream: %v", err)
	}

	defer row.Close()

	if row.Next() {
		var createdAt time.Time
		var id string
		if err := row.Scan(&id, &createdAt); err != nil {
			return fmt.Errorf("error storing model stream: %v", err)
		}

		stream.Id = id
		stream.CreatedAt = createdAt
	}

	return nil
}

func SetModelStreamFinished(id string) error {
	log.Println("Setting model stream finished:", id)

	_, err := Conn.Exec("UPDATE model_streams SET finished_at = NOW() WHERE id = $1", id)

	if err != nil {
		return fmt.Errorf("error setting model stream finished: %v", err)
	}

	log.Println("Set model stream finished successfully:", id)

	return nil
}

func GetActiveModelStream(planId, branch string) (*ModelStream, error) {
	var stream ModelStream
	err := Conn.Get(&stream, "SELECT * FROM model_streams WHERE plan_id = $1 AND branch = $2 AND finished_at IS NULL", planId, branch)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("error getting active model stream: %v", err)
	}

	return &stream, nil
}

// func StoreModelStreamSubscription(subscription *ModelStreamSubscription) error {
// 	query := `INSERT INTO model_stream_subscriptions (model_stream_id, org_id, plan_id, user_id, user_ip) VALUES (:model_stream_id, :org_id, :plan_id, :user_id, :user_ip) RETURNING id, created_at`

// 	row, err := Conn.NamedQuery(query, subscription)

// 	if err != nil {
// 		return fmt.Errorf("error storing model stream subscription: %v", err)
// 	}

// 	defer row.Close()

// 	if row.Next() {
// 		var createdAt time.Time
// 		var id string
// 		if err := row.Scan(&id, &createdAt); err != nil {
// 			return fmt.Errorf("error storing model stream subscription: %v", err)
// 		}

// 		subscription.Id = id
// 		subscription.CreatedAt = createdAt
// 	}

// 	return nil
// }

// func SetModelStreamSubscriptionFinished(id string) error {
// 	_, err := Conn.Exec("UPDATE model_stream_subscriptions SET finished_at = NOW() WHERE id = $1", id)

// 	if err != nil {
// 		return fmt.Errorf("error setting model stream subscription finished: %v", err)
// 	}

// 	return nil
// }
