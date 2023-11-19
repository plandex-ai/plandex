package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"plandex-server/model/proposal"
	"plandex-server/types"

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

	startStream(w, func(onStream types.OnStreamFunc) error {
		fmt.Println("creating proposal and starting stream")
		err = proposal.CreateProposal(requestBody, onStream)

		if err != nil {
			log.Printf("Error creating proposal: %v\n", err)
			http.Error(w, "Error creating proposal: "+err.Error(), http.StatusInternalServerError)
			return err
		}

		return nil
	})
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
