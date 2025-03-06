package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"reflect"

	shared "plandex-shared"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
)

func GetSettingsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetSettingsHandler")

	auth := Authenticate(w, r, true)
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

	var settings *shared.PlanSettings
	ctx, cancel := context.WithCancel(r.Context())

	err := db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   planId,
		Branch:   branch,
		Reason:   "get settings",
		Scope:    db.LockScopeRead,
		Ctx:      ctx,
		CancelFn: cancel,
	}, func(repo *db.GitRepo) error {
		res, err := db.GetPlanSettings(plan, true)
		if err != nil {
			return err
		}

		settings = res

		return nil
	})

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

	auth := Authenticate(w, r, true)
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

	ctx, cancel := context.WithCancel(r.Context())

	var commitMsg string

	err = db.ExecRepoOperation(db.ExecRepoOperationParams{
		OrgId:    auth.OrgId,
		UserId:   auth.User.Id,
		PlanId:   planId,
		Branch:   branch,
		Reason:   "update settings",
		Scope:    db.LockScopeWrite,
		Ctx:      ctx,
		CancelFn: cancel,
	}, func(repo *db.GitRepo) error {
		originalSettings, err := db.GetPlanSettings(plan, true)

		if err != nil {
			return fmt.Errorf("error getting settings: %v", err)
		}

		// log.Println("Original settings:")
		// spew.Dump(originalSettings)

		// log.Println("req.Settings:")
		// spew.Dump(req.Settings)

		err = db.StorePlanSettings(plan, req.Settings)

		if err != nil {
			return fmt.Errorf("error storing settings: %v", err)
		}

		commitMsg = getUpdateCommitMsg(req.Settings, originalSettings, false)

		err = repo.GitAddAndCommit(branch, commitMsg)

		if err != nil {
			return fmt.Errorf("error committing settings: %v", err)
		}

		return nil
	})

	if err != nil {
		log.Println("Error updating settings: ", err)
		http.Error(w, "Error updating settings", http.StatusInternalServerError)
		return
	}

	res := shared.UpdateSettingsResponse{
		Msg: commitMsg,
	}
	bytes, err := json.Marshal(res)

	if err != nil {
		log.Println("Error marshalling response: ", err)
		http.Error(w, "Error marshalling response", http.StatusInternalServerError)
		return
	}

	w.Write(bytes)

	log.Println("UpdateSettingsHandler processed successfully")

}

func GetDefaultSettingsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for GetDefaultSettingsHandler")

	auth := Authenticate(w, r, true)
	if auth == nil {
		return
	}

	settings, err := db.GetOrgDefaultSettings(auth.OrgId, true)

	if err != nil {
		log.Println("Error getting default settings: ", err)
		http.Error(w, "Error getting default settings", http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(settings)

	if err != nil {
		log.Println("Error marshalling default settings: ", err)
		http.Error(w, "Error marshalling default settings", http.StatusInternalServerError)
		return
	}

	w.Write(bytes)

	log.Println("GetDefaultSettingsHandler processed successfully")
}

func UpdateDefaultSettingsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for UpdateDefaultSettingsHandler")

	auth := Authenticate(w, r, true)

	if auth == nil {
		return
	}

	var req shared.UpdateSettingsRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		log.Println("Error decoding request body: ", err)
		http.Error(w, "Error decoding request body", http.StatusInternalServerError)
		return
	}

	var originalSettings *shared.PlanSettings

	err = db.WithTx(r.Context(), "update default settings", func(tx *sqlx.Tx) error {
		var err error

		originalSettings, err = db.GetOrgDefaultSettingsForUpdate(auth.OrgId, tx, true)

		if err != nil {
			log.Println("Error getting default settings: ", err)
			return fmt.Errorf("error getting default settings: %v", err)
		}

		// log.Println("Original settings:")
		// spew.Dump(originalSettings)

		// log.Println("req.Settings:")
		// spew.Dump(req.Settings)

		if !req.Settings.UpdatedAt.Equal(originalSettings.UpdatedAt) {
			err = fmt.Errorf("default settings have been updated since you last fetched them")
			log.Println("Error updating default settings: ", err)
			return fmt.Errorf("error updating default settings: %v", err)
		}

		err = db.StoreOrgDefaultSettings(auth.OrgId, req.Settings, tx)

		if err != nil {
			log.Println("Error storing default settings: ", err)
			return fmt.Errorf("error storing default settings: %v", err)
		}

		return nil
	})

	if err != nil {
		log.Println("Error updating default settings: ", err)
		http.Error(w, "Error updating default settings", http.StatusInternalServerError)
		return
	}

	commitMsg := getUpdateCommitMsg(req.Settings, originalSettings, true)

	res := shared.UpdateSettingsResponse{
		Msg: commitMsg,
	}
	bytes, err := json.Marshal(res)

	if err != nil {
		log.Println("Error marshalling response: ", err)
		http.Error(w, "Error marshalling response", http.StatusInternalServerError)
		return
	}

	w.Write(bytes)

	log.Println("UpdateDefaultSettingsHandler processed successfully")
}

func getUpdateCommitMsg(settings *shared.PlanSettings, originalSettings *shared.PlanSettings, isOrgDefault bool) string {
	// log.Println("Comparing settings")
	// log.Println("Original:")
	// spew.Dump(originalSettings)
	// log.Println("New:")
	// spew.Dump(settings)

	var changes []string
	compareAny(originalSettings, settings, "", &changes)

	if len(changes) == 0 {
		return "No changes to settings"
	}

	// log.Println("Changes to settings:", strings.Join(changes, "\n"))
	var s string
	if isOrgDefault {
		s = "⚙️  Updated org-wide default settings:"
	} else {
		s = "⚙️  Updated model settings:"
	}

	for _, change := range changes {
		s += "\n" + "  • " + change
	}

	return s
}

func compareAny(a, b interface{}, path string, changes *[]string) {
	aVal, bVal := reflect.ValueOf(a), reflect.ValueOf(b)

	// Check validity as before
	if !aVal.IsValid() || !bVal.IsValid() {
		return
	}

	// Dereference pointers
	if aVal.Kind() == reflect.Ptr && bVal.Kind() == reflect.Ptr {
		aVal = aVal.Elem()
		bVal = bVal.Elem()
	}

	// Check validity after dereferencing
	if !aVal.IsValid() || !bVal.IsValid() {
		return
	}

	// Ensure we can safely call Interface()
	if !aVal.CanInterface() || !bVal.CanInterface() {
		return
	}

	if reflect.DeepEqual(aVal.Interface(), bVal.Interface()) {
		return // No difference found
	}

	switch aVal.Kind() {
	case reflect.Struct:
		for i := 0; i < aVal.NumField(); i++ {
			field := aVal.Type().Field(i)
			if !field.IsExported() {
				continue // Skip unexported fields
			}
			fieldName := field.Name
			dasherizedName := shared.Dasherize(fieldName)

			updatedPath := path
			if !(dasherizedName == "model-set" ||
				dasherizedName == "model-role-config" ||
				dasherizedName == "base-model-config" ||
				dasherizedName == "planner-model-config" ||
				dasherizedName == "task-model-config") {
				if updatedPath != "" {
					updatedPath = updatedPath + "." + dasherizedName
				} else {
					if dasherizedName == "model-overrides" {
						dasherizedName = "overrides"
					}
					updatedPath = dasherizedName
				}
			}

			compareAny(aVal.Field(i).Interface(), bVal.Field(i).Interface(), updatedPath, changes)
		}
	default:
		var aStr, bStr string
		if aVal.IsValid() {
			aStr = fmt.Sprintf("%v", aVal.Interface())
		} else {
			aStr = "no override"
		}

		if bVal.IsValid() {
			bStr = fmt.Sprintf("%v", bVal.Interface())
		} else {
			bStr = "no override"
		}

		change := fmt.Sprintf("%s | %v → %v", path, aStr, bStr)
		*changes = append(*changes, change)
	}
}
