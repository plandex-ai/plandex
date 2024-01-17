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

func (a *Api) StartTrial() (*shared.StartTrialResponse, *shared.ApiError) {
	serverUrl := cloudApiHost + "/accounts/start_trial"

	resp, err := unauthenticatedClient.Post(serverUrl, "application/json", nil)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var startTrialResponse shared.StartTrialResponse
	err = json.NewDecoder(resp.Body).Decode(&startTrialResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &startTrialResponse, nil
}

func (a *Api) CreateProject(req shared.CreateProjectRequest) (*shared.CreateProjectResponse, *shared.ApiError) {
	serverUrl := getApiHost() + "/projects"
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	resp, err := authenticatedFastClient.Post(serverUrl, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var respBody shared.CreateProjectResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &respBody, nil
}

func (a *Api) ListProjects() ([]*shared.Project, *shared.ApiError) {
	serverUrl := getApiHost() + "/projects"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var projects []*shared.Project
	err = json.NewDecoder(resp.Body).Decode(&projects)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return projects, nil
}

func (a *Api) SetProjectPlan(projectId string, req shared.SetProjectPlanRequest) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/projects/%s/set_plan", getApiHost(), projectId)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	request, err := http.NewRequest(http.MethodPut, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) RenameProject(projectId string, req shared.RenameProjectRequest) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/projects/%s/rename", getApiHost(), projectId)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	request, err := http.NewRequest(http.MethodPut, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) ListPlans(projectId string) ([]*shared.Plan, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/projects/%s/plans", getApiHost(), projectId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var plans []*shared.Plan
	err = json.NewDecoder(resp.Body).Decode(&plans)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return plans, nil
}

func (a *Api) ListArchivedPlans(projectId string) ([]*shared.Plan, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/projects/%s/plans/archive", getApiHost(), projectId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var plans []*shared.Plan
	err = json.NewDecoder(resp.Body).Decode(&plans)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return plans, nil
}

func (a *Api) ListPlansRunning(projectId string) ([]*shared.Plan, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/projects/%s/plans/ps", getApiHost(), projectId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var plans []*shared.Plan
	err = json.NewDecoder(resp.Body).Decode(&plans)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return plans, nil
}

func (a *Api) CreatePlan(projectId string, req shared.CreatePlanRequest) (*shared.CreatePlanResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/projects/%s/plans", getApiHost(), projectId)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	resp, err := authenticatedFastClient.Post(serverUrl, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var respBody shared.CreatePlanResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &respBody, nil
}

func (a *Api) GetPlan(planId string) (*shared.Plan, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s", getApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var plan shared.Plan
	err = json.NewDecoder(resp.Body).Decode(&plan)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &plan, nil
}

func (a *Api) DeletePlan(planId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s", getApiHost(), planId)

	req, err := http.NewRequest(http.MethodDelete, serverUrl, nil)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) DeleteAllPlans(projectId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/projects/%s/plans", getApiHost(), projectId)

	req, err := http.NewRequest(http.MethodDelete, serverUrl, nil)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) TellPlan(planId string, req shared.TellPlanRequest, onStream types.OnStreamPlan) *shared.ApiError {

	serverUrl := fmt.Sprintf("%s/plans/%s/tell", getApiHost(), planId)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	request, err := http.NewRequest(http.MethodPost, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
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
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	if req.ConnectStream {
		log.Println("Connecting stream")
		connectPlanRespStream(resp.Body, onStream)
	}

	return nil
}

func (a *Api) ConnectPlan(planId string, onStream types.OnStreamPlan) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/connect", getApiHost(), planId)

	req, err := http.NewRequest(http.MethodPatch, serverUrl, nil)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	resp, err := authenticatedStreamingClient.Do(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	connectPlanRespStream(resp.Body, onStream)

	return nil
}

func (a *Api) StopPlan(planId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/stop", getApiHost(), planId)

	req, err := http.NewRequest(http.MethodDelete, serverUrl, nil)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) GetCurrentPlanState(planId string) (*shared.CurrentPlanState, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/current_plan", getApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var state shared.CurrentPlanState
	err = json.NewDecoder(resp.Body).Decode(&state)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &state, nil
}

func (a *Api) ApplyPlan(planId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/apply", getApiHost(), planId)

	req, err := http.NewRequest(http.MethodPatch, serverUrl, nil)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) ArchivePlan(planId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/archive", getApiHost(), planId)

	req, err := http.NewRequest(http.MethodPatch, serverUrl, nil)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) RejectAllChanges(planId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/reject_all", getApiHost(), planId)

	req, err := http.NewRequest(http.MethodPatch, serverUrl, nil)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) RejectResult(planId, resultId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/results/%s/reject", getApiHost(), planId, resultId)

	req, err := http.NewRequest(http.MethodPatch, serverUrl, nil)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) RejectReplacement(planId, resultId, replacementId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/results/%s/replacements/%s/reject", getApiHost(), planId, resultId, replacementId)

	req, err := http.NewRequest(http.MethodPatch, serverUrl, nil)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) LoadContext(planId string, req shared.LoadContextRequest) (*shared.LoadContextResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/context", getApiHost(), planId)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	// use the slow client since we may be uploading relatively large files
	resp, err := authenticatedSlowClient.Post(serverUrl, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var loadContextResponse shared.LoadContextResponse
	err = json.NewDecoder(resp.Body).Decode(&loadContextResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &loadContextResponse, nil
}

func (a *Api) UpdateContext(planId string, req shared.UpdateContextRequest) (*shared.UpdateContextResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/context", getApiHost(), planId)

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	request, err := http.NewRequest(http.MethodPut, serverUrl, bytes.NewBuffer(reqBytes))

	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	request.Header.Set("Content-Type", "application/json")

	// use the slow client since we may be uploading relatively large files
	resp, err := authenticatedSlowClient.Do(request)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var updateContextResponse shared.UpdateContextResponse
	err = json.NewDecoder(resp.Body).Decode(&updateContextResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &updateContextResponse, nil
}

func (a *Api) DeleteContext(planId string, req shared.DeleteContextRequest) (*shared.DeleteContextResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/context", getApiHost(), planId)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	request, err := http.NewRequest(http.MethodDelete, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error creating request: %v", err)}
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var deleteContextResponse shared.DeleteContextResponse
	err = json.NewDecoder(resp.Body).Decode(&deleteContextResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &deleteContextResponse, nil
}

func (a *Api) ListContext(planId string) ([]*shared.Context, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/context", getApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var contexts []*shared.Context
	err = json.NewDecoder(resp.Body).Decode(&contexts)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return contexts, nil
}

func (a *Api) ListConvo(planId string) ([]*shared.ConvoMessage, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/convo", getApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var convos []*shared.ConvoMessage
	err = json.NewDecoder(resp.Body).Decode(&convos)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return convos, nil
}

func (a *Api) ListLogs(planId string) (*shared.LogResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/logs", getApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var logs shared.LogResponse
	err = json.NewDecoder(resp.Body).Decode(&logs)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &logs, nil
}

func (a *Api) RewindPlan(planId string, req shared.RewindPlanRequest) (*shared.RewindPlanResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/rewind", getApiHost(), planId)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	request, err := http.NewRequest(http.MethodPatch, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error creating request: %v", err)}
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var rewindPlanResponse shared.RewindPlanResponse
	err = json.NewDecoder(resp.Body).Decode(&rewindPlanResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &rewindPlanResponse, nil
}

func (a *Api) SignIn(req shared.SignInRequest, customHost string) (*shared.SessionResponse, *shared.ApiError) {
	serverUrl := customHost + "/accounts/sign_in"
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	resp, err := unauthenticatedClient.Post(serverUrl, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var sessionResponse shared.SessionResponse
	err = json.NewDecoder(resp.Body).Decode(&sessionResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &sessionResponse, nil
}

func (a *Api) CreateAccount(req shared.CreateAccountRequest, customHost string) (*shared.SessionResponse, *shared.ApiError) {
	serverUrl := customHost + "/accounts"
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	resp, err := unauthenticatedClient.Post(serverUrl, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var sessionResponse shared.SessionResponse
	err = json.NewDecoder(resp.Body).Decode(&sessionResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &sessionResponse, nil
}

func (a *Api) ConvertTrial(req shared.ConvertTrialRequest) (*shared.SessionResponse, *shared.ApiError) {
	serverUrl := getApiHost() + "/accounts/convert_trial"
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	resp, err := authenticatedFastClient.Post(serverUrl, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var sessionResponse shared.SessionResponse
	err = json.NewDecoder(resp.Body).Decode(&sessionResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &sessionResponse, nil
}

func (a *Api) CreateOrg(req shared.CreateOrgRequest) (*shared.CreateOrgResponse, *shared.ApiError) {
	serverUrl := getApiHost() + "/orgs"
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	resp, err := authenticatedFastClient.Post(serverUrl, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var createOrgResponse shared.CreateOrgResponse
	err = json.NewDecoder(resp.Body).Decode(&createOrgResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &createOrgResponse, nil
}

func (a *Api) ListOrgs() ([]*shared.Org, *shared.ApiError) {
	serverUrl := getApiHost() + "/orgs"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var orgs []*shared.Org
	err = json.NewDecoder(resp.Body).Decode(&orgs)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return orgs, nil
}

func (a *Api) DeleteUser(userId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/orgs/users/%s", getApiHost(), userId)
	req, err := http.NewRequest(http.MethodDelete, serverUrl, nil)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) InviteUser(req shared.InviteRequest) *shared.ApiError {
	serverUrl := getApiHost() + "/invites"
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	resp, err := authenticatedFastClient.Post(serverUrl, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) ListPendingInvites() ([]*shared.Invite, *shared.ApiError) {
	serverUrl := getApiHost() + "/invites/pending"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var invites []*shared.Invite
	err = json.NewDecoder(resp.Body).Decode(&invites)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return invites, nil
}

func (a *Api) ListAcceptedInvites() ([]*shared.Invite, *shared.ApiError) {
	serverUrl := getApiHost() + "/invites/accepted"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var invites []*shared.Invite
	err = json.NewDecoder(resp.Body).Decode(&invites)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return invites, nil
}

func (a *Api) ListAllInvites() ([]*shared.Invite, *shared.ApiError) {
	serverUrl := getApiHost() + "/invites/all"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var invites []*shared.Invite
	err = json.NewDecoder(resp.Body).Decode(&invites)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return invites, nil
}

func (a *Api) DeleteInvite(inviteId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/invites/%s", getApiHost(), inviteId)
	req, err := http.NewRequest(http.MethodDelete, serverUrl, nil)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) CreateEmailVerification(email, customHost, userId string) (*shared.CreateEmailVerificationResponse, *shared.ApiError) {
	serverUrl := customHost + "/accounts/email_verifications"
	req := shared.CreateEmailVerificationRequest{Email: email, UserId: userId}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	resp, err := unauthenticatedClient.Post(serverUrl, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var verificationResponse shared.CreateEmailVerificationResponse
	err = json.NewDecoder(resp.Body).Decode(&verificationResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &verificationResponse, nil
}

func (a *Api) SignOut() *shared.ApiError {
	serverUrl := getApiHost() + "/accounts/sign_out"

	req, err := http.NewRequest(http.MethodPost, serverUrl, nil)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		return handleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) ListUsers() ([]*shared.User, *shared.ApiError) {
	serverUrl := getApiHost() + "/users"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return nil, apiErr
	}

	var users []*shared.User
	err = json.NewDecoder(resp.Body).Decode(&users)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return users, nil
}
