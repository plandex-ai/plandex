package lib

import (
	"fmt"
	"plandex-cli/fs"
	"plandex-cli/types"
)

type ParseInputPathsParams struct {
	FileOrDirPaths []string
	BaseDir        string
	ProjectPaths   *types.ProjectPaths
	LoadParams     *types.LoadContextParams
}

func ParseInputPaths(params ParseInputPathsParams) ([]string, error) {
	fileOrDirPaths := params.FileOrDirPaths
	baseDir := params.BaseDir
	projectPaths := params.ProjectPaths
	loadParams := params.LoadParams

	resPaths := []string{}

	for path := range projectPaths.AllPaths {
		// see if it's a child of any of the fileOrDirPaths
		found := false
		for _, p := range fileOrDirPaths {
			var err error
			found, err = fs.IsSubpathOf(p, path, baseDir)
			if err != nil {
				return nil, fmt.Errorf("error checking if %s is a subpath of %s: %s", path, p, err)
			}
			if found {
				break
			}
		}

		if !found {
			continue
		}

		if projectPaths.AllDirs[path] {
			if !(loadParams.Recursive || loadParams.NamesOnly || loadParams.DefsOnly) {
				// log.Println("path", path, "info.Name()", info.Name())
				return nil, fmt.Errorf("cannot process directory %s: requires --recursive/-r, --tree, or --map flag", path)
			}

			// calculate directory depth from base
			// depth := strings.Count(path[len(p):], string(filepath.Separator))
			// if params.MaxDepth != -1 && depth > params.MaxDepth {
			// 	return filepath.SkipDir
			// }

			if loadParams.NamesOnly {
				// add directory name to results
				resPaths = append(resPaths, path)
			}
		} else {
			// add file path to results
			resPaths = append(resPaths, path)
		}
	}

	return resPaths, nil
}
