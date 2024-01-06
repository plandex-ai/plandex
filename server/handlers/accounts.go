package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"

	"github.com/plandex/plandex/shared"
)

func StartTrialHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for StartTrialHandler")

	// start a transaction
	tx, err := db.Conn.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v\n", err)
		http.Error(w, "Error starting transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Ensure that rollback is attempted in case of failure
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("transaction rollback error: %v\n", rbErr)
			} else {
				log.Println("transaction rolled back")
			}
		}
	}()

	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		log.Printf("Error generating random tag: %v\n", err)
		http.Error(w, "Error generating random tag: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tag := fmt.Sprintf("%x", b)

	name := "Trial User " + tag
	email := tag + "@trial.plandex.ai"

	var userId string
	userRow, err := tx.Query("INSERT INTO users (name, email, is_trial) VALUES ($1, $2, true) RETURNING id", name, email)
	closedUserRow := false

	if err != nil {
		log.Printf("Error creating user: %v\n", err)
		http.Error(w, "Error creating user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	defer func() {
		if !closedUserRow {
			userRow.Close()
		}
	}()

	if userRow.Next() {
		if err := userRow.Scan(&userId); err != nil {
			log.Printf("Error scanning user: %v\n", err)
			http.Error(w, "Error scanning user: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = userRow.Close()
	if err != nil {
		log.Printf("Error closing user row: %v\n", err)
		http.Error(w, "Error closing user row: "+err.Error(), http.StatusInternalServerError)
		return
	}
	closedUserRow = true

	// create a new org
	orgName := "Trial Org " + tag
	orgRow, err := tx.Query("INSERT INTO orgs (name, owner_id, is_trial) VALUES ($1, $2, true) RETURNING id", orgName, userId)
	closedOrgRow := false

	if err != nil {
		log.Printf("Error creating org: %v\n", err)
		http.Error(w, "Error creating org: "+err.Error(), http.StatusInternalServerError)
		return
	}

	defer func() {
		if !closedOrgRow {
			orgRow.Close()
		}
	}()

	var orgId string
	if orgRow.Next() {
		if err := orgRow.Scan(&orgId); err != nil {
			log.Printf("Error scanning org: %v\n", err)
			http.Error(w, "Error scanning org: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = orgRow.Close()
	if err != nil {
		log.Printf("Error closing org row: %v\n", err)
		http.Error(w, "Error closing org row: "+err.Error(), http.StatusInternalServerError)
		return
	}
	closedOrgRow = true

	// insert org user
	_, err = tx.Exec("INSERT INTO orgs_users (org_id, user_id) VALUES ($1, $2)", orgId, userId)
	if err != nil {
		log.Printf("Error inserting org user: %v\n", err)
		http.Error(w, "Error inserting org user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// create auth token
	token, err := db.CreateAuthToken(userId, tx)

	if err != nil {
		log.Printf("Error creating auth token: %v\n", err)
		http.Error(w, "Error creating auth token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// commit transaction
	err = tx.Commit()
	if err != nil {
		log.Printf("Error committing transaction: %v\n", err)
		http.Error(w, "Error committing transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := shared.StartTrialResponse{
		UserId:   userId,
		OrgId:    orgId,
		Token:    token,
		UserName: name,
		OrgName:  orgName,
	}

	bytes, err := json.Marshal(resp)

	if err != nil {
		log.Printf("Error marshalling response: %v\n", err)
		http.Error(w, "Error marshalling response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Successfully started trial")

	w.Write(bytes)
}
