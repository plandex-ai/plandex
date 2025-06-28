package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
		res, err := db.GetPlanSettings(plan)
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

	if req.ModelPackName == "" && req.ModelPack == nil {
		log.Println("No model pack name or model pack provided")
		http.Error(w, "No model pack name or model pack provided", http.StatusBadRequest)
		return
	}

	if req.ModelPackName != "" {
		if mp, builtIn := shared.BuiltInModelPacksByName[req.ModelPackName]; builtIn {
			if os.Getenv("IS_CLOUD") != "" && mp.LocalProvider != "" {
				msg := fmt.Sprintf("Built-in local model pack %s can't be used on Plandex Cloud", req.ModelPackName)
				log.Println(msg)
				http.Error(w, msg, http.StatusUnprocessableEntity)
				return
			}
		}
	}

	if req.ModelPack != nil {
		if req.ModelPack.LocalProvider != "" {
			msg := fmt.Sprintf("Local model pack %s can't be used on Plandex Cloud", req.ModelPack.Name)
			log.Println(msg)
			http.Error(w, msg, http.StatusUnprocessableEntity)
			return
		}

		ids := req.ModelPack.ToModelPackSchema().AllModelIds()
		for _, id := range ids {
			bm, builtIn := shared.BuiltInBaseModelsById[id]
			if builtIn && os.Getenv("IS_CLOUD") != "" && bm.IsLocalOnly() {
				msg := fmt.Sprintf("Built-in local model %s can't be used on Plandex Cloud", id)
				log.Println(msg)
				http.Error(w, msg, http.StatusUnprocessableEntity)
				return
			}
		}
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
		originalSettings, err := db.GetPlanSettings(plan)

		if err != nil {
			return fmt.Errorf("error getting settings: %v", err)
		}

		settings, err := originalSettings.DeepCopy()
		if err != nil {
			return fmt.Errorf("error copying settings: %v", err)
		}

		if req.ModelPackName != "" {
			settings.SetModelPackByName(req.ModelPackName)
		} else if req.ModelPack != nil {
			settings.SetCustomModelPack(req.ModelPack)
		} else {
			return fmt.Errorf("no model pack name or model pack provided")
		}

		// log.Println("Original settings:")
		// spew.Dump(originalSettings)

		// log.Println("req.Settings:")
		// spew.Dump(req.Settings)

		err = db.StorePlanSettings(plan, *settings)

		if err != nil {
			return fmt.Errorf("error storing settings: %v", err)
		}

		commitMsg = getUpdateCommitMsg(settings, originalSettings, false)

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

	settings, err := db.GetOrgDefaultSettings(auth.OrgId)

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

	if req.ModelPackName == "" && req.ModelPack == nil {
		log.Println("No model pack name or model pack provided")
		http.Error(w, "No model pack name or model pack provided", http.StatusBadRequest)
		return
	}

	var originalSettings *shared.PlanSettings
	var settings *shared.PlanSettings

	err = db.WithTx(r.Context(), "update default settings", func(tx *sqlx.Tx) error {
		var err error

		originalSettings, err = db.GetOrgDefaultSettingsForUpdate(auth.OrgId, tx)

		if err != nil {
			log.Println("Error getting default settings: ", err)
			return fmt.Errorf("error getting default settings: %v", err)
		}

		settings, err = originalSettings.DeepCopy()
		if err != nil {
			return fmt.Errorf("error copying settings: %v", err)
		}

		if req.ModelPackName != "" {
			settings.SetModelPackByName(req.ModelPackName)
		} else if req.ModelPack != nil {
			settings.SetCustomModelPack(req.ModelPack)
		} else {
			return fmt.Errorf("no model pack name or model pack provided")
		}

		// log.Println("Original settings:")
		// spew.Dump(originalSettings)

		// log.Println("req.Settings:")
		// spew.Dump(req.Settings)

		err = db.StoreOrgDefaultSettings(auth.OrgId, settings, tx)

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

	commitMsg := getUpdateCommitMsg(settings, originalSettings, true)

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

	// log.Println("Changes to settings:", strings.Join(changes, "\n"))
	var s string
	if isOrgDefault {
		s = "⚙️  Updated org-wide default settings:"
	} else {
		s = "⚙️  Updated model settings:"
	}

	var changes []string
	changes = compareSettings(originalSettings, settings, changes)

	if len(changes) == 0 {
		return "No changes to settings"
	}

	for _, change := range changes {
		s += "\n" + "  • " + change
	}

	return s
}

func compareSettings(original, updated *shared.PlanSettings, changes []string) []string {
	if updated.ModelPackName != "" {
		originalName := "custom"
		if original.ModelPackName != "" {
			originalName = original.ModelPackName
		}
		changes = append(changes, fmt.Sprintf("model-pack | %v → %v", originalName, updated.ModelPackName))
	} else if updated.ModelPack != nil {
		if original.ModelPack == nil {
			changes = append(changes, fmt.Sprintf("model-pack | %v → %v", original.ModelPackName, "custom"))
		}

		changes = compareAny(original.GetModelPack().ToModelPackSchema().ModelPackSchemaRoles, updated.GetModelPack().ToModelPackSchema().ModelPackSchemaRoles, "", changes)
	}

	return changes
}

func compareAny(a, b interface{}, path string, changes []string) []string {
	aVal, bVal := reflect.ValueOf(a), reflect.ValueOf(b)

	if !aVal.IsValid() && !bVal.IsValid() {
		return changes
	}

	// Pointer / nil handling – BEFORE dereferencing
	if aVal.Kind() == reflect.Ptr || bVal.Kind() == reflect.Ptr {
		// both nil → nothing
		if (aVal.Kind() == reflect.Ptr && aVal.IsNil()) &&
			(bVal.Kind() == reflect.Ptr && bVal.IsNil()) {
			return changes
		}

		// one nil, one non-nil → record diff
		if (aVal.Kind() == reflect.Ptr && aVal.IsNil()) ||
			(bVal.Kind() == reflect.Ptr && bVal.IsNil()) {
			aStr := "none"
			bStr := "none"
			if aVal.Kind() != reflect.Ptr || !aVal.IsNil() {
				aStr = short(aVal)
			}
			if bVal.Kind() != reflect.Ptr || !bVal.IsNil() {
				bStr = short(bVal)
			}
			changes = append(changes, fmt.Sprintf("%s | %s → %s", path, aStr, bStr))
			return changes
		}

		// both non-nil pointers → safe to dereference
		if aVal.Kind() == reflect.Ptr {
			aVal = aVal.Elem()
		}
		if bVal.Kind() == reflect.Ptr {
			bVal = bVal.Elem()
		}
	}

	// Check again after dereferencing
	if !aVal.IsValid() && !bVal.IsValid() {
		return changes
	}

	// One side nil → record diff and stop
	if !aVal.IsValid() || !bVal.IsValid() {
		var aStr, bStr string
		if aVal.IsValid() {
			aStr = short(aVal)
		} else {
			aStr = "none"
		}
		if bVal.IsValid() {
			bStr = short(bVal)
		} else {
			bStr = "none"
		}
		changes = append(changes, fmt.Sprintf("%s | %s → %s", path, aStr, bStr))
		return changes
	}

	// Ensure we can safely call Interface()
	if !aVal.CanInterface() || !bVal.CanInterface() {
		return changes
	}

	if reflect.DeepEqual(aVal.Interface(), bVal.Interface()) {
		return changes // No difference found
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

			changes = compareAny(aVal.Field(i).Interface(), bVal.Field(i).Interface(), updatedPath, changes)
		}
	default:
		var aStr, bStr string
		if aVal.IsValid() {
			aStr = short(aVal)
		} else {
			aStr = "no override"
		}

		if bVal.IsValid() {
			bStr = short(bVal)
		} else {
			bStr = "no override"
		}

		change := fmt.Sprintf("%s | %v → %v", path, aStr, bStr)
		changes = append(changes, change)
	}

	return changes
}

func short(v reflect.Value) string {
	if !v.IsValid() {
		return "none"
	}

	// If it’s a pointer, follow it once
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "none"
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", v.Float())
	case reflect.Struct:
		// Special-case ModelRoleConfigSchema: show the ModelId only
		if f := v.FieldByName("ModelId"); f.IsValid() && f.Kind() == reflect.String {
			return f.String()
		}
		return fmt.Sprintf("%T", v.Interface()) // fall-back: just the type name
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}
