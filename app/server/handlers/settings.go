package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"strings"

	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
)

func GetSettingsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetSettingsHandler")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]

	log.Println("planId: ", planId, "branch: ", branch)

	plan := authorizePlan(w, planId, auth)
	if plan == nil {
		return
	}

	var err error
	ctx, cancel := context.WithCancel(context.Background())
	unlockFn := lockRepo(w, r, auth, db.LockScopeRead, ctx, cancel)
	if unlockFn == nil {
		return
	} else {
		defer (*unlockFn)(err)
	}

	settings, err := db.GetPlanSettings(plan, false)

	if err != nil {
		log.Println("Error getting settings: ", err)
		http.Error(w, "Error getting settings", http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(settings)

	if err != nil {
		log.Println("Error marshalling settings: ", err)
		http.Error(w, "Error marshalling settings", http.StatusInternalServerError)
		return
	}

	log.Println("GetSettingsHandler processed successfully")

	w.Write(bytes)
}

func UpdateSettingsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for UpdateSettingsHandler")

	auth := authenticate(w, r, true)
	if auth == nil {
		return
	}

	vars := mux.Vars(r)
	planId := vars["planId"]
	branch := vars["branch"]

	log.Println("planId: ", planId, "branch: ", branch)

	plan := authorizePlan(w, planId, auth)

	if plan == nil {
		return
	}

	var req shared.UpdateSettingsRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		log.Println("Error decoding request body: ", err)
		http.Error(w, "Error decoding request body", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	unlockFn := lockRepo(w, r, auth, db.LockScopeWrite, ctx, cancel)
	if unlockFn == nil {
		return
	} else {
		defer (*unlockFn)(err)
	}

	originalSettings, err := db.GetPlanSettings(plan, false)

	if err != nil {
		log.Println("Error getting settings: ", err)
		http.Error(w, "Error getting settings", http.StatusInternalServerError)
		return
	}

	err = db.StorePlanSettings(plan, req.Settings)

	if err != nil {
		log.Println("Error storing settings: ", err)
		http.Error(w, "Error storing settings", http.StatusInternalServerError)
		return
	}

	commitMsg := getUpdateCommitMsg(req.Settings, originalSettings)

	err = db.GitAddAndCommit(auth.OrgId, planId, branch, commitMsg)

	if err != nil {
		log.Println("Error committing settings: ", err)
		http.Error(w, "Error committing settings", http.StatusInternalServerError)
		return
	}

	log.Println("UpdateSettingsHandler processed successfully")

}

func getUpdateCommitMsg(settings *shared.PlanSettings, originalSettings *shared.PlanSettings) string {
	var changes []string

	dasherize := func(fieldName string) string {
		matches := regexp.MustCompile("([A-Z][a-z0-9]*)").FindAllStringSubmatch(fieldName, -1)
		var parts []string
		for _, match := range matches {
			parts = append(parts, match[0])
		}
		return strings.ToLower(strings.Join(parts, "-"))
	}

	addChange := func(settingName string, original, current interface{}) {
		changes = append(changes, fmt.Sprintf("- %s: %v -> %v", dasherize(settingName), original, current))
	}

	if settings.MaxConvoTokens != originalSettings.MaxConvoTokens {
		addChange("MaxConvoTokens", originalSettings.MaxConvoTokens, settings.MaxConvoTokens)
	}

	if settings.MaxContextTokens != originalSettings.MaxContextTokens {
		addChange("MaxContextTokens", originalSettings.MaxContextTokens, settings.MaxContextTokens)
	}

	compareModelSet := func(ms *shared.ModelSet, oms *shared.ModelSet) []string {
		var modelSetChanges []string
		if ms == nil && oms == nil {
			return modelSetChanges
		}
		if ms == nil {
			ms = &shared.DefaultModelSet
		}
		if oms == nil {
			oms = &shared.DefaultModelSet
		}

		checkAndAddModelSetChange := func(propertyName string, original, current interface{}) {
			if original != current {
				modelSetChanges = append(modelSetChanges, fmt.Sprintf("- %s: %v -> %v", dasherize(propertyName), original, current))
			}
		}

		// Extend this pattern for other properties within ModelSet as needed
		checkAndAddModelSetChange("Planner.MaxConvoTokens", oms.Planner.MaxConvoTokens, ms.Planner.MaxConvoTokens)

		return modelSetChanges
	}

	modelSetChanges := compareModelSet(settings.ModelSet, originalSettings.ModelSet)
	changes = append(changes, modelSetChanges...)

	if len(changes) == 0 {
		return "No changes to settings"
	}

	return "Updated settings:\n" + strings.Join(changes, "\n")
}
