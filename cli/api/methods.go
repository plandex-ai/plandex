package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"plandex/types"

	"github.com/plandex/plandex/shared"
)

func (a *Api) StartTrial() (*shared.StartTrialResponse, error) {
	serverUrl := cloudApiHost + "/accounts/start_trial"

	resp, err := unauthenticatedClient.Post(serverUrl, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error starting trial: %d - %s", resp.StatusCode, string(errorBody))
	}

	var startTrialResponse shared.StartTrialResponse
	err = json.NewDecoder(resp.Body).Decode(&startTrialResponse)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &startTrialResponse, nil
}

func (a *Api) CreateProject(req shared.CreateProjectRequest) (*shared.CreateProjectResponse, error) {
	serverUrl := getApiHost() + "/projects"
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %v", err)
	}

	resp, err := authenticatedFastClient.Post(serverUrl, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error creating project: %d - %s", resp.StatusCode, string(errorBody))
	}

	var respBody shared.CreateProjectResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &respBody, nil
}

func (a *Api) ListProjects() ([]*shared.Project, error) {
	serverUrl := getApiHost() + "/projects"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing projects: %d - %s", resp.StatusCode, string(errorBody))
	}

	var projects []*shared.Project
	err = json.NewDecoder(resp.Body).Decode(&projects)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return projects, nil
}

func (a *Api) SetProjectPlan(projectId string, req shared.SetProjectPlanRequest) error {
	serverUrl := fmt.Sprintf("%s/projects/%s/set_plan", getApiHost(), projectId)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshalling request: %v", err)
	}

	request, err := http.NewRequest(http.MethodPut, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error setting project plan: %d - %s", resp.StatusCode, string(errorBody))
	}

	return nil
}

func (a *Api) RenameProject(projectId string, req shared.RenameProjectRequest) error {
	serverUrl := fmt.Sprintf("%s/projects/%s/rename", getApiHost(), projectId)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshalling request: %v", err)
	}

	request, err := http.NewRequest(http.MethodPut, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error renaming project: %d - %s", resp.StatusCode, string(errorBody))
	}

	return nil
}

func (a *Api) ListPlans(projectId string) ([]*shared.Plan, error) {
	serverUrl := fmt.Sprintf("%s/projects/%s/plans", getApiHost(), projectId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing plans: %d - %s", resp.StatusCode, string(errorBody))
	}

	var plans []*shared.Plan
	err = json.NewDecoder(resp.Body).Decode(&plans)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return plans, nil
}

func (a *Api) ListArchivedPlans(projectId string) ([]*shared.Plan, error) {
	serverUrl := fmt.Sprintf("%s/projects/%s/plans/archive", getApiHost(), projectId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing archived plans: %d - %s", resp.StatusCode, string(errorBody))
	}

	var plans []*shared.Plan
	err = json.NewDecoder(resp.Body).Decode(&plans)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return plans, nil
}

func (a *Api) ListPlansRunning(projectId string) ([]*shared.Plan, error) {
	serverUrl := fmt.Sprintf("%s/projects/%s/plans/ps", getApiHost(), projectId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing running plans: %d - %s", resp.StatusCode, string(errorBody))
	}

	var plans []*shared.Plan
	err = json.NewDecoder(resp.Body).Decode(&plans)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return plans, nil
}

func (a *Api) CreatePlan(projectId string, req shared.CreatePlanRequest) (*shared.CreatePlanResponse, error) {
	serverUrl := fmt.Sprintf("%s/projects/%s/plans", getApiHost(), projectId)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %v", err)
	}

	resp, err := authenticatedFastClient.Post(serverUrl, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error creating plan: %d - %s", resp.StatusCode, string(errorBody))
	}

	var respBody shared.CreatePlanResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &respBody, nil
}

func (a *Api) GetPlan(planId string) (*shared.Plan, error) {
	serverUrl := fmt.Sprintf("%s/plans/%s", getApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error getting plan: %d - %s", resp.StatusCode, string(errorBody))
	}

	var plan shared.Plan
	err = json.NewDecoder(resp.Body).Decode(&plan)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &plan, nil
}

func (a *Api) DeletePlan(planId string) error {
	serverUrl := fmt.Sprintf("%s/plans/%s", getApiHost(), planId)

	req, err := http.NewRequest(http.MethodDelete, serverUrl, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error deleting plan: %d - %s", resp.StatusCode, string(errorBody))
	}

	return nil
}

func (a *Api) DeleteAllPlans(projectId string) error {
	serverUrl := fmt.Sprintf("%s/projects/%s/plans", getApiHost(), projectId)

	req, err := http.NewRequest(http.MethodDelete, serverUrl, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error deleting all plans: %d - %s", resp.StatusCode, string(errorBody))
	}

	return nil
}

func (a *Api) TellPlan(planId string, req shared.TellPlanRequest, onStream types.OnStreamPlan) error {

	serverUrl := fmt.Sprintf("%s/plans/%s/tell", getApiHost(), planId)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshalling request: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	var client *http.Client
	if req.ConnectStream {
		client = authenticatedStreamingClient
	} else {
		client = authenticatedFastClient
	}

	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error telling plan: %d - %s", resp.StatusCode, string(errorBody))
	}

	if req.ConnectStream {
		log.Println("Connecting stream")
		connectPlanRespStream(resp.Body, onStream)
	}

	return nil
}

func (a *Api) ConnectPlan(planId string, onStream types.OnStreamPlan) error {
	serverUrl := fmt.Sprintf("%s/plans/%s/connect", getApiHost(), planId)

	req, err := http.NewRequest(http.MethodPatch, serverUrl, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	resp, err := authenticatedStreamingClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error connecting plan: %d - %s", resp.StatusCode, string(errorBody))
	}

	connectPlanRespStream(resp.Body, onStream)

	return nil
}

func (a *Api) StopPlan(planId string) error {
	serverUrl := fmt.Sprintf("%s/plans/%s/stop", getApiHost(), planId)

	req, err := http.NewRequest(http.MethodDelete, serverUrl, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error stopping plan: %d - %s", resp.StatusCode, string(errorBody))
	}

	return nil
}

func (a *Api) GetCurrentPlanState(planId string) (*shared.CurrentPlanState, error) {
	serverUrl := fmt.Sprintf("%s/plans/%s/current_plan", getApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error getting current plan state: %d - %s", resp.StatusCode, string(errorBody))
	}

	var state shared.CurrentPlanState
	err = json.NewDecoder(resp.Body).Decode(&state)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &state, nil
}

func (a *Api) ApplyPlan(planId string) error {
	serverUrl := fmt.Sprintf("%s/plans/%s/apply", getApiHost(), planId)

	req, err := http.NewRequest(http.MethodPatch, serverUrl, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error applying plan: %d - %s", resp.StatusCode, string(errorBody))
	}

	return nil
}

func (a *Api) ArchivePlan(planId string) error {
	serverUrl := fmt.Sprintf("%s/plans/%s/archive", getApiHost(), planId)

	req, err := http.NewRequest(http.MethodPatch, serverUrl, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error archiving plan: %d - %s", resp.StatusCode, string(errorBody))
	}

	return nil
}

func (a *Api) RejectAllChanges(planId string) error {
	serverUrl := fmt.Sprintf("%s/plans/%s/reject_all", getApiHost(), planId)

	req, err := http.NewRequest(http.MethodPatch, serverUrl, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error rejecting all changes: %d - %s", resp.StatusCode, string(errorBody))
	}

	return nil
}

func (a *Api) RejectResult(planId, resultId string) error {
	serverUrl := fmt.Sprintf("%s/plans/%s/results/%s/reject", getApiHost(), planId, resultId)

	req, err := http.NewRequest(http.MethodPatch, serverUrl, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error rejecting result: %d - %s", resp.StatusCode, string(errorBody))
	}

	return nil
}

func (a *Api) RejectReplacement(planId, resultId, replacementId string) error {
	serverUrl := fmt.Sprintf("%s/plans/%s/results/%s/replacements/%s/reject", getApiHost(), planId, resultId, replacementId)

	req, err := http.NewRequest(http.MethodPatch, serverUrl, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error rejecting replacement: %d - %s", resp.StatusCode, string(errorBody))
	}

	return nil
}

func (a *Api) LoadContext(planId string, req shared.LoadContextRequest) (*shared.LoadContextResponse, error) {
	serverUrl := fmt.Sprintf("%s/plans/%s/context", getApiHost(), planId)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %v", err)
	}

	// use the slow client since we may be uploading relatively large files
	resp, err := authenticatedSlowClient.Post(serverUrl, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error loading context: %d - %s", resp.StatusCode, string(errorBody))
	}

	var loadContextResponse shared.LoadContextResponse
	err = json.NewDecoder(resp.Body).Decode(&loadContextResponse)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &loadContextResponse, nil
}

func (a *Api) UpdateContext(planId string, req shared.UpdateContextRequest) (*shared.UpdateContextResponse, error) {
	serverUrl := fmt.Sprintf("%s/plans/%s/context", getApiHost(), planId)

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %v", err)
	}

	request, err := http.NewRequest(http.MethodPut, serverUrl, bytes.NewBuffer(reqBytes))

	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")

	// use the slow client since we may be uploading relatively large files
	resp, err := authenticatedSlowClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error loading context: %d - %s", resp.StatusCode, string(errorBody))
	}

	var updateContextResponse shared.UpdateContextResponse
	err = json.NewDecoder(resp.Body).Decode(&updateContextResponse)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &updateContextResponse, nil
}

func (a *Api) DeleteContext(planId string, req shared.DeleteContextRequest) (*shared.DeleteContextResponse, error) {
	serverUrl := fmt.Sprintf("%s/plans/%s/context", getApiHost(), planId)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %v", err)
	}

	request, err := http.NewRequest(http.MethodDelete, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error deleting context: %d - %s", resp.StatusCode, string(errorBody))
	}

	var deleteContextResponse shared.DeleteContextResponse
	err = json.NewDecoder(resp.Body).Decode(&deleteContextResponse)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &deleteContextResponse, nil
}

func (a *Api) ListContext(planId string) ([]*shared.Context, error) {
	serverUrl := fmt.Sprintf("%s/plans/%s/context", getApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing context: %d - %s", resp.StatusCode, string(errorBody))
	}

	var contexts []*shared.Context
	err = json.NewDecoder(resp.Body).Decode(&contexts)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return contexts, nil
}

func (a *Api) ListConvo(planId string) ([]*shared.ConvoMessage, error) {
	serverUrl := fmt.Sprintf("%s/plans/%s/convo", getApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing conversations: %d - %s", resp.StatusCode, string(errorBody))
	}

	var convos []*shared.ConvoMessage
	err = json.NewDecoder(resp.Body).Decode(&convos)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return convos, nil
}

func (a *Api) ListLogs(planId string) (*shared.LogResponse, error) {
	serverUrl := fmt.Sprintf("%s/plans/%s/logs", getApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing logs: %d - %s", resp.StatusCode, string(errorBody))
	}

	var logs shared.LogResponse
	err = json.NewDecoder(resp.Body).Decode(&logs)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &logs, nil
}

func (a *Api) RewindPlan(planId string, req shared.RewindPlanRequest) (*shared.RewindPlanResponse, error) {
	serverUrl := fmt.Sprintf("%s/plans/%s/rewind", getApiHost(), planId)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %v", err)
	}

	request, err := http.NewRequest(http.MethodPatch, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error rewinding plan: %d - %s", resp.StatusCode, string(errorBody))
	}

	var rewindPlanResponse shared.RewindPlanResponse
	err = json.NewDecoder(resp.Body).Decode(&rewindPlanResponse)
	if err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	return &rewindPlanResponse, nil
}
