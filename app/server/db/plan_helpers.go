package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func CreatePlan(orgId, projectId, userId, name string) (*Plan, error) {
	// start a transaction
	tx, err := Conn.Begin()
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %v", err)
	}

	// Ensure that rollback is attempted in case of failure
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("transaction rollback error: %v\n", rbErr)
			} else {
				log.Println("transaction rolled back")
			}
		}
	}()

	query := `INSERT INTO plans (org_id, owner_id, project_id, name) 
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at, updated_at`

	plan := &Plan{
		OrgId:     orgId,
		OwnerId:   userId,
		ProjectId: projectId,
		Name:      name,
	}

	err = tx.QueryRow(
		query,
		orgId,
		userId,
		projectId,
		name,
	).Scan(
		&plan.Id,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("error creating plan: %v", err)
	}

	_, err = CreateBranch(plan, nil, "main", tx)

	if err != nil {
		return nil, fmt.Errorf("error creating main branch: %v", err)
	}

	log.Println("Created branch main")

	err = InitPlan(orgId, plan.Id)

	if err != nil {
		return nil, fmt.Errorf("error initializing plan dir: %v", err)
	}

	log.Println("Initialized plan dir")

	// commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error committing transaction: %v", err)
	}

	return plan, nil
}

func ListOwnedPlans(projectIds []string, userId string, archived bool) ([]*Plan, error) {
	qs := "SELECT * FROM plans WHERE project_id = ANY($1) AND owner_id = $2"
	qargs := []interface{}{pq.Array(projectIds), userId}

	if archived {
		qs += " AND archived_at IS NOT NULL"
	} else {
		qs += " AND archived_at IS NULL"
	}

	qs += " ORDER BY updated_at DESC"

	var plans []*Plan
	err := Conn.Select(&plans, qs, qargs...)

	if err != nil {
		return nil, fmt.Errorf("error listing plans: %v", err)
	}

	return plans, nil
}

func AddPlanContextTokens(planId, branch string, addTokens int) error {
	_, err := Conn.Exec("UPDATE branches SET context_tokens = context_tokens + $1 WHERE plan_id = $2 AND name = $3", addTokens, planId, branch)
	if err != nil {
		return fmt.Errorf("error updating plan tokens: %v", err)
	}
	return nil
}

func AddPlanConvoMessage(msg *ConvoMessage, branch string) error {
	errCh := make(chan error)

	go func() {
		_, err := Conn.Exec("UPDATE branches SET convo_tokens = convo_tokens + $1 WHERE plan_id = $2 AND name = $3", msg.Tokens, msg.PlanId, branch)

		if err != nil {
			errCh <- fmt.Errorf("error updating plan tokens: %v", err)
			return
		}

		errCh <- nil
	}()

	go func() {
		if msg.Role != openai.ChatMessageRoleAssistant {
			errCh <- nil
			return
		}
		_, err := Conn.Exec("UPDATE plans SET total_replies = total_replies + 1 WHERE id = $1", msg.PlanId)
		if err != nil {
			errCh <- fmt.Errorf("error updating plan total replies: %v", err)
		}

		errCh <- nil
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error updating plan tokens: %v", err)
		}
	}

	return nil
}

func SyncPlanTokens(orgId, planId, branch string) error {
	var contexts []*Context
	var convos []*ConvoMessage
	errCh := make(chan error)

	go func() {
		var err error
		contexts, err = GetPlanContexts(orgId, planId, false)
		errCh <- err
	}()

	go func() {
		var err error
		convos, err = GetPlanConvo(orgId, planId)
		errCh <- err
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error getting contexts or convo: %v", err)
		}
	}

	contextTokens := 0
	for _, context := range contexts {
		contextTokens += context.NumTokens
	}

	convoTokens := 0
	for _, msg := range convos {
		convoTokens += msg.Tokens
	}

	_, err := Conn.Exec("UPDATE branches SET context_tokens = $1, convo_tokens = $2 WHERE plan_id = $3 AND name = $4", contextTokens, convoTokens, planId, branch)

	if err != nil {
		return fmt.Errorf("error updating plan tokens: %v", err)
	}

	return nil
}

func GetPlan(planId string) (*Plan, error) {
	var plan Plan

	err := Conn.Get(&plan, "SELECT * FROM plans WHERE id = $1", planId)

	if err != nil {
		return nil, fmt.Errorf("error getting plan: %v", err)
	}

	return &plan, nil
}

func SetPlanStatus(planId, branch string, status shared.PlanStatus, errStr string) error {
	_, err := Conn.Exec("UPDATE branches SET status = $1, error = $2 WHERE plan_id = $3 AND name = $4", status, errStr, planId, branch)

	if err != nil {
		return fmt.Errorf("error setting plan status: %v", err)
	}

	return nil
}

func RenamePlan(planId string, name string, tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE plans SET name = $1 WHERE id = $2", name, planId)

	if err != nil {
		return fmt.Errorf("error renaming plan: %v", err)
	}

	return nil
}

func IncActiveBranches(planId string, inc int, tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE plans SET active_branches = active_branches + $1 WHERE id = $2", inc, planId)

	if err != nil {
		return fmt.Errorf("error updating plan active branches: %v", err)
	}

	return nil
}

func IncNumNonDraftPlans(userId string, tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE users SET num_non_draft_plans = num_non_draft_plans + 1 WHERE id = $1", userId)

	if err != nil {
		return fmt.Errorf("error updating user num_non_draft_plans: %v", err)
	}

	return nil
}

func StoreDescription(description *ConvoMessageDescription) error {
	descriptionsDir := getPlanDescriptionsDir(description.OrgId, description.PlanId)

	err := os.MkdirAll(descriptionsDir, os.ModePerm)

	if err != nil {
		return fmt.Errorf("error creating convo message descriptions dir: %v", err)
	}

	now := time.Now()

	if description.Id == "" {
		description.Id = uuid.New().String()
		description.CreatedAt = now
	}
	description.UpdatedAt = now

	bytes, err := json.Marshal(description)

	if err != nil {
		return fmt.Errorf("error marshalling convo message description: %v", err)
	}

	err = os.WriteFile(filepath.Join(descriptionsDir, description.Id+".json"), bytes, os.ModePerm)

	if err != nil {
		return fmt.Errorf("error writing convo message description: %v", err)
	}

	return nil
}

func DeleteDraftPlans(orgId, projectId, userId string) error {
	res, err := Conn.Query("DELETE FROM plans WHERE project_id = $1 AND owner_id = $2 AND name = 'draft' RETURNING id;", projectId, userId)
	if err != nil {
		return fmt.Errorf("error deleting draft plans: %v", err)
	}

	defer res.Close()

	// get ids
	var ids []string

	for res.Next() {
		var id string
		err := res.Scan(&id)
		if err != nil {
			return fmt.Errorf("error scanning deleted draft plan id: %v", err)
		}
		ids = append(ids, id)
	}

	errCh := make(chan error)
	for _, planId := range ids {
		go func(planId string) {
			errCh <- DeletePlanDir(orgId, planId)
		}(planId)
	}

	for i := 0; i < len(ids); i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error deleting draft plan dir: %v", err)
		}
	}

	if len(ids) > 0 {
		log.Println("Deleted", len(ids), "draft plans")
	}

	return nil
}

func DeleteOwnerPlans(orgId, projectId, userId string) error {
	res, err := Conn.Query("DELETE FROM plans WHERE project_id = $1 AND owner_id = $2 RETURNING id;", projectId, userId)
	if err != nil {
		return fmt.Errorf("error deleting plans: %v", err)
	}

	defer res.Close()

	// get ids
	var ids []string

	for res.Next() {
		var id string
		err := res.Scan(&id)
		if err != nil {
			return fmt.Errorf("error scanning deleted draft plan id: %v", err)
		}
		ids = append(ids, id)
	}

	errCh := make(chan error)
	for _, planId := range ids {
		go func(planId string) {
			errCh <- DeletePlanDir(orgId, planId)
		}(planId)
	}

	for i := 0; i < len(ids); i++ {
		err := <-errCh
		if err != nil {
			return fmt.Errorf("error deleting plan dir: %v", err)
		}
	}

	if len(ids) > 0 {
		log.Println("Deleted", len(ids), "plans")
	}

	return nil
}

func ValidatePlanAccess(planId, userId, orgId string) (*Plan, error) {
	// get plan
	plan, err := GetPlan(planId)

	if err != nil {
		return nil, fmt.Errorf("error getting plan: %v", err)
	}

	if plan == nil {
		return nil, nil
	}

	if plan.OrgId != orgId {
		return nil, nil
	}

	hasProjectAccess, err := ProjectExists(orgId, plan.ProjectId)

	if err != nil {
		return nil, fmt.Errorf("error validating project membership: %v", err)
	}

	if !hasProjectAccess {
		return nil, nil
	}

	// owner has access
	if plan.OwnerId == userId {
		return plan, nil
	}

	// plan is shared with org
	if plan.SharedWithOrgAt != nil {
		return plan, nil
	}

	return nil, nil
}

func BumpPlanUpdatedAt(planId string, t time.Time) error {
	_, err := Conn.Exec("UPDATE plans SET updated_at = $1 WHERE id = $2", t, planId)

	if err != nil {
		return fmt.Errorf("error updating plan updated at: %v", err)
	}

	return nil
}
