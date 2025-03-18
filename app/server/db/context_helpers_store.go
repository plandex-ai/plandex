package db

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	shared "plandex-shared"

	"github.com/google/uuid"
)

func StoreContext(context *Context, skipMapCache bool) error {
	// log.Println("StoreContext - Storing context", context.Id, context.Name, context.ContextType)
	// log.Println("StoreContext - Num tokens", context.NumTokens)

	contextDir := getPlanContextDir(context.OrgId, context.PlanId)

	err := os.MkdirAll(contextDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error creating context dir: %v", err)
	}

	ts := time.Now().UTC()
	if context.Id == "" {
		context.Id = uuid.New().String()
		context.CreatedAt = ts
	}
	context.UpdatedAt = ts
	context.BodySize = int64(len(context.Body))

	metaFilename := context.Id + ".meta"
	metaPath := filepath.Join(contextDir, metaFilename)

	originalBody := context.Body
	originalBody = strings.ReplaceAll(originalBody, "\\`\\`\\`", "\\\\`\\\\`\\\\`")
	originalBody = strings.ReplaceAll(originalBody, "```", "\\`\\`\\`")

	bodyFilename := context.Id + ".body"
	bodyPath := filepath.Join(contextDir, bodyFilename)
	body := []byte(originalBody)
	context.Body = ""

	originalMapParts := context.MapParts
	var mapPath string
	var mapBytes []byte
	if len(context.MapParts) > 0 {
		mapFilename := context.Id + ".map-parts"
		mapPath = filepath.Join(contextDir, mapFilename)
		mapBytes, err = json.MarshalIndent(context.MapParts, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal map parts: %v", err)
		}
		context.MapParts = nil
	}

	// Convert the ModelContextPart to JSON
	data, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context context: %v", err)
	}

	// Write the body to the file
	if err = os.WriteFile(bodyPath, body, 0644); err != nil {
		return fmt.Errorf("failed to write context body to file %s: %v", bodyPath, err)
	}

	// Write the meta data to the file
	if err = os.WriteFile(metaPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write context meta to file %s: %v", metaPath, err)
	}

	if mapPath != "" {
		if err = os.WriteFile(mapPath, mapBytes, 0644); err != nil {
			return fmt.Errorf("failed to write context map to file %s: %v", mapPath, err)
		}
	}

	context.Body = originalBody
	context.MapParts = originalMapParts

	if mapPath != "" && !skipMapCache {
		log.Println("StoreContext - context.MapParts length", len(context.MapParts))

		mapCacheDir := getProjectMapCacheDir(context.OrgId, context.ProjectId)

		// ensure map cache dir exists
		err = os.MkdirAll(mapCacheDir, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error creating map cache dir: %v", err)
		}

		filePathHash := md5.Sum([]byte(context.FilePath))
		filePathHashStr := hex.EncodeToString(filePathHash[:])

		mapCachePath := filepath.Join(mapCacheDir, filePathHashStr+".json")

		log.Println("StoreContext - mapCachePath", mapCachePath)

		cachedContext := Context{
			ContextType: shared.ContextMapType,
			FilePath:    context.FilePath,
			Name:        context.Name,
			Body:        context.Body,
			NumTokens:   context.NumTokens,
			MapParts:    context.MapParts,
			MapShas:     context.MapShas,
			MapTokens:   context.MapTokens,
			MapSizes:    context.MapSizes,
			UpdatedAt:   context.UpdatedAt,
		}

		cachedContextBytes, err := json.MarshalIndent(cachedContext, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal cached context: %v", err)
		}

		err = os.WriteFile(mapCachePath, cachedContextBytes, 0644)
		if err != nil {
			return fmt.Errorf("failed to write context map to file %s: %v", mapCachePath, err)
		}
	}

	return nil
}
