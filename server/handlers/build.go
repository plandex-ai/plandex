package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/plandex/plandex/shared"
)

func BuildHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for BuildHandler")

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Unmarshal the request body to get the 'prompt'
	var requestBody shared.BuildRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	// startStream(w, func(onStream types.OnStreamFunc) error {

	// 	err := proposal.BuildPlan(&types.BuildParams{
	// 		ProposalId:  requestBody.ProposalId,
	// 		BuildPaths:  requestBody.BuildPaths,
	// 		CurrentPlan: requestBody.CurrentPlan,
	// 	}, onStream)

	// 	if err != nil {
	// 		log.Printf("Error building plan: %v\n", err)
	// 		http.Error(w, "Error building plan: "+err.Error(), http.StatusInternalServerError)
	// 		return err
	// 	}

	// 	return nil

	// })

}
