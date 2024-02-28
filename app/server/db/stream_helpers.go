package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
	"github.com/plandex/plandex/shared"
)

const modelStreamHeartbeatInterval = 1 * time.Second
const modelStreamHeartbeatTimeout = 5 * time.Second

func StoreModelStream(stream *ModelStream, ctx context.Context, cancelFn context.CancelFunc) error {
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

	// Start a goroutine to keep the lock alive
	go func() {
		numErrors := 0
		for {
			select {
			case <-ctx.Done():
				err := SetModelStreamFinished(stream.Id)
				if err != nil {
					log.Printf("Error setting model stream %s finished: %v\n", stream.Id, err)
				}
				return

			default:
				_, err := Conn.Exec("UPDATE model_streams SET last_heartbeat_at = NOW() WHERE id = $1", stream.Id)

				if err != nil {
					log.Printf("Error updating model stream last heartbeat: %v\n", err)
					numErrors++

					if numErrors > 5 {
						log.Printf("Too many errors updating model stream last heartbeat: %v\n", err)
						cancelFn()
						return
					}
				}

				time.Sleep(modelStreamHeartbeatInterval)
			}

		}
	}()

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

	if time.Now().Add(-modelStreamHeartbeatTimeout).After(stream.LastHeartbeatAt) {
		log.Printf("Model stream %s has not sent a heartbeat in %s\n", stream.Id, modelStreamHeartbeatTimeout)

		err := SetModelStreamFinished(stream.Id)

		if err != nil {
			return nil, fmt.Errorf("error setting model stream finished: %v", err)
		}

		err = SetPlanStatus(planId, branch, shared.PlanStatusError, "Model stream has not sent a heartbeat in 5 seconds")

		if err != nil {
			return nil, fmt.Errorf("error setting plan status to error: %v", err)
		}

		return nil, nil
	} else {
		log.Printf("Model stream %s sent heartbeat %d seconds ago\n", stream.Id, int(time.Since(stream.LastHeartbeatAt).Seconds()))
	}

	return &stream, nil
}

func GetActiveOrRecentModelStreams(planIds []string) ([]*ModelStream, error) {
	var streams []*ModelStream
	err := Conn.Select(&streams, "SELECT * FROM model_streams WHERE plan_id = ANY($1) AND (finished_at IS NULL OR finished_at > NOW() - INTERVAL '1 hour') ORDER BY created_at", pq.Array(planIds))

	if err != nil {
		return nil, fmt.Errorf("error getting active or recent model streams: %v", err)
	}

	return streams, nil
}

func GetActiveModelStreams(planIds []string) ([]*ModelStream, error) {
	var streams []*ModelStream
	err := Conn.Select(&streams, "SELECT * FROM model_streams WHERE plan_id = ANY($1) AND finished_at IS NULL ORDER BY created_at", pq.Array(planIds))

	if err != nil {
		return nil, fmt.Errorf("error getting active  model streams: %v", err)
	}

	return streams, nil
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
