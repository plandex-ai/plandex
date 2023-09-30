package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"plandex-server/model/proposal"
	"sync"

	"github.com/plandex/plandex/shared"
)

func ProposalHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for ProposalHandler")

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Unmarshal the request body to get the 'prompt'
	var requestBody shared.PromptRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	if requestBody.Prompt == "" {
		log.Println("Received empty prompt field")
		http.Error(w, "prompt field is required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	var cancel *context.CancelFunc
	var isResponseClosed bool
	var mu sync.Mutex
	done := make(chan struct{})

	onStream := func(content string, finished bool, err error) {
		if isResponseClosed {
			return
		}

		if err != nil {
			mu.Lock()
			isResponseClosed = true
			mu.Unlock()
			if cancel != nil {
				(*cancel)()
			}

			log.Printf("Error writing stream content to client: %v\n", err)

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))

			close(done)

			return
		}

		if finished {
			mu.Lock()
			isResponseClosed = true
			mu.Unlock()
			if cancel != nil {
				(*cancel)()
			}
			w.Write([]byte(shared.STREAM_FINISHED))
			close(done)

			return
		}

		// stream content to client
		if content != "" {
			_, err = w.Write([]byte(content))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

			if err != nil {
				mu.Lock()
				isResponseClosed = true
				mu.Unlock()
				log.Printf("Error writing stream content to client: %v\n", err)
				if cancel != nil {
					(*cancel)()
				}
				close(done)
				return
			}

		}

	}

	cancel, err = proposal.CreateProposal(requestBody, onStream)

	if err != nil {
		log.Printf("Error creating proposal: %v\n", err)
		http.Error(w, "Error creating proposal: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// block until streaming is done
	<-done

}

func ConfirmProposalHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for ConfirmProposalHandler")

	proposalId := r.URL.Query().Get("proposalId")

	if proposalId == "" {
		log.Println("Received empty proposalId field")
		http.Error(w, "proposalId field is required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	var cancel *context.CancelFunc
	var err error
	var isResponseClosed bool

	onStream := func(planChunk *shared.PlanChunk, finished bool, err error) {
		if isResponseClosed {
			return
		}

		if err != nil {
			isResponseClosed = true
			if cancel != nil {
				(*cancel)()
			}
			log.Printf("Error writing plan chunk to client. chunk: , err: %v\n", planChunk, err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("%v: %v", planChunk, err)))
			return
		}

		if finished {
			isResponseClosed = true
			if cancel != nil {
				(*cancel)()
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(shared.STREAM_FINISHED))

			return
		}

		// stream content to client
		if planChunk != nil {
			// convert plan chunk to json
			json, err := json.Marshal(planChunk)
			if err != nil {
				isResponseClosed = true
				if cancel != nil {
					(*cancel)()
				}
				log.Printf("Error marshalling plan chunk to json. chunk: , err: %v\n", planChunk, err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("%v: %v", planChunk, err)))
				return
			}

			_, err = w.Write([]byte(json))

			if err != nil {
				isResponseClosed = true
				log.Printf("Error writing stream content to client: %v\n", err)
				if cancel != nil {
					(*cancel)()
				}
				return
			}
		}
	}

	cancel, err = proposal.ConfirmProposal(proposalId, onStream)

	if err != nil {
		log.Printf("Error confirming proposal: %v\n", err)
		http.Error(w, "Error confirming proposal: "+err.Error(), http.StatusInternalServerError)
		return
	}

}

func AbortProposalHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for AbortProposalHandler")

	proposalId := r.URL.Query().Get("proposalId")

	if proposalId == "" {
		log.Println("Received empty proposalId field")
		http.Error(w, "proposalId field is required", http.StatusBadRequest)
		return
	}

	err := proposal.AbortProposal(proposalId)
	if err != nil {
		log.Printf("Error aborting proposal: %v\n", err)
		http.Error(w, "Error aborting proposal: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully aborted proposal")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))
}

// func ReviseProposalHandler(w http.ResponseWriter, r *http.Request) {
// 	log.Println("Received a request for ReviseProposalHandler")

// 	// Read the request body
// 	body, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		log.Printf("Error reading request body: %v\n", err)
// 		http.Error(w, "Error reading request body", http.StatusInternalServerError)
// 		return
// 	}
// 	defer r.Body.Close()

// 	// Unmarshal the request body to get the 'prompt'
// 	var requestBody shared.PromptRequest
// 	if err := json.Unmarshal(body, &requestBody); err != nil {
// 		log.Printf("Error parsing request body: %v\n", err)
// 		http.Error(w, "Error parsing request body", http.StatusBadRequest)
// 		return
// 	}

// 	if requestBody.Prompt == "" {
// 		log.Println("Received empty prompt field")
// 		http.Error(w, "prompt field is required", http.StatusBadRequest)
// 		return
// 	}

// 	modelRes, err := model.ReviseProposal(requestBody.ProposalId, requestBody.NewPrompt)
// 	if err != nil {
// 		log.Printf("Error revising proposal: %v\n", err)
// 		http.Error(w, "Error revising proposal: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	log.Println("Successfully revised proposal")

// 	// Return the response from OpenAI to the client
// 	w.Write(modelRes)
// }
