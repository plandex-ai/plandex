package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"reflect"

	"github.com/gorilla/mux"
	"github.com/plandex/plandex/shared"
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

	var err error
	ctx, cancel := context.WithCancel(context.Background())
	unlockFn := lockRepo(w, r, auth, db.LockScopeRead, ctx, cancel, true)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	settings, err := db.GetPlanSettings(plan, true)

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

	ctx, cancel := context.WithCancel(context.Background())
	unlockFn := lockRepo(w, r, auth, db.LockScopeWrite, ctx, cancel, true)
	if unlockFn == nil {
		return
	} else {
		defer func() {
			(*unlockFn)(err)
		}()
	}

	originalSettings, err := db.GetPlanSettings(plan, true)

	if err != nil {
		log.Println("Error getting settings: ", err)
		http.Error(w, "Error getting settings", http.StatusInternalServerError)
		return
	}

	// log.Println("Original settings:")
	// spew.Dump(originalSettings)

	// log.Println("req.Settings:")
	// spew.Dump(req.Settings)

	err = db.StorePlanSettings(plan, req.Settings)

	if err != nil {
		log.Println("Error storing settings: ", err)
		http.Error(w, "Error storing settings", http.StatusInternalServerError)
		return
	}

	commitMsg := getUpdateCommitMsg(req.Settings, originalSettings, false)

	err = db.GitAddAndCommit(auth.OrgId, planId, branch, commitMsg)

	if err != nil {
		log.Println("Error committing settings: ", err)
		http.Error(w, "Error committing settings", http.StatusInternalServerError)
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

	tx, err := db.Conn.Beginx()

	if err != nil {
		log.Println("Error starting transaction: ", err)
		http.Error(w, "Error starting transaction", http.StatusInternalServerError)
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

	originalSettings, err := db.GetOrgDefaultSettingsForUpdate(auth.OrgId, tx, true)

	if err != nil {
		log.Println("Error getting default settings: ", err)
		http.Error(w, "Error getting default settings", http.StatusInternalServerError)
		return
	}

	// log.Println("Original settings:")
	// spew.Dump(originalSettings)

	// log.Println("req.Settings:")
	// spew.Dump(req.Settings)

	if !req.Settings.UpdatedAt.Equal(originalSettings.UpdatedAt) {
		err = fmt.Errorf("default settings have been updated since you last fetched them")
		log.Println("Error updating default settings: ", err)
		http.Error(w, "Error updating default settings: "+err.Error(), http.StatusBadRequest)
		return
	}

	err = db.StoreOrgDefaultSettings(auth.OrgId, req.Settings, tx)

	if err != nil {
		log.Println("Error storing default settings: ", err)
		http.Error(w, "Error storing default settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = tx.Commit()

	if err != nil {
		log.Println("Error committing transaction: ", err)
		http.Error(w, "Error committing transaction: "+err.Error(), http.StatusInternalServerError)
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
