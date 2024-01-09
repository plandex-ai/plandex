package db

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
)

func CreatePlan(orgId, projectId, userId, name string) (*Plan, error) {
	query := "INSERT INTO plans (org_id, owner_id, project_id, name, status) VALUES (:org_id, :owner_id, :project_id, :name, :status) RETURNING id, created_at, updated_at"

	plan := &Plan{
		OrgId:     orgId,
		OwnerId:   userId,
		ProjectId: projectId,
		Name:      name,
		Status:    shared.PlanStatusDraft,
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

func ListOwnedPlans(projectId, userId string, archived bool, status string) ([]shared.Plan, error) {
	qs := "SELECT id, owner_id, name, status, context_tokens, convo_tokens, shared_with_org_at, archived_at, created_at, updated_at FROM plans WHERE project_id = $1 AND owner_id = $2"
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

		err := res.Scan(&plan.Id, &plan.OwnerId, &plan.Name, &plan.Status, &plan.ContextTokens, &plan.ConvoTokens, &plan.SharedWithOrgAt, &plan.ArchivedAt, &plan.CreatedAt, &plan.UpdatedAt)

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

func AddPlanConvoMessage(planId string, addTokens int) error {
	_, err := Conn.Exec("UPDATE plans SET convo_tokens = convo_tokens + $1, total_messages = total_messages + 1 WHERE id = $2", addTokens, planId)
	if err != nil {
		return fmt.Errorf("error updating plan tokens: %v", err)
	}
	return nil
}

func SyncPlanTokens(orgId, planId string) error {
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

	_, err := Conn.Exec("UPDATE plans SET context_tokens = $1, convo_tokens = $2 WHERE id = $3", contextTokens, convoTokens, planId)

	if err != nil {
		return fmt.Errorf("error updating plan tokens: %v", err)
	}

	return nil
}

func GetPlan(planId string) (*Plan, error) {
	var plan Plan

	err := Conn.Get(&plan, "SELECT id, org_id, owner_id, project_id, name, status, error, context_tokens, convo_tokens, shared_with_org_at, total_messages, archived_at, created_at, updated_at FROM plans WHERE id = $1", planId)

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

func RenamePlan(planId string, name string) error {
	_, err := Conn.Exec("UPDATE plans SET name = $1 WHERE id = $2", name, planId)

	if err != nil {
		return fmt.Errorf("error renaming plan: %v", err)
	}

	return nil
}

func IncNumNonDraftPlans(userId string) error {
	_, err := Conn.Exec("UPDATE users SET num_non_draft_plans = num_non_draft_plans + 1 WHERE id = $1", userId)

	if err != nil {
		return fmt.Errorf("error updating user num_non_draft_plans: %v", err)
	}

	return nil
}

func StoreDescription(description *ConvoMessageDescription) error {
	descriptionsDir := getPlanDescriptionsDir(description.OrgId, description.PlanId)

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

	hasProjectAccess, err := ValidateProjectAccess(plan.ProjectId, userId, orgId)

	if err != nil {
		return nil, fmt.Errorf("error validating project membership: %v", err)
	}

	if !hasProjectAccess {
		return nil, nil
	}

	// owner has access
	if plan.ProjectId == userId {
		return plan, nil
	}

	// plan is shared with org
	if plan.SharedWithOrgAt != nil {
		return plan, nil
	}

	return nil, nil
}
