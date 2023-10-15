package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"plandex-server/model"
	"plandex-server/types"

	"github.com/google/uuid"
	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

const systemMessageHead = `You are Plandex, an AI programming and system administration assistant. You offer a structured, versioned, and iterative approach to AI-driven development. 
		
		You and the programmer collaborate to create a 'plan' for the task at hand. A plan is a set of files with an attached context.` +

	"Your instructions:\n\n```\n" +

	`Based on user-provided context, create a plan for the task using the following steps:

		1. Decide whether you've been given enough information and context to make a good plan. 
			a. If not:
		    - Explicitly say "I need more information or context to make a plan for this task."
			  - Ask the user for more information or context and stop there.

		2. Decide whether this task is small enough to be completed in a single response.
			a. If so, write out the code to complete the task. Precede the code block with the file path like this '- file_path:'--for example:
				- src/main.rs:				
				- lib/utils.go:
				- main.py:
				File paths should always come *before* the opening triple backticks of a code block. They should *not* be included in the code block itself.
				File paths should appear *immediately* before the opening triple backticks of a code block. There should be *no other lines* between the file path and the code block. Any explanations should come either *before the file path or after the code block.*
			b. If not: 
			  - Explicitly say "I will break this large task into subtasks."
				- Divide the task into smaller subtasks and list them in a numbered list. Stop there.
		
		Always precede code blocks the file path as described above in 2a. Code must *always* be labelled with the path. You can have multiple code blocks labelled with the same file path. 
		
		Every file you reference should either exist in the context directly or be a new file that will be created in the same base directory a file in the context. For example, if there is a file in context at path 'lib/utils.go', you can create a new file at path 'lib/utils_test.go' but *not* at path 'src/lib/utils.go'.

		For code in markdown blocks, always include the language name after the opening triple backticks.
		
		Don't include unnecessary comments in code. Lean towards no comments as much as you can. If you must include a comment to make the code understandable, be sure it is concise. Don't use comments to communicate with the user.

		At the end of a plan, you can suggest additional iterations to make the plan better. You can also ask the user to load more files or information into context if it would help you make a better plan.` +
	"\n```\n\n" +
	"User-provided context:"

var systemHeadNumTokens = shared.GetNumTokens(systemMessageHead)

const promptWrapperFormatStr = "The user's latest prompt:\n```\n%s\n```\n\nPlease respond according to the 'Your instructions' section above. Remember to precede code blocks with the file path *exactly* as described in 2a. Do not use any other formatting for file paths."

var promptWrapperTokens = shared.GetNumTokens(fmt.Sprintf(promptWrapperFormatStr, ""))

// Proposal function to create a new proposal
func CreateProposal(req shared.PromptRequest, onStream types.OnStreamFunc) error {
	goEnv := os.Getenv("GOENV") // Fetch the GO_ENV environment variable

	fmt.Println("GOENV: " + goEnv)
	if goEnv == "test" {
		streamLoremIpsum(onStream)
		return nil
	}

	contextText, contextTokens := model.FormatModelContext(req.ModelContext)
	systemMessageText := systemMessageHead + contextText
	systemMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemMessageText,
	}

	messages := []openai.ChatCompletionMessage{
		systemMessage,
	}

	promptTokens := promptWrapperTokens + shared.GetNumTokens(req.Prompt)

	totalTokens := systemHeadNumTokens + contextTokens + promptTokens

	// print out breakdown of token usage
	fmt.Printf("System message tokens: %d\n", systemHeadNumTokens)
	fmt.Printf("Context tokens: %d\n", contextTokens)
	fmt.Printf("Prompt tokens: %d\n", promptTokens)
	fmt.Printf("Total tokens before convo: %d\n", totalTokens)

	if totalTokens > shared.MaxTokens {
		// token limit already exceeded before adding conversation
		err := fmt.Errorf("token limit exceeded before adding conversation")
		fmt.Printf("Error: %v\n", err)
		return err
	}

	if len(req.Conversation) > 0 {
		for _, convoMessage := range req.Conversation {
			messages = append(messages, convoMessage.Message)
		}
	}

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: fmt.Sprintf(promptWrapperFormatStr, req.Prompt),
	})

	fmt.Println("\n\nMessages:")
	for _, message := range messages {
		fmt.Printf("%s: %s\n", message.Role, message.Content)
	}

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
		Id:       proposalId,
		ParentId: req.ParentProposalId,
		IsRoot:   req.ParentProposalId == "",
		Request:  &req,
		Content:  "",
		ProposalStage: types.ProposalStage{
			CancelFn: &cancel,
		},
	})

	replyInfo := shared.NewReplyInfo()

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

					files, _, _ := replyInfo.FinishAndRead()

					if len(files) == 0 {
						planDescription := &shared.PlanDescription{
							Files:    []string{},
							MadePlan: false,
						}
						bytes, err := json.Marshal(planDescription)
						if err != nil {
							onError(fmt.Errorf("failed to marshal plan description: %v", err))
							return
						}
						planDescriptionJson := string(bytes)

						proposals.Update(proposalId, func(proposal *types.Proposal) {
							proposal.Finish(planDescription)
						})

						fmt.Println(planDescriptionJson)

						onStream(planDescriptionJson, nil)
						onStream(shared.STREAM_FINISHED, nil)
						return
					}

					planDescription, err := genPlanDescriptionJson(proposalId, ctx)
					if err != nil {
						onError(fmt.Errorf("failed to generate plan description json: %v", err))
						return
					}

					planDescription.MadePlan = true
					planDescription.Files = files

					bytes, err := json.Marshal(planDescription)
					if err != nil {
						onError(fmt.Errorf("failed to marshal plan description: %v", err))
						return
					}
					planDescriptionJson := string(bytes)
					fmt.Println(planDescriptionJson)

					proposals.Update(proposalId, func(proposal *types.Proposal) {
						proposal.Finish(planDescription)
					})

					onStream(planDescriptionJson, nil)

					onStream(shared.STREAM_BUILD_PHASE, nil)
					err = confirmProposal(proposalId, onStream)
					if err != nil {
						onError(fmt.Errorf("failed to confirm proposal: %v", err))
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
				replyInfo.AddToken(content, true)

			}
		}
	}()

	return nil
}

// func summarizeConversationIfPossible(req *shared.PromptRequest) (*shared.PromptRequest, error) {
// 	msgs := req.Conversation
// 	if len(msgs) >= 2 {
// 		for i := 1; i < len(msgs); i++ {
// 			if msgs[i].Role == openai.ChatMessageRoleAssistant && !msgs[i].Summarized {
// 				// assuming the summarize function takes a text and returns
// 				// a summarization of that text
// 				summary, err := model.Summarize(msgs[i].Content)
// 				if err != nil {
// 					return nil, err
// 				}

// 				// In case the message has multiple sentences. Concatenate the first
// 				// two and add '... (summarized)' for clarity on UI.
// 				sentences := strings.Split(summary, ".")
// 				if len(sentences) > 2 {
// 					summary = sentences[0] + ". " + sentences[1] + "... (summarized)"
// 				}

// 				msgs[i].Content = summary
// 				msgs[i].Summarized = true
// 				break
// 			}
// 		}
// 	}
// 	req.Conversation = msgs
// 	return req, nil
// }
