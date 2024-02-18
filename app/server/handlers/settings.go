package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"plandex-server/db"
	"reflect"
	"regexp"
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
	compareAny(settings, originalSettings, "settings", &changes)

	if len(changes) == 0 {
		return "No changes to settings"
	}

	return "Updated settings:\n" + strings.Join(changes, "\n")
}

func compareAny(a, b interface{}, path string, changes *[]string) {
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

	// Handle nil for slices, maps, ptrs
	if a == nil || b == nil || (aVal.Kind() != bVal.Kind()) {
		change := fmt.Sprintf("- %s: %v -> %v", path, a, b)
		*changes = append(*changes, change)
		return
	}

	switch aVal.Kind() {
	case reflect.Struct:
		for i := 0; i < aVal.NumField(); i++ {
			fieldName := aVal.Type().Field(i).Name
			matches := regexp.MustCompile("([A-Z[a-z0-9]*)").FindAllStringSubmatch(fieldName, -1)
			var parts []string
			for _, match := range matches {
				parts = append(parts, match[0])
			}
			dasherizedName := strings.ToLower(strings.Join(parts, "-"))
			compareAny(aVal.Field(i).Interface(), bVal.Field(i).Interface(),
				path+"."+dasherizedName, changes)
		}
	case reflect.Slice, reflect.Array:
		if aVal.Len() != bVal.Len() {
			change := fmt.Sprintf("- %s: %v -> %v", path, aVal.Interface(),
				bVal.Interface())
			*changes = append(*changes, change)
			return
		}
		for i := 0; i < aVal.Len(); i++ {
			compareAny(aVal.Index(i).Interface(), bVal.Index(i).Interface(),
				fmt.Sprintf("%s[%d]", path, i), changes)
		}
	default:
		change := fmt.Sprintf("- %s: %v -> %v", path, aVal.Interface(),
			bVal.Interface())
		*changes = append(*changes, change)
	}
}
