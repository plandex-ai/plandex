package proposal

import (
	"context"
	"fmt"
	"os"
	"time"

	"plandex-server/model"
	"plandex-server/types"

	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

const systemMessageHead = `
		You are Plandex, an AI programming and system administration assistant. You offer a structured, versioned, and iterative approach to AI-driven development. 
		
		You and the programmer collaborate to create a 'plan' for the task at hand. A plan is a set of files with an attached context.

		Based on user-provided context, create a plan for the task using the following steps:

		1. Decide whether you've been given enough information and context to make a good plan. 
			a. If not:
		    - Explicitly say "I need more information or context to make a plan for this task."
			  - Ask the user for more information or context and stop there.

		2. Decide whether this task is small enough to be completed in a single response.
			a. If so, write out the code to complete the task. Precede the code with the file path like this '- file_path:'.
			b. If not: 
			  - Explicitly say "I will break this large task into subtasks."
				- Divide the task into smaller subtasks and list them in a numbered list. Stop there.
		
		Always precede code blocks them with the file path like this '- file_path:'. 
		
		Code should always be inside a file block labelled with the path. You can have multiple blocks labelled with the same file path. Every file you reference should either exist in the context directly or be a new file that will be created in the same base directory as the context file paths.

		For code in markdown blocks, always include the language name after the opening triple backticks.
		
		Don't include unnecessary comments in code. Only add comments if they make the code significantly more readable. Make comments concise.		

		At the end of a plan, you can suggest additional iterations to make the plan better. You can also ask the user to load more files or information into context if it would help you make a better plan.
		
		Context from the user:`

// Proposal function to create a new proposal
func CreateProposal(req shared.PromptRequest, onStream types.OnStreamFunc) error {
	goEnv := os.Getenv("GOENV") // Fetch the GO_ENV environment variable

	fmt.Println("GOENV: " + goEnv)

	if goEnv == "test" {
		streamLoremIpsum(onStream)
		return nil
	}

	contextText := model.FormatModelContext(req.ModelContext)
	systemMessageText := systemMessageHead + contextText
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

	// for _, message := range messages {
	// 	fmt.Printf("%s: %s\n", message.Role, message.Content)
	// }

	proposalUUID, err := uuid.NewRandom()
	if err != nil {
		fmt.Printf("Failed to generate proposal id: %v\n", err)
		return err
	}
	proposalId := proposalUUID.String()

	onStream(proposalId, nil)

	ctx, cancel := context.WithCancel(context.Background())

	// store the proposal
	proposals.Set(proposalId, &types.Proposal{
		Id:           proposalId,
		ModelContext: &req.ModelContext,
		Content:      "",
		ProposalStage: types.ProposalStage{
			CancelFn: &cancel,
		},
	})

	modelReq := openai.ChatCompletionRequest{
		Model:    openai.GPT4,
		Messages: messages,
		Stream:   true,
	}

	stream, err := model.Client.CreateChatCompletionStream(ctx, modelReq)
	if err != nil {
		fmt.Printf("Error creating proposal GPT4 stream: %v\n", err)
		proposals.Delete(proposalId)
		return err
	}

	onError := func(err error) {
		fmt.Printf("\nStream error: %v\n", err)

		proposals.Update(proposalId, func(proposal *types.Proposal) {
			proposal.SetErr(err)
		})
		onStream("", err)
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

				if err != nil {
					onError(fmt.Errorf("stream error: %v", err))
					return
				}

				if len(response.Choices) == 0 {
					onError(fmt.Errorf("stream finished with no choices"))
					return
				}

				if len(response.Choices) > 1 {
					onError(fmt.Errorf("stream finished with more than one choice"))
					return
				}

				choice := response.Choices[0]

				if choice.FinishReason != "" {
					onStream(shared.STREAM_DESCRIPTION_PHASE, nil)

					planDescription, planDescriptionJson, err := genPlanDescriptionJson(proposalId, ctx)

					if err != nil {
						onError(fmt.Errorf("failed to generate plan description json: %v", err))
						return
					}

					proposals.Update(proposalId, func(proposal *types.Proposal) {
						proposal.Finish(planDescription)
					})

					onStream(planDescriptionJson, nil)

					if planDescription.MadePlan && (len(planDescription.Files) > 0 /*|| planDescription.HasExec*/) {
						onStream(shared.STREAM_BUILD_PHASE, nil)
						err = confirmProposal(proposalId, onStream)
						if err != nil {
							onError(fmt.Errorf("failed to confirm proposal: %v", err))
						}
					} else {
						onStream(shared.STREAM_FINISHED, nil)
					}

					return
				}

				delta := choice.Delta
				content := delta.Content
				proposals.Update(proposalId, func(proposal *types.Proposal) {
					proposal.Content += content
				})

				// fmt.Printf("%s", content)
				onStream(content, nil)

			}
		}
	}()

	return nil
}
