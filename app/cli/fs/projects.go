package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plandex-cli/term"
	"plandex-cli/types"
	"strings"
)

func GetParentProjectIdsWithPaths(currentUserId string) ([][2]string, error) {
	var parentProjectIds [][2]string
	currentDir := filepath.Dir(Cwd)

	for currentDir != "/" {
		plandexDir := findPlandex(currentDir)
		projectSettingsPath := filepath.Join(plandexDir, "projects-v2.json")
		if _, err := os.Stat(projectSettingsPath); err == nil {
			bytes, err := os.ReadFile(projectSettingsPath)
			if err != nil {
				return nil, fmt.Errorf("error reading projectId file: %s", err)
			}

			var settingsByAccount types.CurrentProjectSettingsByAccount
			err = json.Unmarshal(bytes, &settingsByAccount)

			if err != nil {
				term.OutputErrorAndExit("error unmarshalling projects-v2.json: %v", err)
			}

			settings := settingsByAccount[currentUserId]

			if settings == nil {
				return parentProjectIds, nil
			}

			projectId := string(settings.Id)
			parentProjectIds = append(parentProjectIds, [2]string{currentDir, projectId})
		}
		currentDir = filepath.Dir(currentDir)
	}

	return parentProjectIds, nil
}

func GetChildProjectIdsWithPaths(ctx context.Context, currentUserId string) ([][2]string, error) {
	var childProjectIds [][2]string

	err := filepath.Walk(Cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// if permission denied, skip the path
			if os.IsPermission(err) {
				if info.IsDir() {
					return filepath.SkipDir
				} else {
					return nil
				}
			}

			return err
		}

		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("context timeout")
		default:
		}

		if info.IsDir() && path != Cwd {
			plandexDir := findPlandex(path)
			projectSettingsPath := filepath.Join(plandexDir, "projects-v2.json")
			if _, err := os.Stat(projectSettingsPath); err == nil {
				bytes, err := os.ReadFile(projectSettingsPath)
				if err != nil {
					return fmt.Errorf("error reading projectId file: %s", err)
				}
				var settingsByAccount types.CurrentProjectSettingsByAccount
				err = json.Unmarshal(bytes, &settingsByAccount)

				if err != nil {
					term.OutputErrorAndExit("error unmarshalling projects-v2.json: %v", err)
				}

				settings := settingsByAccount[currentUserId]

				if settings == nil {
					return nil
				}

				projectId := string(settings.Id)
				childProjectIds = append(childProjectIds, [2]string{path, projectId})
			}
		}
		return nil
	})

	if err != nil {
		if err.Error() == "context timeout" {
			return childProjectIds, nil
		}

		return nil, fmt.Errorf("error walking the path %s: %s", Cwd, err)
	}

	return childProjectIds, nil
}
