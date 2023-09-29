package proposal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"plandex-server/model"
	"plandex-server/types"

	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

// map to store the proposals
var (
	proposalsMap = make(map[string]*types.Proposal)
	mu           sync.Mutex
)

// Proposal function to create a new proposal
func CreateProposal(req shared.PromptRequest, onStream types.OnStreamProposalFunc) (*context.CancelFunc, error) {
	contextText := model.FormatModelContext(req.ModelContext)

	systemMessageText := `
		You are Plandex, an AI programming and system administration assistant.
		You help programmers with tasks, especially those that involve multiple files and shell commands. You offer a structured, versioned, and iterative approach to AI-driven development. 
		You and the programmer collaborate to create a 'plan' for the task at hand. A plan is a set of files and an 'exec' script with an attached context.
		Based on user-provided context, please create a plan for the task. When suggesting changes that would modify files from the context or create new files, always precede them with the file path.
		Context from the user:` + contextText

	systemMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemMessageText,
	}

	messages := []openai.ChatCompletionMessage{
		systemMessage,
	}

	if len(req.Conversation) > 0 {
		messages = append(messages, req.Conversation...)
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: req.Prompt,
	})

	for _, message := range messages {
		fmt.Printf("%s: %s\n", message.Role, message.Content)
	}

	proposalId, err := uuid.NewRandom()
	if err != nil {
		fmt.Printf("Failed to generate proposal id: %v\n", err)
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	// store the proposal
	mu.Lock()
	proposalsMap[proposalId.String()] = &types.Proposal{
		Id:           proposalId.String(),
		Cancel:       &cancel,
		ModelContext: &req.ModelContext,
		Content:      "",
	}
	mu.Unlock()

	modelReq := openai.ChatCompletionRequest{
		Model:    openai.GPT4,
		Messages: messages,
		Stream:   true,
	}

	stream, err := model.Client.CreateChatCompletionStream(ctx, modelReq)
	if err != nil {
		fmt.Printf("Error creating proposal GPT4 stream: %v\n", err)
		mu.Lock()
		delete(proposalsMap, proposalId.String()) // Remove the proposal from the map
		mu.Unlock()
		return nil, err
	}

	onFinish := func() {
		mu.Lock()
		proposal := proposalsMap[proposalId.String()]
		proposal.FinishedProposal = true
		proposalsMap[proposalId.String()] = proposal
		mu.Unlock()
		onStream("", true, nil)
	}

	onError := func(err error) {
		fmt.Printf("\nStream error: %v\n", err)
		mu.Lock()
		proposal := proposalsMap[proposalId.String()]
		proposal.ProposalError = err
		proposalsMap[proposalId.String()] = proposal
		mu.Unlock()
		onStream("", true, err)
	}

	go func() {
		defer stream.Close()

		// Create a timer that will trigger if no chunk is received within the specified duration
		timer := time.NewTimer(model.OPENAI_STREAM_CHUNK_TIMEOUT)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				// The main context was canceled (not the timer)
				return
			case <-timer.C:
				// Timer triggered because no new chunk was received in time
				fmt.Println("\nStream timeout due to inactivity")
				onError(fmt.Errorf("stream timeout due to inactivity"))
				return
			default:
				response, err := stream.Recv()

				if err == nil {
					// Successfully received a chunk, reset the timer
					if !timer.Stop() {
						<-timer.C
					}
					timer.Reset(model.OPENAI_STREAM_CHUNK_TIMEOUT)
				}

				if errors.Is(err, io.EOF) {
					fmt.Println("\nStream finished")
					onFinish()
					return
				}

				if err != nil {
					onError(err)
					return
				}

				if len(response.Choices) == 0 {
					fmt.Println("\nStream finished")
					onFinish()
					return
				}

				// TODO handle different finish reasons
				if response.Choices[0].FinishReason != "" {
					fmt.Println("\nStream finished")
					onFinish()
				}

				content := response.Choices[0].Delta.Content

				// add to the proposal
				mu.Lock()
				proposal := proposalsMap[proposalId.String()]
				proposal.Content += content
				proposalsMap[proposalId.String()] = proposal
				mu.Unlock()

				fmt.Printf("%s", content)
				onStream(content, false, nil)
			}
		}
	}()

	return &cancel, nil
}
