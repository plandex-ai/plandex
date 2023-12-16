package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// func ConvoSummaryHandler(w http.ResponseWriter, r *http.Request) {
// 	log.Println("Received a request for ConvoSummaryHandler")

// 	// get the rootId from the request
// 	vars := mux.Vars(r)
// 	rootId := vars["rootId"]

// 	if rootId == "" {
// 		log.Println("Received empty rootId field")
// 		http.Error(w, "rootId field is required", http.StatusBadRequest)
// 		return
// 	}

// 	latestTimestamp := r.URL.Query().Get("latestTimestamp")

// 	summary := plan.GetConvoSummary(rootId)

// 	if summary == nil {
// 		log.Printf("No summary found for rootId %s\n", rootId)
// 		w.WriteHeader(http.StatusNoContent)
// 		return
// 	}

// 	if latestTimestamp != "" && latestTimestamp == summary.LastMessageTimestamp {
// 		log.Printf("No new messages found for rootId %s\n", rootId)
// 		w.WriteHeader(http.StatusNoContent)
// 		return
// 	}

// 	jsonBytes, err := json.Marshal(summary)
// 	if err != nil {
// 		log.Printf("Error marshalling summary: %v\n", err)
// 		http.Error(w, "Error marshalling summary: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	log.Println("Successfully processed ConvoSummaryHandler request")
// 	w.Write(jsonBytes)
// }

func ListConvoHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request for ListConvoHandler")

	vars := mux.Vars(r)
	planId := vars["planId"]

	log.Println("planId: ", planId)

}
