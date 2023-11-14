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

	errCh := make(chan error, 1)
	contextCh := make(chan shared.ModelContext, 1)
	conversationCh := make(chan []*shared.ConversationMessage, 1)
	planFilesCh := make(chan *shared.CurrentPlanFiles, 1)
	summaryCh := make(chan []*shared.ConversationSummary, 1)

	// Goroutine for loading conversation
	go func() {
		conversation, err := LoadConversation()
		if err != nil {
			fmt.Println("Error loading conversation")
			errCh <- err
			return
		}
		conversationCh <- conversation
	}()

	// Goroutine for loading plan files and context
	go func() {
		planFiles, _, modelContext, err := GetCurrentPlanStateWithContext()
		if err != nil {
			fmt.Println("Error loading plan")
			errCh <- err
			return
		}
		planFilesCh <- planFiles
		contextCh <- modelContext
	}()

	// Goroutine for loading summaries
	go func() {
		summaries, err := LoadSummaries()
		if err != nil {
			fmt.Println("Error loading summaries")
			errCh <- err
			return
		}
		summaryCh <- summaries
	}()

	var modelContext shared.ModelContext
	var currentPlanFiles *shared.CurrentPlanFiles
	var conversation []*shared.ConversationMessage
	var summaries []*shared.ConversationSummary

	for i := 0; i < 4; i++ {
		select {
		case err := <-errCh:
			return nil, fmt.Errorf("error loading plan data: %v", err)
		case modelContext = <-contextCh:
		case conversation = <-conversationCh:
		case currentPlanFiles = <-planFilesCh:
		case summaries = <-summaryCh:
		}
	}

	payload := shared.PromptRequest{
		Prompt:                prompt,
		ModelContext:          modelContext,
		Conversation:          conversation,
		ConversationSummaries: summaries,
		CurrentPlan:           currentPlanFiles,
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
			} else if s == shared.STREAM_WRITE_PHASE {
				err = streamState.Event(context.Background(), shared.EVENT_WRITE)
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
