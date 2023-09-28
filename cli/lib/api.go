package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/plandex/plandex/shared"
	openai "github.com/sashabaranov/go-openai"
)

const apiHost = "http://localhost:8088"

func ApiPrompt(prompt string, chatOnly bool) (*shared.PromptResponse, error) {
	serverUrl := apiHost + "/prompt"

	// Channels to receive data and errors
	contextChan := make(chan shared.ModelContext, 1) // Buffered channels to prevent deadlock
	contextErrChan := make(chan error, 1)

	conversationChan := make(chan []openai.ChatCompletionMessage, 1)
	conversationErrChan := make(chan error, 1)

	planChan := make(chan shared.CurrentPlanFiles, 1)
	planErrChan := make(chan error, 1)

	// Goroutine for loading context
	go func() {
		context, err := GetAllContext(false)
		if err != nil {
			fmt.Println("Error loading context:" + err.Error())
			contextErrChan <- err
			return
		}
		contextChan <- context
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

	var context shared.ModelContext
	var currentPlan shared.CurrentPlanFiles
	var conversation []openai.ChatCompletionMessage
	var err error

	// Using select to receive from either data or error channel for context
	select {
	case context = <-contextChan:
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
		Prompt:       prompt,
		ModelContext: context,
		Conversation: conversation,
		CurrentPlan:  currentPlan,
		ChatOnly:     chatOnly,
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

	// Check the HTTP status code
	if resp.StatusCode >= 400 {
		// Read the error message from the body
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned an error %d: %s", resp.StatusCode, string(errorBody))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed shared.PromptResponse
	err = json.Unmarshal(body, &parsed)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func ApiSummarize(text string) (*shared.SummarizeResponse, error) {
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

	fmt.Println("ApiSummarize response body:")
	fmt.Println(string(body))

	var summarized shared.SummarizeResponse
	err = json.Unmarshal(body, &summarized)
	if err != nil {
		return nil, err
	}

	return &summarized, nil
}
