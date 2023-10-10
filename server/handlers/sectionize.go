package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"plandex-server/model"
	"github.com/plandex/plandex/shared"
)

func SectionizeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for SectionizeHandler")

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Unmarshal the request body to obtain the 'text' that needs sectionizing
	var requestBody shared.SectionizeRequest
	if err := json.Unmarshal(body, &requestBody); err != nil {
		log.Printf("Error parsing request body: %v\n", err)
		http.Error(w, "Error parsing request body", http.StatusBadRequest)
		return
	}

	if requestBody.Text == "" {
		log.Println("Received an empty text field")
		http.Error(w, "text field is required", http.StatusBadRequest)
		return
	}

	// Call the model's Sectionize function and handle any returned errors
	modelRes, err := model.Sectionize(requestBody.Text)
	if err != nil {
		log.Printf("Error in sectionizing text: %v\n", err)
		http.Error(w, "Error in sectionizing text: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully processed sectionize request")
	// Write the response from the model's Sectionize function back to the client
	w.Write(modelRes)
}
