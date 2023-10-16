package lib

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"plandex/types"

	"github.com/plandex/plandex/shared"
)

const apiHost = "http://localhost:8088"

// Check that API implements the types.APIHandler interface
var Api types.APIHandler = (*API)(nil)

type API struct{}

func (api *API) Propose(prompt, parentProposalId string, onStream types.OnStreamPlan) (*shared.PromptRequest, error) {
	serverUrl := apiHost + "/proposal"

	// Channels to receive data and errors
	contextChan := make(chan shared.ModelContext, 1) // Buffered channels to prevent deadlock
	contextErrChan := make(chan error, 1)

	conversationChan := make(chan []shared.ConversationMessage, 1)
	conversationErrChan := make(chan error, 1)

	planChan := make(chan shared.CurrentPlanFiles, 1)
	planErrChan := make(chan error, 1)

	// Goroutine for loading context
	go func() {
		modelContext, err := GetAllContext(false)
		if err != nil {
			fmt.Println("Error loading context:" + err.Error())
			contextErrChan <- err
			return
		}
		contextChan <- modelContext
	}()

	// Goroutine for loading conversation
	go func() {
		conversation, err := loadConversation()
		if err != nil {
			fmt.Println("Error loading conversation")
			conversationErrChan <- err
			return
		}
		conversationChan <- conversation
	}()

	// Goroutine for loading plan
	go func() {
		plan, err := getCurrentPlanFiles()
		if err != nil {
			fmt.Println("Error loading plan")
			planErrChan <- err
			return
		}
		planChan <- plan
	}()

	var modelContext shared.ModelContext
	var currentPlan shared.CurrentPlanFiles
	var conversation []shared.ConversationMessage
	var err error

	// Using select to receive from either data or error channel for context
	select {
	case modelContext = <-contextChan:
	case err = <-contextErrChan:
		return nil, err
	}

	// Using select to receive from either data or error channel for conversation
	select {
	case conversation = <-conversationChan:
	case err = <-conversationErrChan:
		return nil, err
	}

	// Using select to receive from either data or error channel for plan
	select {
	case plan := <-planChan:
		currentPlan = plan
	case err = <-planErrChan:
		return nil, err
	}

	payload := shared.PromptRequest{
		Prompt:           prompt,
		ModelContext:     modelContext,
		Conversation:     conversation,
		CurrentPlan:      currentPlan,
		ParentProposalId: parentProposalId,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(serverUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	// Check the HTTP status code
	if resp.StatusCode >= 400 {
		// Read the error message from the body
		errorBody, _ := io.ReadAll(resp.Body)
		return &payload, fmt.Errorf("server returned an error %d: %s", resp.StatusCode, string(errorBody))
	}

	reader := bufio.NewReader(resp.Body)
	streamState := shared.NewPlanStreamState()

	go func() {
		for {
			s, err := readUntilSeparator(reader, shared.STREAM_MESSAGE_SEPARATOR)
			if err != nil {
				fmt.Println("Error reading line:", err)
				streamState.Event(context.Background(), shared.EVENT_ERROR)
				onStream(types.OnStreamPlanParams{Content: "", State: streamState, Err: err})
				resp.Body.Close()
				return
			}

			if s == shared.STREAM_FINISHED || s == shared.STREAM_ABORTED {
				var evt string
				if s == shared.STREAM_FINISHED {
					evt = shared.EVENT_FINISH
				} else {
					evt = shared.STATE_ABORTED
				}
				err := streamState.Event(context.Background(), evt)
				if err != nil {
					fmt.Printf("Error triggering state change %s: %s\n", evt, err)
				}
				onStream(types.OnStreamPlanParams{Content: "", State: streamState, Err: err})
				resp.Body.Close()
				return
			}

			if s == shared.STREAM_DESCRIPTION_PHASE {
				err = streamState.Event(context.Background(), shared.EVENT_DESCRIBE)
			} else if s == shared.STREAM_BUILD_PHASE {
				err = streamState.Event(context.Background(), shared.EVENT_BUILD)
			}

			if err != nil {
				fmt.Println("Error setting state:", err)
				onStream(types.OnStreamPlanParams{Content: "", State: streamState, Err: err})
				resp.Body.Close()
				return
			}

			onStream(types.OnStreamPlanParams{Content: s, State: streamState, Err: nil})
		}
	}()

	return &payload, nil
}

func (api *API) Abort(proposalId string) error {
	serverUrl := apiHost + "/abort"

	req, err := http.NewRequest("GET", serverUrl, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %s\n", err)
	}

	q := req.URL.Query()
	q.Add("proposalId", proposalId)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to server: %s\n", err)
	}

	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		// Read the error message from the body
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned an error %d: %s", resp.StatusCode,
			string(errorBody))
	}

	return nil
}

func (api *API) Summarize(text string) (*shared.SummarizeResponse, error) {
	serverUrl := apiHost + "/summarize"

	payload := shared.SummarizeRequest{
		Text: text,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(serverUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// fmt.Println("Summarize response body:")
	// fmt.Println(string(body))

	var summarized shared.SummarizeResponse
	err = json.Unmarshal(body, &summarized)
	if err != nil {
		return nil, err
	}

	return &summarized, nil
}

func (api *API) Sectionize(text string) (*shared.SectionizeResponse, error) {
	serverUrl := apiHost + "/sectionize"

	payload := shared.SectionizeRequest{
		Text: text,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(serverUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var sectionized shared.SectionizeResponse
	err = json.Unmarshal(rawBody, &sectionized)
	if err != nil {
		fmt.Printf("Failed JSON: %s\n", rawBody) // Printing the JSON causing the failure
		return nil, err
	}

	return &sectionized, nil
}

func readUntilSeparator(reader *bufio.Reader, separator string) (string, error) {
	var result []byte
	sepBytes := []byte(separator)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return string(result), err
		}
		result = append(result, b)
		if len(result) >= len(sepBytes) && bytes.HasSuffix(result, sepBytes) {
			return string(result[:len(result)-len(separator)]), nil
		}
	}
}
