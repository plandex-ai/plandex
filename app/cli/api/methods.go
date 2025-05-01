package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"plandex-cli/types"
	"strings"

	shared "plandex-shared"

	"github.com/shopspring/decimal"
)

func (a *Api) CreateCliTrialSession() (string, *shared.ApiError) {
	serverUrl := CloudApiHost + "/accounts/cli_trial_session"

	resp, err := unauthenticatedClient.Post(serverUrl, "application/json", nil)

	if err != nil {
		return "", &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		return "", apiErr
	}

	bytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error reading response: %v", err)}
	}

	return string(bytes), nil
}

func (a *Api) GetCliTrialSession(token string) (*shared.SessionResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/accounts/cli_trial_session/%s", CloudApiHost, token)

	resp, err := unauthenticatedClient.Get(serverUrl)

	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}

	if resp.StatusCode == 404 {
		return nil, nil
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		return nil, apiErr
	}

	var session shared.SessionResponse
	err = json.NewDecoder(resp.Body).Decode(&session)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &session, nil
}

func (a *Api) CreateProject(req shared.CreateProjectRequest) (*shared.CreateProjectResponse, *shared.ApiError) {
	serverUrl := GetApiHost() + "/projects"

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
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.CreateProject(req)
		}
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
	serverUrl := GetApiHost() + "/projects"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListProjects()
		}
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
	serverUrl := fmt.Sprintf("%s/projects/%s/set_plan", GetApiHost(), projectId)
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
		apiErr := HandleApiError(resp, errorBody)
		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)
		if didRefresh {
			return a.SetProjectPlan(projectId, req)
		}
		return apiErr
	}

	return nil
}

func (a *Api) RenameProject(projectId string, req shared.RenameProjectRequest) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/projects/%s/rename", GetApiHost(), projectId)
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
		apiErr := HandleApiError(resp, errorBody)

		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)
		if didRefresh {
			return a.RenameProject(projectId, req)
		}
		return apiErr
	}

	return nil
}
func (a *Api) ListPlans(projectIds []string) ([]*shared.Plan, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans?", GetApiHost())
	parts := []string{}
	for _, projectId := range projectIds {
		parts = append(parts, fmt.Sprintf("projectId=%s", projectId))
	}
	serverUrl += strings.Join(parts, "&")

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)

		apiErr := HandleApiError(resp, errorBody)

		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)
		if didRefresh {
			return a.ListPlans(projectIds)
		}
		return nil, apiErr
	}

	var plans []*shared.Plan
	err = json.NewDecoder(resp.Body).Decode(&plans)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return plans, nil
}

func (a *Api) ListArchivedPlans(projectIds []string) ([]*shared.Plan, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/archive?", GetApiHost())
	parts := []string{}
	for _, projectId := range projectIds {
		parts = append(parts, fmt.Sprintf("projectId=%s", projectId))
	}
	serverUrl += strings.Join(parts, "&")

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListArchivedPlans(projectIds)
		}
		return nil, apiErr
	}

	var plans []*shared.Plan
	err = json.NewDecoder(resp.Body).Decode(&plans)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return plans, nil
}

func (a *Api) ListPlansRunning(projectIds []string, includeRecent bool) (*shared.ListPlansRunningResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/ps?", GetApiHost())
	parts := []string{}
	for _, projectId := range projectIds {
		parts = append(parts, fmt.Sprintf("projectId=%s", projectId))
	}
	serverUrl += strings.Join(parts, "&")
	if includeRecent {
		serverUrl += "&recent=true"
	}

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListPlansRunning(projectIds, includeRecent)
		}
		return nil, apiErr
	}

	var respBody *shared.ListPlansRunningResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return respBody, nil
}

func (a *Api) GetCurrentBranchByPlanId(projectId string, req shared.GetCurrentBranchByPlanIdRequest) (map[string]*shared.Branch, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/projects/%s/plans/current_branches", GetApiHost(), projectId)

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	request, err := http.NewRequest(http.MethodPost, serverUrl, bytes.NewBuffer(reqBytes))

	if err != nil {
		return nil, &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)

	if err != nil {
		return nil, &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)

		apiErr := HandleApiError(resp, errorBody)

		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)
		if didRefresh {
			return a.GetCurrentBranchByPlanId(projectId, req)
		}
		return nil, apiErr
	}

	var respBody map[string]*shared.Branch
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		return nil, &shared.ApiError{Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return respBody, nil
}

func (a *Api) CreatePlan(projectId string, req shared.CreatePlanRequest) (*shared.CreatePlanResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/projects/%s/plans", GetApiHost(), projectId)
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
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.CreatePlan(projectId, req)
		}
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
	serverUrl := fmt.Sprintf("%s/plans/%s", GetApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.GetPlan(planId)
		}
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
	serverUrl := fmt.Sprintf("%s/plans/%s", GetApiHost(), planId)

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
		apiErr := HandleApiError(resp, errorBody)

		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)
		if didRefresh {
			return a.DeletePlan(planId)
		}
		return apiErr
	}

	return nil
}

func (a *Api) DeleteAllPlans(projectId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/projects/%s/plans", GetApiHost(), projectId)

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
		apiErr := HandleApiError(resp, errorBody)

		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)

		if didRefresh {
			return a.DeleteAllPlans(projectId)
		}
		return apiErr
	}

	return nil
}

func (a *Api) TellPlan(planId, branch string, req shared.TellPlanRequest, onStream types.OnStreamPlan) *shared.ApiError {

	serverUrl := fmt.Sprintf("%s/plans/%s/%s/tell", GetApiHost(), planId, branch)
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
		apiErr := HandleApiError(resp, errorBody)

		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)

		if didRefresh {
			return a.TellPlan(planId, branch, req, onStream)
		}
		return apiErr
	}

	if req.ConnectStream {
		log.Println("Connecting stream")
		connectPlanRespStream(resp.Body, onStream)
	} else {
		// log.Println("Background exec - not connecting stream")
		resp.Body.Close()
	}

	return nil
}

func (a *Api) BuildPlan(planId, branch string, req shared.BuildPlanRequest, onStream types.OnStreamPlan) *shared.ApiError {

	log.Println("Calling BuildPlan")

	serverUrl := fmt.Sprintf("%s/plans/%s/%s/build", GetApiHost(), planId, branch)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	request, err := http.NewRequest(http.MethodPatch, serverUrl, bytes.NewBuffer(reqBytes))
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
		log.Println("Error response from build plan", resp.StatusCode)

		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)

		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)

		if didRefresh {
			return a.BuildPlan(planId, branch, req, onStream)
		}
		return apiErr
	}

	if req.ConnectStream {
		log.Println("Connecting stream")
		connectPlanRespStream(resp.Body, onStream)
	} else {
		// log.Println("Background exec - not connecting stream")
		resp.Body.Close()
	}

	return nil
}

func (a *Api) RespondMissingFile(planId, branch string, req shared.RespondMissingFileRequest) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/respond_missing_file", GetApiHost(), planId, branch)

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	request, err := http.NewRequest(http.MethodPost, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)

		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)

		if didRefresh {
			return a.RespondMissingFile(planId, branch, req)
		}
		return apiErr
	}

	return nil

}

func (a *Api) ConnectPlan(planId, branch string, onStream types.OnStreamPlan) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/connect", GetApiHost(), planId, branch)

	req, err := http.NewRequest(http.MethodPatch, serverUrl, nil)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	resp, err := authenticatedStreamingClient.Do(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)

		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)

		if didRefresh {
			return a.ConnectPlan(planId, branch, onStream)
		}

		return apiErr
	}

	connectPlanRespStream(resp.Body, onStream)

	return nil
}

func (a *Api) StopPlan(ctx context.Context, planId, branch string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/stop", GetApiHost(), planId, branch)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, serverUrl, nil)
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
		apiErr := HandleApiError(resp, errorBody)
		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)
		if didRefresh {
			return a.StopPlan(ctx, planId, branch)
		}
		return apiErr
	}

	return nil
}

func (a *Api) GetCurrentPlanState(planId, branch string) (*shared.CurrentPlanState, *shared.ApiError) {
	return a.getCurrentPlanState(planId, branch, "")
}

func (a *Api) GetCurrentPlanStateAtSha(planId, sha string) (*shared.CurrentPlanState, *shared.ApiError) {
	return a.getCurrentPlanState(planId, "", sha)
}

func (a *Api) getCurrentPlanState(planId, branch, sha string) (*shared.CurrentPlanState, *shared.ApiError) {
	var serverUrl string
	if sha != "" {
		serverUrl = fmt.Sprintf("%s/plans/%s/current_plan/%s", GetApiHost(), planId, sha)
	} else {
		serverUrl = fmt.Sprintf("%s/plans/%s/%s/current_plan", GetApiHost(), planId, branch)
	}

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.getCurrentPlanState(planId, branch, sha)
		}
		return nil, apiErr
	}

	var state shared.CurrentPlanState
	err = json.NewDecoder(resp.Body).Decode(&state)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &state, nil
}

func (a *Api) ApplyPlan(planId, branch string, req shared.ApplyPlanRequest) (string, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/apply", GetApiHost(), planId, branch)

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return "", &shared.ApiError{Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	request, err := http.NewRequest(http.MethodPatch, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)
	if err != nil {
		return "", &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)

		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)
		if didRefresh {
			return a.ApplyPlan(planId, branch, req)
		}
		return "", apiErr
	}

	// Reading the body on success
	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &shared.ApiError{Msg: fmt.Sprintf("error reading response body: %v", err)}
	}

	return string(responseData), nil
}

func (a *Api) ArchivePlan(planId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/archive", GetApiHost(), planId)

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
		apiErr := HandleApiError(resp, errorBody)

		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)
		if didRefresh {
			return a.ArchivePlan(planId)
		}
		return apiErr
	}

	return nil
}

func (a *Api) UnarchivePlan(planId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/unarchive", GetApiHost(), planId)

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
		apiErr := HandleApiError(resp, errorBody)

		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)
		if didRefresh {
			return a.ArchivePlan(planId)
		}
		return apiErr
	}

	return nil
}

func (a *Api) RenamePlan(planId string, name string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/rename", GetApiHost(), planId)

	reqBytes, err := json.Marshal(shared.RenamePlanRequest{Name: name})
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	request, err := http.NewRequest(http.MethodPatch, serverUrl, bytes.NewBuffer(reqBytes))

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

		apiErr := HandleApiError(resp, errorBody)

		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)
		if didRefresh {
			return a.RenamePlan(planId, name)
		}
		return apiErr
	}

	return nil
}

func (a *Api) RejectAllChanges(planId, branch string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/reject_all", GetApiHost(), planId, branch)

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
		apiErr := HandleApiError(resp, errorBody)

		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)
		if didRefresh {
			return a.RejectAllChanges(planId, branch)
		}
		return apiErr
	}

	return nil
}

func (a *Api) RejectFile(planId, branch, filePath string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/reject_file", GetApiHost(), planId, branch)

	reqBytes, err := json.Marshal(shared.RejectFileRequest{FilePath: filePath})

	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	req, err := http.NewRequest(http.MethodPatch, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)
		if didRefresh {
			a.RejectFile(planId, branch, filePath)
		}
		return apiErr
	}

	return nil
}

func (a *Api) RejectFiles(planId, branch string, paths []string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/reject_files", GetApiHost(), planId, branch)

	reqBytes, err := json.Marshal(shared.RejectFilesRequest{Paths: paths})

	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	req, err := http.NewRequest(http.MethodPatch, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error creating request: %v", err)}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		didRefresh, apiErr := refreshAuthIfNeeded(apiErr)
		if didRefresh {
			a.RejectFiles(planId, branch, paths)
		}
		return apiErr
	}

	return nil
}

func (a *Api) LoadContext(planId, branch string, req shared.LoadContextRequest) (*shared.LoadContextResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/context", GetApiHost(), planId, branch)
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
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.LoadContext(planId, branch, req)
		}
		return nil, apiErr
	}

	var loadContextResponse shared.LoadContextResponse
	err = json.NewDecoder(resp.Body).Decode(&loadContextResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &loadContextResponse, nil
}

func (a *Api) UpdateContext(planId, branch string, req shared.UpdateContextRequest) (*shared.UpdateContextResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/context", GetApiHost(), planId, branch)

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
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.UpdateContext(planId, branch, req)
		}
		return nil, apiErr
	}

	var updateContextResponse shared.UpdateContextResponse
	err = json.NewDecoder(resp.Body).Decode(&updateContextResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &updateContextResponse, nil
}

func (a *Api) DeleteContext(planId, branch string, req shared.DeleteContextRequest) (*shared.DeleteContextResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/context", GetApiHost(), planId, branch)
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
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.DeleteContext(planId, branch, req)
		}
		return nil, apiErr
	}

	var deleteContextResponse shared.DeleteContextResponse
	err = json.NewDecoder(resp.Body).Decode(&deleteContextResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &deleteContextResponse, nil
}

func (a *Api) ListContext(planId, branch string) ([]*shared.Context, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/context", GetApiHost(), planId, branch)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListContext(planId, branch)
		}
		return nil, apiErr
	}

	var contexts []*shared.Context
	err = json.NewDecoder(resp.Body).Decode(&contexts)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return contexts, nil
}

func (a *Api) LoadCachedFileMap(planId, branch string, req shared.LoadCachedFileMapRequest) (*shared.LoadCachedFileMapResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/load_cached_file_map", GetApiHost(), planId, branch)
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
		apiErr := HandleApiError(resp, errorBody)
		return nil, apiErr
	}

	var loadResp shared.LoadCachedFileMapResponse
	err = json.NewDecoder(resp.Body).Decode(&loadResp)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &loadResp, nil
}

func (a *Api) ListConvo(planId, branch string) ([]*shared.ConvoMessage, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/convo", GetApiHost(), planId, branch)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListConvo(planId, branch)
		}
		return nil, apiErr
	}

	var convos []*shared.ConvoMessage
	err = json.NewDecoder(resp.Body).Decode(&convos)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return convos, nil
}

func (a *Api) GetPlanStatus(planId, branch string) (string, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/status", GetApiHost(), planId, branch)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return "", &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.GetPlanStatus(planId, branch)
		}
		return "", apiErr
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error reading response body: %v", err)}
	}

	return string(body), nil
}

func (a *Api) GetPlanDiffs(planId, branch string, plain bool) (string, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/diffs", GetApiHost(), planId, branch)

	if plain {
		serverUrl += "?plain=true"
	}

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return "", &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.GetPlanDiffs(planId, branch, plain)
		}
		return "", apiErr
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error reading response body: %v", err)}
	}

	return string(body), nil
}

func (a *Api) ListLogs(planId, branch string) (*shared.LogResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/logs", GetApiHost(), planId, branch)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListLogs(planId, branch)
		}
		return nil, apiErr
	}

	var logs shared.LogResponse
	err = json.NewDecoder(resp.Body).Decode(&logs)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &logs, nil
}

func (a *Api) RewindPlan(planId, branch string, req shared.RewindPlanRequest) (*shared.RewindPlanResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/rewind", GetApiHost(), planId, branch)
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
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.RewindPlan(planId, branch, req)
		}
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
	host := customHost
	if host == "" {
		host = CloudApiHost
	}
	serverUrl := host + "/accounts/sign_in"
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
		apiErr := HandleApiError(resp, errorBody)
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
	host := customHost
	if host == "" {
		host = CloudApiHost
	}
	serverUrl := host + "/accounts"
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
		apiErr := HandleApiError(resp, errorBody)
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
	serverUrl := GetApiHost() + "/orgs"
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
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.CreateOrg(req)
		}
		return nil, apiErr
	}

	var createOrgResponse shared.CreateOrgResponse
	err = json.NewDecoder(resp.Body).Decode(&createOrgResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &createOrgResponse, nil
}

func (a *Api) GetOrgSession() (*shared.Org, *shared.ApiError) {
	serverUrl := GetApiHost() + "/orgs/session"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.GetOrgSession()
		}
		return nil, apiErr
	}

	var org *shared.Org

	err = json.NewDecoder(resp.Body).Decode(&org)

	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return org, nil
}

func (a *Api) ListOrgs() ([]*shared.Org, *shared.ApiError) {
	serverUrl := GetApiHost() + "/orgs"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListOrgs()
		}
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
	serverUrl := fmt.Sprintf("%s/orgs/users/%s", GetApiHost(), userId)
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
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.DeleteUser(userId)
		}
		return apiErr
	}

	return nil
}

func (a *Api) ListOrgRoles() ([]*shared.OrgRole, *shared.ApiError) {
	serverUrl := GetApiHost() + "/orgs/roles"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %s", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)

		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListOrgRoles()
		}
		return nil, apiErr
	}

	var roles []*shared.OrgRole
	err = json.NewDecoder(resp.Body).Decode(&roles)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %s", err)}
	}

	return roles, nil
}

func (a *Api) InviteUser(req shared.InviteRequest) *shared.ApiError {
	serverUrl := GetApiHost() + "/invites"
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
		apiErr := HandleApiError(resp, errorBody)

		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.InviteUser(req)
		}
		return apiErr
	}

	return nil
}

func (a *Api) ListPendingInvites() ([]*shared.Invite, *shared.ApiError) {
	serverUrl := GetApiHost() + "/invites/pending"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListPendingInvites()
		}
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
	serverUrl := GetApiHost() + "/invites/accepted"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListAcceptedInvites()
		}
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
	serverUrl := GetApiHost() + "/invites/all"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListAllInvites()
		}
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
	serverUrl := fmt.Sprintf("%s/invites/%s", GetApiHost(), inviteId)
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
		apiErr := HandleApiError(resp, errorBody)

		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.DeleteInvite(inviteId)
		}
		return apiErr
	}

	return nil
}

func (a *Api) CreateEmailVerification(email, customHost, userId string) (*shared.CreateEmailVerificationResponse, *shared.ApiError) {
	host := customHost
	if host == "" {
		host = CloudApiHost
	}
	serverUrl := host + "/accounts/email_verifications"
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
		return nil, HandleApiError(resp, errorBody)
	}

	var verificationResponse shared.CreateEmailVerificationResponse
	err = json.NewDecoder(resp.Body).Decode(&verificationResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &verificationResponse, nil
}

func (a *Api) CreateSignInCode() (string, *shared.ApiError) {
	serverUrl := GetApiHost() + "/accounts/sign_in_codes"
	resp, err := authenticatedFastClient.Post(serverUrl, "application/json", nil)
	if err != nil {
		return "", &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.CreateSignInCode()
		}
		return "", apiErr
	}

	var signInCode string
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error reading response body: %v", err)}
	}
	signInCode = string(body)

	return signInCode, nil
}

func (a *Api) SignOut() *shared.ApiError {
	serverUrl := GetApiHost() + "/accounts/sign_out"

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
		return HandleApiError(resp, errorBody)
	}

	return nil
}

func (a *Api) ListUsers() (*shared.ListUsersResponse, *shared.ApiError) {
	serverUrl := GetApiHost() + "/users"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListUsers()
		}
		return nil, apiErr
	}

	var r *shared.ListUsersResponse
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return r, nil
}

func (a *Api) ListBranches(planId string) ([]*shared.Branch, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/branches", GetApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %s", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)

		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListBranches(planId)
		}
		return nil, apiErr
	}

	var branches []*shared.Branch
	err = json.NewDecoder(resp.Body).Decode(&branches)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %s", err)}
	}

	return branches, nil
}

func (a *Api) CreateBranch(planId, branch string, req shared.CreateBranchRequest) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/branches", GetApiHost(), planId, branch)

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %s", err)}
	}

	resp, err := authenticatedFastClient.Post(serverUrl, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %s", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)

		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.CreateBranch(planId, branch, req)
		}
		return apiErr
	}

	return nil
}

func (a *Api) DeleteBranch(planId, branch string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/branches/%s", GetApiHost(), planId, branch)

	req, err := http.NewRequest(http.MethodDelete, serverUrl, nil)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error creating request: %s", err)}
	}

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %s", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)

		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.DeleteBranch(planId, branch)
		}
		return apiErr
	}

	return nil
}

func (a *Api) GetSettings(planId, branch string) (*shared.PlanSettings, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/settings", GetApiHost(), planId, branch)

	resp, err := authenticatedFastClient.Get(serverUrl)

	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %s", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.GetSettings(planId, branch)
		}
		return nil, apiErr
	}

	var settings shared.PlanSettings
	err = json.NewDecoder(resp.Body).Decode(&settings)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %s", err)}
	}

	return &settings, nil
}

func (a *Api) UpdateSettings(planId, branch string, req shared.UpdateSettingsRequest) (*shared.UpdateSettingsResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/settings", GetApiHost(), planId, branch)

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %s", err)}
	}

	// log.Println("UpdateSettings", string(reqBytes))

	request, err := http.NewRequest(http.MethodPut, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error creating request: %s", err)}
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %s", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)

		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.UpdateSettings(planId, branch, req)
		}
		return nil, apiErr
	}

	var updateRes shared.UpdateSettingsResponse
	err = json.NewDecoder(resp.Body).Decode(&updateRes)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %s", err)}
	}

	return &updateRes, nil

}

func (a *Api) GetOrgDefaultSettings() (*shared.PlanSettings, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/default_settings", GetApiHost())

	resp, err := authenticatedFastClient.Get(serverUrl)

	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %s", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.GetOrgDefaultSettings()
		}
		return nil, apiErr
	}

	var settings shared.PlanSettings
	err = json.NewDecoder(resp.Body).Decode(&settings)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %s", err)}
	}

	return &settings, nil
}

func (a *Api) UpdateOrgDefaultSettings(req shared.UpdateSettingsRequest) (*shared.UpdateSettingsResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/default_settings", GetApiHost())

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %s", err)}
	}

	// log.Println("UpdateSettings", string(reqBytes))

	request, err := http.NewRequest(http.MethodPut, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error creating request: %s", err)}
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %s", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)

		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.UpdateOrgDefaultSettings(req)
		}
		return nil, apiErr
	}

	var updateRes shared.UpdateSettingsResponse
	err = json.NewDecoder(resp.Body).Decode(&updateRes)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %s", err)}
	}

	return &updateRes, nil
}

func (a *Api) GetPlanConfig(planId string) (*shared.PlanConfig, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/config", GetApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.GetPlanConfig(planId)
		}
		return nil, apiErr
	}

	var res shared.GetPlanConfigResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return res.Config, nil
}

func (a *Api) UpdatePlanConfig(planId string, req shared.UpdatePlanConfigRequest) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/config", GetApiHost(), planId)

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	request, err := http.NewRequest(http.MethodPut, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error creating request: %v", err)}
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.UpdatePlanConfig(planId, req)
		}
		return apiErr
	}

	return nil
}

func (a *Api) GetDefaultPlanConfig() (*shared.PlanConfig, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/default_plan_config", GetApiHost())

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.GetDefaultPlanConfig()
		}
		return nil, apiErr
	}

	var res shared.GetDefaultPlanConfigResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return res.Config, nil
}

func (a *Api) UpdateDefaultPlanConfig(req shared.UpdateDefaultPlanConfigRequest) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/default_plan_config", GetApiHost())

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	request, err := http.NewRequest(http.MethodPut, serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error creating request: %v", err)}
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(request)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.UpdateDefaultPlanConfig(req)
		}
		return apiErr
	}

	return nil
}

func (a *Api) CreateCustomModel(model *shared.AvailableModel) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/custom_models", GetApiHost())
	body, err := json.Marshal(model)
	if err != nil {
		return &shared.ApiError{Msg: "Failed to marshal model"}
	}

	resp, err := authenticatedFastClient.Post(serverUrl, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)

		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.CreateCustomModel(model)
		}
		return apiErr
	}

	return nil
}

func (a *Api) ListCustomModels() ([]*shared.AvailableModel, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/custom_models", GetApiHost())
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)

		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListCustomModels()
		}
		return nil, apiErr
	}

	var models []*shared.AvailableModel
	err = json.NewDecoder(resp.Body).Decode(&models)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return models, nil
}

func (a *Api) DeleteAvailableModel(modelId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/custom_models/%s", GetApiHost(), modelId)
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

		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.DeleteAvailableModel(modelId)
		}
		return apiErr
	}

	return nil
}

func (a *Api) UpdateCustomModel(model *shared.AvailableModel) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/custom_models/%s", GetApiHost(), model.Id)
	body, err := json.Marshal(model)
	if err != nil {
		return &shared.ApiError{Msg: "Failed to marshal model"}
	}

	req, err := http.NewRequest(http.MethodPut, serverUrl, bytes.NewBuffer(body))
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error creating request: %v", err)}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.UpdateCustomModel(model)
		}
		return apiErr
	}

	return nil
}

func (a *Api) CreateModelPack(set *shared.ModelPack) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/model_sets", GetApiHost())
	body, err := json.Marshal(set)
	if err != nil {
		return &shared.ApiError{Msg: "Failed to marshal model pack"}
	}

	resp, err := authenticatedFastClient.Post(serverUrl, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)

		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.CreateModelPack(set)
		}
		return apiErr
	}

	return nil

}

func (a *Api) ListModelPacks() ([]*shared.ModelPack, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/model_sets", GetApiHost())

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)

		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.ListModelPacks()
		}
		return nil, apiErr
	}

	var sets []*shared.ModelPack
	err = json.NewDecoder(resp.Body).Decode(&sets)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return sets, nil

}

func (a *Api) DeleteModelPack(setId string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/model_sets/%s", GetApiHost(), setId)

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

		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.DeleteModelPack(setId)
		}
		return apiErr
	}

	return nil
}

func (a *Api) UpdateModelPack(set *shared.ModelPack) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/model_sets/%s", GetApiHost(), set.Id)
	body, err := json.Marshal(set)
	if err != nil {
		return &shared.ApiError{Msg: "Failed to marshal model pack"}
	}

	req, err := http.NewRequest(http.MethodPut, serverUrl, bytes.NewBuffer(body))
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := authenticatedFastClient.Do(req)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.UpdateModelPack(set)
		}
		return apiErr
	}

	return nil
}

func (a *Api) GetCreditsTransactions(pageSize, pageNum int, req shared.CreditsLogRequest) (*shared.CreditsLogResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/billing/credits_transactions?size=%d&page=%d", GetApiHost(), pageSize, pageNum)

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

		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.GetCreditsTransactions(pageSize, pageNum, req)
		}
		return nil, apiErr
	}

	var res *shared.CreditsLogResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return res, nil
}

func (a *Api) GetCreditsSummary(req shared.CreditsLogRequest) (*shared.CreditsSummaryResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/billing/credits_summary", GetApiHost())

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
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.GetCreditsSummary(req)
		}
		return nil, apiErr
	}

	var res *shared.CreditsSummaryResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return res, nil
}

func (a *Api) GetBalance() (decimal.Decimal, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/billing/balance", GetApiHost())

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return decimal.Zero, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.GetBalance()
		}
		return decimal.Zero, apiErr
	}

	var res *shared.GetBalanceResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return decimal.Zero, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return res.Balance, nil
}

func (a *Api) GetFileMap(req shared.GetFileMapRequest) (*shared.GetFileMapResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/file_map", GetApiHost())
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	resp, err := authenticatedSlowClient.Post(serverUrl, "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.GetFileMap(req)
		}
		return nil, apiErr
	}

	var respBody shared.GetFileMapResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &respBody, nil
}

func (a *Api) GetContextBody(planId, branch, contextId string) (*shared.GetContextBodyResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/context/%s/body", GetApiHost(), planId, branch, contextId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.GetContextBody(planId, branch, contextId)
		}
		return nil, apiErr
	}

	var respBody shared.GetContextBodyResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &respBody, nil
}

func (a *Api) AutoLoadContext(ctx context.Context, planId, branch string, req shared.LoadContextRequest) (*shared.LoadContextResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/auto_load_context", GetApiHost(), planId, branch)
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error marshalling request: %v", err)}
	}

	// Create a new request with context
	httpReq, err := http.NewRequestWithContext(ctx, "POST", serverUrl, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error creating request: %v", err)}
	}

	// Set the content type header
	httpReq.Header.Set("Content-Type", "application/json")

	// Use the slow client since we may be uploading relatively large files
	resp, err := authenticatedSlowClient.Do(httpReq)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.LoadContext(planId, branch, req)
		}
		return nil, apiErr
	}

	var loadContextResponse shared.LoadContextResponse
	err = json.NewDecoder(resp.Body).Decode(&loadContextResponse)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &loadContextResponse, nil
}

func (a *Api) GetBuildStatus(planId, branch string) (*shared.GetBuildStatusResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/build_status", GetApiHost(), planId, branch)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := HandleApiError(resp, errorBody)
		authRefreshed, apiErr := refreshAuthIfNeeded(apiErr)
		if authRefreshed {
			return a.GetBuildStatus(planId, branch)
		}
		return nil, apiErr
	}

	var respBody shared.GetBuildStatusResponse
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error decoding response: %v", err)}
	}

	return &respBody, nil
}
