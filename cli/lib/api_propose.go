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

func (api *API) Propose(prompt, parentProposalId, rootId string, onStream types.OnStreamPlan) (*shared.PromptRequest, error) {
	serverUrl := apiHost + "/proposal"

	// Channels to receive data and errors
	contextChan := make(chan shared.ModelContext, 1) // Buffered channels to prevent deadlock
	contextErrChan := make(chan error, 1)

	conversationChan := make(chan []shared.ConversationMessage, 1)
	conversationErrChan := make(chan error, 1)

	planChan := make(chan shared.CurrentPlanFiles, 1)
	planErrChan := make(chan error, 1)

	summaryChan := make(chan []shared.ConversationSummary, 1)
	summaryErrChan := make(chan error, 1)

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
		conversation, err := LoadConversation()
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

	// Goroutine for loading summaries
	go func() {
		summaries, err := LoadSummaries()
		if err != nil {
			fmt.Println("Error loading summaries")
			summaryErrChan <- err
			return
		}
		summaryChan <- summaries
	}()

	var modelContext shared.ModelContext
	var currentPlan shared.CurrentPlanFiles
	var conversation []shared.ConversationMessage
	var summaries []shared.ConversationSummary
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

	// Using select to receive from either data or error channel for summaries
	select {
	case summaries = <-summaryChan:
	case err = <-summaryErrChan:
		return nil, err
	}

	payload := shared.PromptRequest{
		Prompt:                prompt,
		ModelContext:          modelContext,
		Conversation:          conversation,
		ConversationSummaries: summaries,
		CurrentPlan:           currentPlan,
		ParentProposalId:      parentProposalId,
		RootProposalId:        rootId,
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
