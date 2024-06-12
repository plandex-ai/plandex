package plan

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/model"
	"plandex-server/types"
	"strings"
	"time"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

const MaxAutoContinueIterations = 100

func (state *activeTellStreamState) listenStream(stream *openai.ChatCompletionStream) {
	defer stream.Close()

	clients := state.clients
	auth := state.auth
	req := state.req
	plan := state.plan
	planId := plan.Id
	branch := state.branch
	currentOrgId := state.currentOrgId
	currentUserId := state.currentUserId
	convo := state.convo
	summaries := state.summaries
	summarizedToMessageId := state.summarizedToMessageId
	iteration := state.iteration
	missingFileResponse := state.missingFileResponse
	replyId := state.replyId
	replyParser := state.replyParser
	settings := state.settings

	active := GetActivePlan(planId, branch)

	if active == nil {
		log.Printf("listenStream - Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return
	}

	replyFiles := []string{}
	chunksReceived := 0
	maybeRedundantBacktickContent := ""

	// Create a timer that will trigger if no chunk is received within the specified duration
	timer := time.NewTimer(model.OPENAI_STREAM_CHUNK_TIMEOUT)
	defer timer.Stop()

	for {
		select {
		case <-active.Ctx.Done():
			// The main modelContext was canceled (not the timer)
			log.Println("\nTell: stream canceled")
			return
		case <-timer.C:
			// Timer triggered because no new chunk was received in time
			log.Println("\nTell: stream timeout due to inactivity")
			state.onError(fmt.Errorf("stream timeout due to inactivity"), true, "", "")
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
				if err.Error() == "context canceled" {
					log.Println("Tell: stream context canceled")
					return
				}

				state.onError(fmt.Errorf("stream error: %v", err), true, "", "")
				return
			}

			if len(response.Choices) == 0 {
				state.onError(fmt.Errorf("stream finished with no choices"), true, "", "")
				return
			}

			if len(response.Choices) > 1 {
				state.onError(fmt.Errorf("stream finished with more than one choice"), true, "", "")
				return
			}

			choice := response.Choices[0]

			if choice.FinishReason != "" {
				log.Println("Model stream finished")

				active.Stream(shared.StreamMessage{
					Type: shared.StreamMessageDescribing,
				})

				err := db.SetPlanStatus(planId, branch, shared.PlanStatusDescribing, "")
				if err != nil {
					state.onError(fmt.Errorf("failed to set plan status to describing: %v", err), true, "", "")
					return
				}

				latestSummaryCh := active.LatestSummaryCh

				var generatedDescription *db.ConvoMessageDescription
				var shouldContinue bool
				var nextTask string
				var errCh = make(chan error, 2)

				go func() {
					if len(replyFiles) > 0 {
						log.Println("Generating plan description")

						envVar := settings.ModelPack.CommitMsg.BaseModelConfig.ApiKeyEnvVar
						client := clients[envVar]

						res, err := genPlanDescription(client, settings.ModelPack.CommitMsg, planId, branch, active.Ctx)
						if err != nil {
							errCh <- fmt.Errorf("failed to generate plan description: %v", err)
							return
						}

						generatedDescription = res
						generatedDescription.OrgId = currentOrgId
						generatedDescription.SummarizedToMessageId = summarizedToMessageId
						generatedDescription.MadePlan = true
						generatedDescription.Files = replyFiles
					}
					errCh <- nil
				}()

				go func() {
					log.Println("Getting exec status")
					shouldContinue, nextTask, err = state.execStatusShouldContinue(active.CurrentReplyContent, latestSummaryCh, active.Ctx)
					if err != nil {
						errCh <- fmt.Errorf("failed to get exec status: %v", err)
						return
					}

					log.Printf("Should continue: %v\n", shouldContinue)

					errCh <- nil
				}()

				for i := 0; i < 2; i++ {
					err := <-errCh
					if err != nil {
						state.onError(err, true, "", "")
						return
					}
				}

				log.Println("Locking repo to store assistant reply and description")

				repoLockId, err := db.LockRepo(
					db.LockRepoParams{
						OrgId:    currentOrgId,
						UserId:   currentUserId,
						PlanId:   planId,
						Branch:   branch,
						Scope:    db.LockScopeWrite,
						Ctx:      active.Ctx,
						CancelFn: active.CancelFn,
					},
				)

				if err != nil {
					log.Printf("Error locking repo: %v\n", err)
					active.StreamDoneCh <- &shared.ApiError{
						Type:   shared.ApiErrorTypeOther,
						Status: http.StatusInternalServerError,
						Msg:    "Error locking repo",
					}
					return
				}

				log.Println("Locked repo for assistant reply and description")

				err = func() error {
					defer func() {
						if err != nil {
							log.Printf("Error storing reply and description: %v\n", err)
							err = db.GitClearUncommittedChanges(auth.OrgId, planId)
							if err != nil {
								log.Printf("Error clearing uncommitted changes: %v\n", err)
							}
						}

						log.Println("Unlocking repo for assistant reply and description")

						err = db.DeleteRepoLock(repoLockId)
						if err != nil {
							log.Printf("Error unlocking repo: %v\n", err)
							active.StreamDoneCh <- &shared.ApiError{
								Type:   shared.ApiErrorTypeOther,
								Status: http.StatusInternalServerError,
								Msg:    "Error unlocking repo",
							}
						}
					}()

					assistantMsg, convoCommitMsg, err := state.storeAssistantReply() // updates state.convo
					convo = state.convo

					if err != nil {
						state.onError(fmt.Errorf("failed to store assistant message: %v", err), true, "", "")
						return err
					}

					log.Println("getting description for assistant message: ", assistantMsg.Id)

					var description *db.ConvoMessageDescription
					if len(replyFiles) == 0 {
						description = &db.ConvoMessageDescription{
							OrgId:                 currentOrgId,
							PlanId:                planId,
							ConvoMessageId:        assistantMsg.Id,
							SummarizedToMessageId: summarizedToMessageId,
							BuildPathsInvalidated: map[string]bool{},
							MadePlan:              false,
						}
					} else {
						description = generatedDescription
						description.ConvoMessageId = assistantMsg.Id
					}

					log.Println("Storing description")
					err = db.StoreDescription(description)

					if err != nil {
						state.onError(fmt.Errorf("failed to store description: %v", err), false, assistantMsg.Id, convoCommitMsg)
						return err
					}
					log.Println("Description stored")
					// spew.Dump(description)

					log.Println("Comitting reply message and description")

					err = db.GitAddAndCommit(currentOrgId, planId, branch, convoCommitMsg)
					if err != nil {
						state.onError(fmt.Errorf("failed to commit: %v", err), false, assistantMsg.Id, convoCommitMsg)
						return err
					}
					log.Println("Assistant reply and description committed")

					return nil
				}()

				if err != nil {
					return
				}

				// summarize convo needs to come *after* the reply is stored in order to correctly summarize the latest message
				log.Println("summarize convo")
				envVar := settings.ModelPack.PlanSummary.BaseModelConfig.ApiKeyEnvVar
				client := clients[envVar]

				// summarize in the background
				go summarizeConvo(client, settings.ModelPack.PlanSummary, summarizeConvoParams{
					planId:       planId,
					branch:       branch,
					convo:        convo,
					summaries:    summaries,
					userPrompt:   state.userPrompt,
					currentOrgId: currentOrgId,
					currentReply: active.CurrentReplyContent,
				}, active.SummaryCtx)

				log.Println("Sending active.CurrentReplyDoneCh <- true")

				active.CurrentReplyDoneCh <- true

				log.Println("Resetting active.CurrentReplyDoneCh")

				UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
					ap.CurrentStreamingReplyId = ""
					ap.CurrentReplyDoneCh = nil
				})

				if req.AutoContinue && shouldContinue && iteration < MaxAutoContinueIterations {
					log.Println("Auto continue plan")
					// continue plan
					execTellPlan(clients, plan, branch, auth, req, iteration+1, "", false, nextTask)
				} else {
					var buildFinished bool
					UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
						buildFinished = ap.BuildFinished()
						ap.RepliesFinished = true
					})

					log.Printf("Won't continue plan. Build finished: %v\n", buildFinished)

					time.Sleep(50 * time.Millisecond)

					if buildFinished {
						log.Println("Plan is finished")
						active.Stream(shared.StreamMessage{
							Type: shared.StreamMessageFinished,
						})
					} else {
						log.Println("Plan is still building")
						log.Println("Updating status to building")
						err := db.SetPlanStatus(planId, branch, shared.PlanStatusBuilding, "")
						if err != nil {
							log.Printf("Error setting plan status to building: %v\n", err)
							active.StreamDoneCh <- &shared.ApiError{
								Type:   shared.ApiErrorTypeOther,
								Status: http.StatusInternalServerError,
								Msg:    "Error setting plan status to building",
							}
							return
						}

						log.Println("Sending RepliesFinished stream message")
						active.Stream(shared.StreamMessage{
							Type: shared.StreamMessageRepliesFinished,
						})
					}
				}

				return
			}

			chunksReceived++
			delta := choice.Delta
			content := delta.Content

			if missingFileResponse != "" {
				if maybeRedundantBacktickContent != "" {
					if strings.Contains(content, "\n") {
						maybeRedundantBacktickContent = ""
					} else {
						maybeRedundantBacktickContent += content
					}
					continue
				} else if chunksReceived < 3 && strings.Contains(content, "```") {
					// received closing triple backticks in first 3 chunks after missing file response
					// means this is a redundant start of a new file block, so just ignore it

					maybeRedundantBacktickContent += content
					continue
				}
			}

			UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
				ap.CurrentReplyContent += content
				ap.NumTokens++
			})

			// log.Printf("Sending stream msg: %s", content)
			active.Stream(shared.StreamMessage{
				Type:       shared.StreamMessageReply,
				ReplyChunk: content,
			})

			replyParser.AddChunk(content, true)
			parserRes := replyParser.Read()
			files := parserRes.Files
			fileContents := parserRes.FileContents
			state.replyNumTokens = parserRes.TotalTokens
			currentFile := parserRes.CurrentFilePath
			fileDescriptions := parserRes.FileDescriptions

			// log.Printf("currentFile: %s\n", currentFile)
			// log.Println("files:")
			// spew.Dump(files)

			// Handle file that is present in project paths but not in context
			// Prompt user for what to do on the client side, stop the stream, and wait for user response before proceeding
			if currentFile != "" &&
				active.ContextsByPath[currentFile] == nil &&
				req.ProjectPaths[currentFile] && !active.AllowOverwritePaths[currentFile] {
				log.Printf("Attempting to overwrite a file that isn't in context: %s\n", currentFile)

				// attempting to overwrite a file that isn't in context
				// we will stop the stream and ask the user what to do
				err := db.SetPlanStatus(planId, branch, shared.PlanStatusMissingFile, "")

				if err != nil {
					log.Printf("Error setting plan %s status to prompting: %v\n", planId, err)
					active.StreamDoneCh <- &shared.ApiError{
						Type:   shared.ApiErrorTypeOther,
						Status: http.StatusInternalServerError,
						Msg:    "Error setting plan status to prompting",
					}
					return
				}

				UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
					ap.MissingFilePath = currentFile
				})

				log.Printf("Prompting user for missing file: %s\n", currentFile)

				active.Stream(shared.StreamMessage{
					Type:            shared.StreamMessagePromptMissingFile,
					MissingFilePath: currentFile,
				})

				log.Printf("Stopping stream for missing file: %s\n", currentFile)

				// log.Printf("Current reply content: %s\n", active.CurrentReplyContent)

				// stop stream for now
				active.CancelModelStreamFn()

				log.Printf("Stopped stream for missing file: %s\n", currentFile)

				// wait for user response to come in
				var userChoice shared.RespondMissingFileChoice
				select {
				case <-active.Ctx.Done():
					log.Println("Context cancelled while waiting for missing file response")
					return
				case userChoice = <-active.MissingFileResponseCh:
				}

				log.Printf("User choice for missing file: %s\n", userChoice)

				active.ResetModelCtx()

				UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
					ap.MissingFilePath = ""
				})

				log.Println("Continuing stream")

				// continue plan
				execTellPlan(
					clients,
					plan,
					branch,
					auth,
					req,
					iteration, // keep the same iteration
					userChoice,
					false,
					"",
				)
				return
			}

			// log.Println("Content:", content)
			// log.Println("Current reply content:", active.CurrentReplyContent)
			// log.Println("Current file:", currentFile)
			// log.Println("files:")
			// spew.Dump(files)
			// log.Println("replyFiles:")
			// spew.Dump(replyFiles)

			if len(files) > len(replyFiles) {
				log.Printf("%d new files\n", len(files)-len(replyFiles))

				for i, file := range files {
					if i < len(replyFiles) {
						continue
					}

					log.Printf("Detected file: %s\n", file)
					if req.BuildMode == shared.BuildModeAuto {
						log.Printf("Queuing build for %s\n", file)
						buildState := &activeBuildStreamState{
							clients:       clients,
							auth:          auth,
							currentOrgId:  currentOrgId,
							currentUserId: currentUserId,
							plan:          plan,
							branch:        branch,
							settings:      settings,
							modelContext:  state.modelContext,
						}

						fileContentTokens, err := shared.GetNumTokens(fileContents[i])

						if err != nil {
							log.Printf("Error getting num tokens for file %s: %v\n", file, err)
							state.onError(fmt.Errorf("error getting num tokens for file %s: %v", file, err), true, "", "")
							return
						}

						buildState.queueBuilds([]*types.ActiveBuild{{
							ReplyId:           replyId,
							Idx:               i,
							FileDescription:   fileDescriptions[i],
							FileContent:       fileContents[i],
							FileContentTokens: fileContentTokens,
							Path:              file,
						}})
					}
					replyFiles = append(replyFiles, file)
					UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
						ap.Files = append(ap.Files, file)
					})
				}
			}
		}
	}
}

func (state *activeTellStreamState) storeAssistantReply() (*db.ConvoMessage, string, error) {
	currentOrgId := state.currentOrgId
	currentUserId := state.currentUserId
	planId := state.plan.Id
	branch := state.branch
	auth := state.auth
	replyNumTokens := state.replyNumTokens
	replyId := state.replyId
	convo := state.convo

	num := len(convo) + 1

	log.Printf("storing assistant reply | len(convo) %d | num %d\n", len(convo), num)

	activePlan := GetActivePlan(planId, branch)

	if activePlan == nil {
		return nil, "", fmt.Errorf("active plan not found")
	}

	assistantMsg := db.ConvoMessage{
		Id:      replyId,
		OrgId:   currentOrgId,
		PlanId:  planId,
		UserId:  currentUserId,
		Role:    openai.ChatMessageRoleAssistant,
		Tokens:  replyNumTokens,
		Num:     num,
		Message: activePlan.CurrentReplyContent,
	}

	commitMsg, err := db.StoreConvoMessage(&assistantMsg, auth.User.Id, branch, false)

	if err != nil {
		log.Printf("Error storing assistant message: %v\n", err)
		return nil, "", err
	}

	UpdateActivePlan(planId, branch, func(ap *types.ActivePlan) {
		ap.MessageNum = num
		ap.StoredReplyIds = append(ap.StoredReplyIds, replyId)
	})

	convo = append(convo, &assistantMsg)
	state.convo = convo

	return &assistantMsg, commitMsg, err
}

func (state *activeTellStreamState) onError(streamErr error, storeDesc bool, convoMessageId, commitMsg string) {
	log.Printf("\nStream error: %v\n", streamErr)

	planId := state.plan.Id
	branch := state.branch
	currentOrgId := state.currentOrgId
	summarizedToMessageId := state.summarizedToMessageId

	active := GetActivePlan(planId, branch)

	if active == nil {
		log.Printf("tellStream onError - Active plan not found for plan ID %s on branch %s\n", planId, branch)
		return
	}

	storeDescAndReply := func() {
		ctx, cancelFn := context.WithCancel(context.Background())

		repoLockId, err := db.LockRepo(
			db.LockRepoParams{
				UserId:   state.currentUserId,
				OrgId:    state.currentOrgId,
				PlanId:   planId,
				Branch:   branch,
				Scope:    db.LockScopeWrite,
				Ctx:      ctx,
				CancelFn: cancelFn,
			},
		)

		if err != nil {
			log.Printf("Error locking repo for plan %s: %v\n", planId, err)
		} else {

			defer func() {
				err := db.DeleteRepoLock(repoLockId)
				if err != nil {
					log.Printf("Error unlocking repo for plan %s: %v\n", planId, err)
				}
			}()

			err := db.GitClearUncommittedChanges(state.currentOrgId, planId)
			if err != nil {
				log.Printf("Error clearing uncommitted changes for plan %s: %v\n", planId, err)
			}
		}

		storedMessage := false
		storedDesc := false

		if convoMessageId == "" {
			assistantMsg, msg, err := state.storeAssistantReply()
			if err == nil {
				convoMessageId = assistantMsg.Id
				commitMsg = msg
				storedMessage = true
			} else {
				log.Printf("Error storing assistant message after stream error: %v\n", err)
			}
		}

		if storeDesc && convoMessageId != "" {
			err := db.StoreDescription(&db.ConvoMessageDescription{
				OrgId:                 currentOrgId,
				PlanId:                planId,
				SummarizedToMessageId: summarizedToMessageId,
				MadePlan:              false,
				ConvoMessageId:        convoMessageId,
				BuildPathsInvalidated: map[string]bool{},
				Error:                 streamErr.Error(),
			})
			if err == nil {
				storedDesc = true
			} else {
				log.Printf("Error storing description after stream error: %v\n", err)
			}
		}

		if storedMessage || storedDesc {
			err := db.GitAddAndCommit(currentOrgId, planId, branch, commitMsg)
			if err != nil {
				log.Printf("Error committing after stream error: %v\n", err)
			}
		}
	}

	storeDescAndReply()

	active.StreamDoneCh <- &shared.ApiError{
		Type:   shared.ApiErrorTypeOther,
		Status: http.StatusInternalServerError,
		Msg:    "Stream error: " + streamErr.Error(),
	}
}
