package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"plandex-server/model"

	"github.com/plandex/plandex/shared"
)

func PromptHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for PromptHandler")

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

	modelRes, err := model.Prompt(requestBody)
	if err != nil {
		log.Printf("Error sending prompt to model: %v\n", err)
		http.Error(w, "Error sending prompt to model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully processed prompt request")

	fmt.Println("Final response from model:")
	fmt.Println(string(modelRes))

	// Return the response from OpenAI to the client
	w.Write(modelRes)
}
