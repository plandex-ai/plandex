package db

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	shared "plandex-shared"
)

func GetCachedMap(orgId, projectId, filePath string) (*Context, error) {
	mapCacheDir := getProjectMapCacheDir(orgId, projectId)

	filePathHash := md5.Sum([]byte(filePath))
	filePathHashStr := hex.EncodeToString(filePathHash[:])

	mapCachePath := filepath.Join(mapCacheDir, filePathHashStr+".json")

	log.Println("GetCachedMap - mapCachePath", mapCachePath)

	mapCacheBytes, err := os.ReadFile(mapCachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("error reading cached map: %v", err)
	}

	var context Context
	err = json.Unmarshal(mapCacheBytes, &context)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling cached map: %v", err)
	}

	return &context, nil
}

type CachedMap struct {
	MapParts  shared.FileMapBodies
	MapShas   map[string]string
	MapTokens map[string]int
	MapSizes  map[string]int64
}
