package db

import (
	"fmt"
	"log"
	"time"

	"github.com/plandex/plandex/shared"
)

func CreatePlan(orgId, projectId, userId, name string) (*Plan, error) {
	query := "INSERT INTO plans (org_id, creator_id, project_id, name) VALUES (:org_id, :creator_id, :project_id, :name) RETURNING id, created_at, updated_at"

	plan := &Plan{
		OrgId:     orgId,
		CreatorId: userId,
		ProjectId: projectId,
		Name:      name,
	}

	row, err := Conn.NamedQuery(query, plan)

	if err != nil {
		return nil, fmt.Errorf("error creating plan: %v", err)
	}

	defer row.Close()

	if row.Next() {
		var createdAt, updatedAt time.Time
		var id string
		if err := row.Scan(&id, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("error creating plan: %v", err)
		}

		plan.Id = id
		plan.CreatedAt = createdAt
		plan.UpdatedAt = updatedAt
	}

	err = InitPlan(orgId, plan.Id)

	if err != nil {
		return nil, fmt.Errorf("error initializing plan dir: %v", err)
	}

	return plan, nil
}

func ListPlans(projectId, userId string, archived bool, status string) ([]shared.Plan, error) {
	qs := "SELECT id, creator_id, name, status, context_tokens, convo_tokens, applied_at, archived_at, created_at, updated_at FROM plans WHERE project_id = $1 AND creator_id = $2 ORDER BY updated_at DESC"
	qargs := []interface{}{projectId, userId}

	if archived {
		qs += " AND archived_at IS NOT NULL"
	} else {
		qs += " AND archived_at IS NULL"
	}

	if status != "" {
		qs += " AND status = $3"
		qargs = append(qargs, status)
	}

	qs += " ORDER BY updated_at DESC"

	res, err := Conn.Query(qs, qargs...)

	if err != nil {
		return nil, fmt.Errorf("error listing plans: %v", err)
	}

	defer res.Close()

	var plans []shared.Plan

	for res.Next() {
		var plan shared.Plan

		err := res.Scan(&plan.Id, &plan.CreatorId, &plan.Name, &plan.Status, &plan.ContextTokens, &plan.ConvoTokens, &plan.AppliedAt, &plan.ArchivedAt, &plan.CreatedAt, &plan.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("error scanning plan: %v", err)
		}
		plans = append(plans, plan)
	}

	return plans, nil
}

func AddPlanContextTokens(planId string, addTokens int) error {
	_, err := Conn.Exec("UPDATE plans SET context_tokens = context_tokens + $1 WHERE id = $2", addTokens, planId)
	if err != nil {
		return fmt.Errorf("error updating plan tokens: %v", err)
	}
	return nil
}

func AddPlanConvoTokens(planId string, addTokens int) error {
	_, err := Conn.Exec("UPDATE plans SET convo_tokens = convo_tokens + $1 WHERE id = $2", addTokens, planId)
	if err != nil {
		return fmt.Errorf("error updating plan tokens: %v", err)
	}
	return nil
}

func GetPlan(planId string) (*shared.Plan, error) {
	var plan shared.Plan

	err := Conn.Get(&plan, "SELECT id, creator_id, name, status, context_tokens, convo_tokens, applied_at, archived_at, created_at, updated_at FROM plans WHERE id = $1", planId)

	if err != nil {
		return nil, fmt.Errorf("error getting plan: %v", err)
	}

	return &plan, nil
}

func SetPlanStatus(planId string, status shared.PlanStatus, errStr string) error {
	_, err := Conn.Exec("UPDATE plans SET status = $1, error = $2 WHERE id = $3", status, errStr, planId)

	if err != nil {
		return fmt.Errorf("error setting plan status: %v", err)
	}

	return nil
}

func StoreDescription(description *ConvoMessageDescription) error {
	query := "INSERT INTO convo_message_descriptions (org_id, plan_id, convo_message_id, summarized_to_message_created_at, made_plan, commit_msg, files, error) VALUES (:id, :org_id, :plan_id, :convo_message_id, :summarized_to_message_created_at, :made_plan, :commit_msg, :files, :error) RETURNING id, created_at, updated_at"

	row, err := Conn.NamedQuery(query, description)

	if err != nil {
		return fmt.Errorf("error storing convo message description: %v", err)
	}

	defer row.Close()

	if row.Next() {
		var createdAt, updatedAt time.Time
		var id string
		if err := row.Scan(&id, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("error storing convo message description: %v", err)
		}

		description.Id = id
		description.CreatedAt = createdAt
		description.UpdatedAt = updatedAt
	}

	return nil
}

func DeleteDraftPlans(orgId, projectId, userId string) error {
	res, err := Conn.Query("DELETE FROM plans WHERE project_id = $1 AND creator_id = $2 AND name = 'draft' RETURNING id;", projectId, userId)
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

func DeletePlans(orgId, projectId, userId string) error {
	res, err := Conn.Query("DELETE FROM plans WHERE project_id = $1 AND creator_id = $2 RETURNING id;", projectId, userId)
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
