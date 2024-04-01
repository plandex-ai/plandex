package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"plandex/types"
	"strings"

	"github.com/plandex/plandex/shared"
)

func (a *Api) StartTrial() (*shared.StartTrialResponse, *shared.ApiError) {
	serverUrl := cloudApiHost + "/accounts/start_trial"

	log.Println("Sending request to", serverUrl)

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
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
	serverUrl := getApiHost() + "/projects"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
		apiErr := handleApiError(resp, errorBody)
		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)
		if didRefresh {
			return a.SetProjectPlan(projectId, req)
		}
		return apiErr
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
		apiErr := handleApiError(resp, errorBody)

		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)
		if didRefresh {
			return a.RenameProject(projectId, req)
		}
		return apiErr
	}

	return nil
}
func (a *Api) ListPlans(projectIds []string) ([]*shared.Plan, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans?", getApiHost())
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

		apiErr := handleApiError(resp, errorBody)

		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)
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
	serverUrl := fmt.Sprintf("%s/plans/archive?", getApiHost())
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
		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
	serverUrl := fmt.Sprintf("%s/plans/ps?", getApiHost())
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
		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
	serverUrl := fmt.Sprintf("%s/projects/%s/plans/current_branches", getApiHost(), projectId)

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

		apiErr := handleApiError(resp, errorBody)

		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)
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
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
	serverUrl := fmt.Sprintf("%s/plans/%s", getApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
		apiErr := handleApiError(resp, errorBody)

		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)
		if didRefresh {
			return a.DeletePlan(planId)
		}
		return apiErr
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
		apiErr := handleApiError(resp, errorBody)

		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)

		if didRefresh {
			return a.DeleteAllPlans(projectId)
		}
		return apiErr
	}

	return nil
}

func (a *Api) TellPlan(planId, branch string, req shared.TellPlanRequest, onStream types.OnStreamPlan) *shared.ApiError {

	serverUrl := fmt.Sprintf("%s/plans/%s/%s/tell", getApiHost(), planId, branch)
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
		apiErr := handleApiError(resp, errorBody)

		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)

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

	serverUrl := fmt.Sprintf("%s/plans/%s/%s/build", getApiHost(), planId, branch)
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
		apiErr := handleApiError(resp, errorBody)

		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)

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
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/respond_missing_file", getApiHost(), planId, branch)

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
		apiErr := handleApiError(resp, errorBody)

		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)

		if didRefresh {
			return a.RespondMissingFile(planId, branch, req)
		}
		return apiErr
	}

	return nil

}

func (a *Api) ConnectPlan(planId, branch string, onStream types.OnStreamPlan) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/connect", getApiHost(), planId, branch)

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
		apiErr := handleApiError(resp, errorBody)

		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)

		if didRefresh {
			return a.ConnectPlan(planId, branch, onStream)
		}

		return apiErr
	}

	connectPlanRespStream(resp.Body, onStream)

	return nil
}

func (a *Api) StopPlan(planId, branch string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/stop", getApiHost(), planId, branch)

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
		apiErr := handleApiError(resp, errorBody)
		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)
		if didRefresh {
			return a.StopPlan(planId, branch)
		}
		return apiErr
	}

	return nil
}

func (a *Api) GetCurrentPlanState(planId, branch string) (*shared.CurrentPlanState, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/current_plan", getApiHost(), planId, branch)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
			return a.GetCurrentPlanState(planId, branch)
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

func (a *Api) ApplyPlan(planId, branch string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/apply", getApiHost(), planId, branch)

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
		apiErr := handleApiError(resp, errorBody)

		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)
		if didRefresh {
			return a.ApplyPlan(planId, branch)
		}
		return apiErr
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
		apiErr := handleApiError(resp, errorBody)

		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)
		if didRefresh {
			return a.ArchivePlan(planId)
		}
		return apiErr
	}

	return nil
}

func (a *Api) RejectAllChanges(planId, branch string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/reject_all", getApiHost(), planId, branch)

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
		apiErr := handleApiError(resp, errorBody)

		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)
		if didRefresh {
			return a.RejectAllChanges(planId, branch)
		}
		return apiErr
	}

	return nil
}

func (a *Api) RejectFile(planId, branch, filePath string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/reject_file", getApiHost(), planId, branch)

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
		apiErr := handleApiError(resp, errorBody)
		didRefresh, apiErr := refreshTokenIfNeeded(apiErr)
		if didRefresh {
			a.RejectFile(planId, branch, filePath)
		}
		return apiErr
	}

	return nil
}

func (a *Api) LoadContext(planId, branch string, req shared.LoadContextRequest) (*shared.LoadContextResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/context", getApiHost(), planId, branch)
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
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/context", getApiHost(), planId, branch)

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
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/context", getApiHost(), planId, branch)
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
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/context", getApiHost(), planId, branch)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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

func (a *Api) ListConvo(planId, branch string) ([]*shared.ConvoMessage, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/convo", getApiHost(), planId, branch)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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

func (a *Api) ListLogs(planId, branch string) (*shared.LogResponse, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/logs", getApiHost(), planId, branch)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/rewind", getApiHost(), planId, branch)
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
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
		host = cloudApiHost
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
	host := customHost
	if host == "" {
		host = cloudApiHost
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
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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

func (a *Api) GetOrgSession() *shared.ApiError {
	serverUrl := getApiHost() + "/orgs/session"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		return apiErr
	}

	return nil
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
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
			return a.DeleteUser(userId)
		}
		return apiErr
	}

	return nil
}

func (a *Api) ListOrgRoles() ([]*shared.OrgRole, *shared.ApiError) {
	serverUrl := getApiHost() + "/orgs/roles"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %s", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)

		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
		apiErr := handleApiError(resp, errorBody)

		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
			return a.InviteUser(req)
		}
		return apiErr
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
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
	serverUrl := getApiHost() + "/invites/accepted"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
	serverUrl := getApiHost() + "/invites/all"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
		apiErr := handleApiError(resp, errorBody)

		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
			return a.DeleteInvite(inviteId)
		}
		return apiErr
	}

	return nil
}

func (a *Api) CreateEmailVerification(email, customHost, userId string) (*shared.CreateEmailVerificationResponse, *shared.ApiError) {
	host := customHost
	if host == "" {
		host = cloudApiHost
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
		return nil, handleApiError(resp, errorBody)
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

func (a *Api) ListUsers() (*shared.ListUsersResponse, *shared.ApiError) {
	serverUrl := getApiHost() + "/users"
	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
	serverUrl := fmt.Sprintf("%s/plans/%s/branches", getApiHost(), planId)

	resp, err := authenticatedFastClient.Get(serverUrl)
	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %s", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)

		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/branches", getApiHost(), planId, branch)

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

		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
			return a.CreateBranch(planId, branch, req)
		}
		return apiErr
	}

	return nil
}

func (a *Api) DeleteBranch(planId, branch string) *shared.ApiError {
	serverUrl := fmt.Sprintf("%s/plans/%s/branches/%s", getApiHost(), planId, branch)

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

		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
			return a.DeleteBranch(planId, branch)
		}
		return apiErr
	}

	return nil
}

func (a *Api) GetSettings(planId, branch string) (*shared.PlanSettings, *shared.ApiError) {
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/settings", getApiHost(), planId, branch)

	resp, err := authenticatedFastClient.Get(serverUrl)

	if err != nil {
		return nil, &shared.ApiError{Type: shared.ApiErrorTypeOther, Msg: fmt.Sprintf("error sending request: %s", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errorBody, _ := io.ReadAll(resp.Body)
		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
	serverUrl := fmt.Sprintf("%s/plans/%s/%s/settings", getApiHost(), planId, branch)

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

		apiErr := handleApiError(resp, errorBody)
		tokenRefreshed, apiErr := refreshTokenIfNeeded(apiErr)
		if tokenRefreshed {
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
