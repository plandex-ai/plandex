package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"

	shared "plandex-shared"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/sashabaranov/go-openai"
)

func CreatePlan(ctx context.Context, orgId, projectId, userId, name string) (*Plan, error) {
	var plan *Plan
	err := WithTx(ctx, "create plan", func(tx *sqlx.Tx) error {

		planConfig, err := GetDefaultPlanConfig(userId)

		if err != nil {
			return fmt.Errorf("error getting default plan config: %v", err)
		}

		query := `INSERT INTO plans (org_id, owner_id, project_id, name, plan_config) 
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id, created_at, updated_at`

		plan = &Plan{
			OrgId:      orgId,
			OwnerId:    userId,
			ProjectId:  projectId,
			Name:       name,
			PlanConfig: planConfig,
		}

		err = tx.QueryRow(
			query,
			orgId,
			userId,
			projectId,
			name,
			planConfig,
		).Scan(
			&plan.Id,
			&plan.CreatedAt,
			&plan.UpdatedAt,
		)

		if err != nil {
			return fmt.Errorf("error creating plan: %v", err)
		}

		_, err = tx.Exec("INSERT INTO lockable_plan_ids (plan_id) VALUES ($1)", plan.Id)

		if err != nil {
			return fmt.Errorf("error inserting lockable plan id: %v", err)
		}

		// the one place where we do this to skip the locking queue
		// ok to cheat this once since we're creating a new plan
		repo := getGitRepo(orgId, plan.Id)
		_, err = CreateBranch(repo, plan, nil, "main", tx)

		if err != nil {
			return fmt.Errorf("error creating main branch: %v", err)
		}

		log.Println("Created branch main")

		err = InitPlan(orgId, plan.Id)

		if err != nil {
			return fmt.Errorf("error initializing plan dir: %v", err)
		}

		log.Println("Initialized plan dir")

		return nil
	})

	if err != nil {
		return nil, err
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

func GetPlanNamesById(planIds []string) (map[string]string, error) {
	var plans []*Plan
	err := Conn.Select(&plans, "SELECT id, name FROM plans WHERE id = ANY($1)", pq.Array(planIds))
	if err != nil {
		return nil, fmt.Errorf("error getting plan names: %v", err)
	}

	namesMap := make(map[string]string)
	for _, plan := range plans {
		namesMap[plan.Id] = plan.Name
	}

	return namesMap, nil
}

func AddPlanContextTokens(planId, branch string, addTokens int) error {
	_, err := Conn.Exec("UPDATE branches SET context_tokens = context_tokens + $1 WHERE plan_id = $2 AND name = $3", addTokens, planId, branch)
	if err != nil {
		return fmt.Errorf("error updating plan tokens: %v", err)
	}
	return nil
}

func AddPlanConvoMessage(msg *ConvoMessage, branch string) error {
	errCh := make(chan error, 2)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in AddPlanConvoMessage: %v\n%s", r, debug.Stack())
				errCh <- fmt.Errorf("panic in AddPlanConvoMessage: %v\n%s", r, debug.Stack())
				runtime.Goexit() // don't allow outer function to continue and double-send to channel
			}
		}()

		_, err := Conn.Exec("UPDATE branches SET convo_tokens = convo_tokens + $1 WHERE plan_id = $2 AND name = $3", msg.Tokens, msg.PlanId, branch)

		if err != nil {
			errCh <- fmt.Errorf("error updating plan tokens: %v", err)
			return
		}

		errCh <- nil
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in AddPlanConvoMessage: %v\n%s", r, debug.Stack())
				errCh <- fmt.Errorf("panic in AddPlanConvoMessage: %v\n%s", r, debug.Stack())
				runtime.Goexit() // don't allow outer function to continue and double-send to channel
			}
		}()

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
	errCh := make(chan error, 2)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in SyncPlanTokens: %v\n%s", r, debug.Stack())
				errCh <- fmt.Errorf("panic in SyncPlanTokens: %v\n%s", r, debug.Stack())
				runtime.Goexit() // don't allow outer function to continue and double-send to channel
			}
		}()
		var err error
		contexts, err = GetPlanContexts(orgId, planId, false, false)
		errCh <- err
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic in SyncPlanTokens: %v\n%s", r, debug.Stack())
				errCh <- fmt.Errorf("panic in SyncPlanTokens: %v\n%s", r, debug.Stack())
				runtime.Goexit() // don't allow outer function to continue and double-send to channel
			}
		}()
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

func RenamePlan(planId string, name string, tx *sqlx.Tx) error {
	var err error
	if tx == nil {
		_, err = Conn.Exec("UPDATE plans SET name = $1 WHERE id = $2", name, planId)
	} else {
		_, err = tx.Exec("UPDATE plans SET name = $1 WHERE id = $2", name, planId)
	}

	if err != nil {
		return fmt.Errorf("error renaming plan: %v", err)
	}

	return nil
}

func IncActiveBranches(planId string, inc int, tx *sqlx.Tx) error {
	_, err := tx.Exec("UPDATE plans SET active_branches = active_branches + $1 WHERE id = $2", inc, planId)

	if err != nil {
		return fmt.Errorf("error updating plan active branches: %v", err)
	}

	return nil
}

func IncNumNonDraftPlans(userId string, tx *sqlx.Tx) error {
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

	for _, op := range description.Operations {
		if op.Content != "" {
			quoted := strconv.Quote(op.Content)
			op.Content = quoted[1 : len(quoted)-1]
		}
		if op.Description != "" {
			quoted := strconv.Quote(op.Description)
			op.Description = quoted[1 : len(quoted)-1]
		}
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

	errCh := make(chan error, len(ids))
	for _, planId := range ids {
		go func(planId string) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in DeleteDraftPlans: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in DeleteDraftPlans: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
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

	errCh := make(chan error, len(ids))
	for _, planId := range ids {
		go func(planId string) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic in DeleteOwnerPlans: %v\n%s", r, debug.Stack())
					errCh <- fmt.Errorf("panic in DeleteOwnerPlans: %v\n%s", r, debug.Stack())
					runtime.Goexit() // don't allow outer function to continue and double-send to channel
				}
			}()
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

func GetPlanIdsForProject(projectId string) ([]string, error) {
	var ids []string
	err := Conn.Select(&ids, "SELECT id FROM plans WHERE project_id = $1", projectId)
	if err != nil {
		return nil, fmt.Errorf("error getting plan ids for project: %v", err)
	}
	return ids, nil
}
