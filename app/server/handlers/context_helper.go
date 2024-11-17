package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"plandex-server/model"
	"plandex-server/types"

	"github.com/plandex/plandex/shared"
	"github.com/sashabaranov/go-openai"
)

func loadContexts(w http.ResponseWriter, r *http.Request, auth *types.ServerAuth, loadReq *shared.LoadContextRequest, plan *db.Plan, branchName string) (*shared.LoadContextResponse, []*db.Context) {
	// check file count and size limits
	totalFiles := 0
	for _, context := range *loadReq {
		totalFiles++
		if totalFiles > shared.MaxContextCount {
			log.Printf("Error: Too many contexts to load (found %d, limit is %d)\n", totalFiles, shared.MaxContextCount)
			http.Error(w, fmt.Sprintf("Too many contexts to load (found %d, limit is %d)", totalFiles, shared.MaxContextCount), http.StatusBadRequest)
			return nil, nil
		}

		fileSize := int64(len(context.Body))
		if fileSize > shared.MaxContextBodySize {
			log.Printf("Error: Context %s exceeds size limit (size %.2f MB, limit %d MB)\n", context.Name, float64(fileSize)/1024/1024, int(shared.MaxContextBodySize)/1024/1024)
			http.Error(w, fmt.Sprintf("Context %s exceeds size limit (size %.2f MB, limit %d MB)", context.Name, float64(fileSize)/1024/1024, int(shared.MaxContextBodySize)/1024/1024), http.StatusBadRequest)
			return nil, nil
		}
	}

	var err error
	var settings *shared.PlanSettings
	var client *openai.Client

	for _, context := range *loadReq {
		if context.ContextType == shared.ContextPipedDataType || context.ContextType == shared.ContextNoteType || context.ContextType == shared.ContextImageType {
			settings, err = db.GetPlanSettings(plan, true)

			if err != nil {
				log.Printf("Error getting plan settings: %v\n", err)
				http.Error(w, "Error getting plan settings: "+err.Error(), http.StatusInternalServerError)
				return nil, nil
			}

			clients := initClients(
				initClientsParams{
					w:           w,
					auth:        auth,
					apiKeys:     context.ApiKeys,
					openAIBase:  context.OpenAIBase,
					openAIOrgId: context.OpenAIOrgId,
					plan:        plan,
				},
			)

			envVar := settings.ModelPack.Namer.BaseModelConfig.ApiKeyEnvVar
			client = clients[envVar]

			break
		}
	}

	// ensure image compatibility if we're loading an image
	for _, context := range *loadReq {
		if context.ContextType == shared.ContextImageType {
			if !settings.ModelPack.Planner.BaseModelConfig.HasImageSupport {
				log.Printf("Error loading context: %s does not support images in context\n", settings.ModelPack.Planner.BaseModelConfig.ModelName)
				http.Error(w, fmt.Sprintf("Error loading context: %s does not support images in context", settings.ModelPack.Planner.BaseModelConfig.ModelName), http.StatusBadRequest)
				return nil, nil
			}
		}
	}

	// get name for piped data or notes if present
	num := 0
	errCh := make(chan error, len(*loadReq))
	for _, context := range *loadReq {
		if context.ContextType == shared.ContextPipedDataType {
			num++

			go func(context *shared.LoadContextParams) {
				name, err := model.GenPipedDataName(auth, plan, settings, client, context.Body)

				if err != nil {
					errCh <- fmt.Errorf("error generating name for piped data: %v", err)
				}

				context.Name = name
				errCh <- nil
			}(context)
		} else if context.ContextType == shared.ContextNoteType {
			num++

			go func(context *shared.LoadContextParams) {
				name, err := model.GenNoteName(auth, plan, settings, client, context.Body)

				if err != nil {
					errCh <- fmt.Errorf("error generating name for note: %v", err)
				}

				context.Name = name
				errCh <- nil
			}(context)
		}
	}
	if num > 0 {
		for i := 0; i < num; i++ {
			err := <-errCh
			if err != nil {
				log.Printf("Error: %v\n", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return nil, nil
			}
		}
	}

	ctx, cancel := context.WithCancel(r.Context())
	unlockFn := LockRepo(w, r, auth, db.LockScopeWrite, ctx, cancel, true)
	if unlockFn == nil {
		return nil, nil
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	res, dbContexts, err := db.LoadContexts(ctx, db.LoadContextsParams{
		OrgId:      auth.OrgId,
		Plan:       plan,
		BranchName: branchName,
		Req:        loadReq,
		UserId:     auth.User.Id,
	})

	if err != nil {
		log.Printf("Error loading contexts: %v\n", err)
		http.Error(w, "Error loading contexts: "+err.Error(), http.StatusInternalServerError)
		return nil, nil
	}

	if res.MaxTokensExceeded {
		log.Printf("The total number of tokens (%d) exceeds the maximum allowed (%d)", res.TotalTokens, res.MaxTokens)
		bytes, err := json.Marshal(res)

		if err != nil {
			log.Printf("Error marshalling response: %v\n", err)
			http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
			return nil, nil
		}

		w.Write(bytes)
		return nil, nil
	}

	err = db.GitAddAndCommit(auth.OrgId, plan.Id, branchName, res.Msg)

	if err != nil {
		log.Printf("Error committing changes: %v\n", err)
		http.Error(w, "Error committing changes: "+err.Error(), http.StatusInternalServerError)
		return nil, nil
	}

	return res, dbContexts
}
