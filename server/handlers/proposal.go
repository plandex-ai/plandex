package handlers

import (
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

	var isResponseClosed bool
	done := make(chan struct{})

	var streamMu sync.Mutex

	onStream := func(content string, err error) {
		streamMu.Lock()
		defer streamMu.Unlock()

		if isResponseClosed {
			return
		}

		if err != nil {
			isResponseClosed = true
			log.Printf("Error writing stream content to client: %v\n", err)
			close(done)
			return
		}

		// stream content to client
		if content != "" {
			// fmt.Println("writing stream content to client:", content)

			bytes := []byte(content + shared.STREAM_MESSAGE_SEPARATOR)
			_, err = w.Write(bytes)
			if err != nil {
				isResponseClosed = true
				log.Printf("Error writing stream content to client: %v\n", err)
				log.Printf("Content: %s\n", string(bytes))
				close(done)
				return
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}

		if content == shared.STREAM_FINISHED {
			isResponseClosed = true
			close(done)
			fmt.Println("proposal stream finished")
			return
		}
	}

	fmt.Println("creating proposal and starting stream")
	err = proposal.CreateProposal(requestBody, onStream)

	if err != nil {
		log.Printf("Error creating proposal: %v\n", err)
		http.Error(w, "Error creating proposal: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// block until streaming is done
	<-done

	fmt.Println("done")

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
