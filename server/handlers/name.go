package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"plandex-server/model"

	"github.com/plandex/plandex/shared"
)

func FileNameHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for FileNameHandler")

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Unmarshal the request body to get the 'prompt'
	var requestBody shared.FileNameRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	if requestBody.Text == "" {
		log.Println("Received empty text field")
		http.Error(w, "text field is required", http.StatusBadRequest)
		return
	}

	modelResp, _, err := model.FileName(requestBody.Text)
	if err != nil {
		log.Printf("Error creating file name for text: %v\n", err)
		http.Error(w, "Error creating file name for text: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully processed file name request", string(modelResp))
	// Return the response from OpenAI to the client
	w.Write(modelResp)
}
