package lib

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"plandex/types"
	"strings"

	"github.com/plandex/plandex/shared"
	openai "github.com/sashabaranov/go-openai"
)

const apiHost = "http://localhost:8088"

func ApiPropose(prompt string, chatOnly bool, onStream types.OnStreamProposal) error {
	serverUrl := apiHost + "/proposal"

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
		return err
	}

	// Using select to receive from either data or error channel for conversation
	select {
	case conversation = <-conversationChan:
	case err = <-conversationErrChan:
		return err
	}

	// Using select to receive from either data or error channel for plan
	select {
	case plan := <-planChan:
		currentPlan = plan
	case err = <-planErrChan:
		return err
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
		return err
	}

	resp, err := http.Post(serverUrl, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	// Check the HTTP status code
	if resp.StatusCode >= 400 {
		// Read the error message from the body
		errorBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned an error %d: %s", resp.StatusCode, string(errorBody))
	}

	reader := bufio.NewReader(resp.Body)
	buf := make([]byte, 32)

	go func() {
		for {
			n, err := reader.Read(buf)
			if err == io.EOF {
				fmt.Println("EOF")
				onStream("", true, nil)
				resp.Body.Close()
				return
			}
			if err != nil {
				fmt.Println("Error reading line:", err)
				onStream("", true, err)
				resp.Body.Close()
				return
			}

			s := string(buf[:n])

			if strings.Contains(s, shared.STREAM_FINISHED) {
				onStream("", true, nil)
				resp.Body.Close()
				return
			}

			onStream(s, false, nil)
		}
	}()

	return nil
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
