package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"reflect"
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

	commitMsg := getUpdateCommitMsg(req.Settings, originalSettings)

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

func getUpdateCommitMsg(settings *shared.PlanSettings, originalSettings *shared.PlanSettings) string {
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

	s := "⚙️  Updated model settings:"

	for _, change := range changes {
		s += "\n" + "  • " + change
	}

	return s
}

func compareAny(a, b interface{}, path string, changes *[]string) {
	// log.Println("Comparing", path)
	// log.Println("a")
	// spew.Dump(a)
	// log.Println("b")
	// spew.Dump(b)

	if strings.HasSuffix(path, "updated-at") ||
		strings.HasSuffix(path, "open-ai-response-format") {
		return
	}

	if reflect.DeepEqual(a, b) {
		return
	}

	aVal, bVal := reflect.ValueOf(a), reflect.ValueOf(b)
	if aVal.Kind() == reflect.Ptr {
		aVal = aVal.Elem()
	}
	if bVal.Kind() == reflect.Ptr {
		bVal = bVal.Elem()
	}

	// log.Println("Comparing", path, aVal.Kind(), bVal.Kind())
	// log.Println("aVal", aVal)
	// log.Println("bVal", bVal)

	switch aVal.Kind() {
	case reflect.Struct:
		for i := 0; i < aVal.NumField(); i++ {
			fieldName := aVal.Type().Field(i).Name
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

			// log.Println("field", fieldName, "updatedPath", updatedPath)

			compareAny(aVal.Field(i).Interface(), bVal.Field(i).Interface(),
				updatedPath, changes)
		}
	default:
		var a string
		var b string

		if aVal.IsValid() {
			a = fmt.Sprintf("%v", aVal.Interface())
		} else {
			a = "no override"
		}

		if bVal.IsValid() {
			b = fmt.Sprintf("%v", bVal.Interface())
		} else {
			b = "no override"
		}

		change := fmt.Sprintf("%s | %v → %v", path, a, b)
		*changes = append(*changes, change)
	}
}
