package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"plandex/api"
	"plandex/fs"
	"plandex/term"
	"plandex/types"
	"plandex/url"
	"strconv"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
)

func MustCheckOutdatedContext(quiet bool, maybeContexts []*shared.Context) (contextOutdated, updated bool) {
	if !quiet {
		term.StartSpinner("🔬 Checking context...")
	}

	outdatedRes, err := CheckOutdatedContext(maybeContexts)
	if err != nil {
		term.StopSpinner()
		term.OutputErrorAndExit("failed to check outdated context: %s", err)
	}

	term.StopSpinner()

	if len(outdatedRes.UpdatedContexts) == 0 {
		if !quiet {
			fmt.Println("✅ Context is up to date")
		}
		return false, false
	}
	types := []string{}
	if outdatedRes.NumFiles > 0 {
		lbl := "file"
		if outdatedRes.NumFiles > 1 {
			lbl = "files"
		}
		lbl = strconv.Itoa(outdatedRes.NumFiles) + " " + lbl
		types = append(types, lbl)
	}
	if outdatedRes.NumUrls > 0 {
		lbl := "url"
		if outdatedRes.NumUrls > 1 {
			lbl = "urls"
		}
		lbl = strconv.Itoa(outdatedRes.NumUrls) + " " + lbl
		types = append(types, lbl)
	}
	if outdatedRes.NumTrees > 0 {
		lbl := "directory tree"
		if outdatedRes.NumTrees > 1 {
			lbl = "directory trees"
		}
		lbl = strconv.Itoa(outdatedRes.NumTrees) + " " + lbl
		types = append(types, lbl)
	}

	var msg string
	if len(types) <= 2 {
		msg += strings.Join(types, " and ")
	} else {
		for i, add := range types {
			if i == len(types)-1 {
				msg += ", and " + add
			} else {
				msg += ", " + add
			}
		}
	}

	phrase := "have been"
	if len(outdatedRes.UpdatedContexts) == 1 {
		phrase = "has been"
	}
	color.New(term.ColorHiCyan, color.Bold).Printf("%s in context %s modified 👇\n\n", msg, phrase)

	tableString := tableForContextOutdated(outdatedRes)
	fmt.Println(tableString)

	fmt.Println()

	var confirmed bool

	confirmed, err = term.ConfirmYesNo("Update context now?")

	if err != nil {
		term.OutputErrorAndExit("failed to get user input: %s", err)
	}

	if confirmed {
		MustUpdateContext(maybeContexts)
		return true, true
	} else {
		return true, false
	}

}

func MustUpdateContext(maybeContexts []*shared.Context) {
	term.StartSpinner("🔄 Updating context...")

	updateRes, err := UpdateContext(maybeContexts)

	if err != nil {
		term.StopSpinner()
		term.OutputErrorAndExit("Error updating context: %v", err)
	}

	term.StopSpinner()

	fmt.Println("✅ " + updateRes.Msg)

}

func UpdateContext(maybeContexts []*shared.Context) (*types.ContextOutdatedResult, error) {
	return checkOutdatedAndMaybeUpdateContext(true, maybeContexts)
}

func CheckOutdatedContext(maybeContexts []*shared.Context) (*types.ContextOutdatedResult, error) {
	return checkOutdatedAndMaybeUpdateContext(false, maybeContexts)
}

func checkOutdatedAndMaybeUpdateContext(doUpdate bool, maybeContexts []*shared.Context) (*types.ContextOutdatedResult, error) {
	var contexts []*shared.Context

	if maybeContexts == nil {
		var apiErr *shared.ApiError
		contexts, apiErr = api.Client.ListContext(CurrentPlanId, CurrentBranch)
		if apiErr != nil {
			return nil, fmt.Errorf("error retrieving context: %v", apiErr)
		}
	} else {
		contexts = maybeContexts
	}

	var errs []error

	req := shared.UpdateContextRequest{}
	var updatedContexts []*shared.Context
	var tokenDiffsById = map[string]int{}
	var numFiles int
	var numUrls int
	var numTrees int
	var mu sync.Mutex
	var wg sync.WaitGroup
	contextsById := map[string]*shared.Context{}

	var paths *fs.ProjectPaths
	var hasDirectoryTreeWithIgnoredPaths bool

	for _, context := range contexts {
		if context.ContextType == shared.ContextDirectoryTreeType && !context.ForceSkipIgnore {
			hasDirectoryTreeWithIgnoredPaths = true
			break
		}
	}

	if hasDirectoryTreeWithIgnoredPaths {
		baseDir := fs.GetBaseDirForContexts(contexts)
		var err error
		paths, err = fs.GetProjectPaths(baseDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get project paths: %v", err)
		}
	}

	for _, context := range contexts {
		contextsById[context.Id] = context

		if context.ContextType == shared.ContextFileType {
			wg.Add(1)
			go func(context *shared.Context) {
				defer wg.Done()
				fileContent, err := os.ReadFile(context.FilePath)

				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to read the file %s: %v", context.FilePath, err))
					return
				}

				hash := sha256.Sum256(fileContent)
				sha := hex.EncodeToString(hash[:])

				if sha != context.Sha {
					body := string(fileContent)

					numTokens, err := shared.GetNumTokens(body)
					if err != nil {
						errs = append(errs, fmt.Errorf("failed to get the number of tokens in the file %s: %v", context.FilePath, err))
						return
					}
					tokenDiffsById[context.Id] = numTokens - context.NumTokens

					numFiles++
					updatedContexts = append(updatedContexts, context)

					req[context.Id] = &shared.UpdateContextParams{
						Body: body,
					}
				}
			}(context)

		} else if context.ContextType == shared.ContextDirectoryTreeType {
			wg.Add(1)
			go func(context *shared.Context) {
				defer wg.Done()
				flattenedPaths, err := ParseInputPaths([]string{context.FilePath}, &types.LoadContextParams{
					NamesOnly:       true,
					ForceSkipIgnore: context.ForceSkipIgnore,
				})

				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					errs = append(errs, fmt.Errorf("failed to get the directory tree %s: %v", context.FilePath, err))
					return
				}

				if !context.ForceSkipIgnore {
					if paths == nil {
						errs = append(errs, fmt.Errorf("project paths are nil"))
						return
					}

					var filteredPaths []string
					for _, path := range flattenedPaths {
						if _, ok := paths.ActivePaths[path]; ok {
							filteredPaths = append(filteredPaths, path)
						}
					}
					flattenedPaths = filteredPaths
				}

				body := strings.Join(flattenedPaths, "\n")
				bytes := []byte(body)

				hash := sha256.Sum256(bytes)
				sha := hex.EncodeToString(hash[:])

				if sha != context.Sha {
					numTokens, err := shared.GetNumTokens(body)
					if err != nil {
						errs = append(errs, fmt.Errorf("failed to get the number of tokens in the file %s: %v", context.FilePath, err))
						return
					}
					tokenDiffsById[context.Id] = numTokens - context.NumTokens

					numTrees++
					updatedContexts = append(updatedContexts, context)
					req[context.Id] = &shared.UpdateContextParams{
						Body: body,
					}
				}
			}(context)

		} else if context.ContextType == shared.ContextURLType {
			wg.Add(1)
			go func(context *shared.Context) {
				defer wg.Done()
				body, err := url.FetchURLContent(context.Url)

				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					errs = append(errs, fmt.Errorf("failed to fetch the URL %s: %v", context.Url, err))
					return
				}

				hash := sha256.Sum256([]byte(body))
				sha := hex.EncodeToString(hash[:])

				if sha != context.Sha {
					numTokens, err := shared.GetNumTokens(body)
					if err != nil {
						errs = append(errs, fmt.Errorf("failed to get the number of tokens in the file %s: %v", context.FilePath, err))
						return
					}
					tokenDiffsById[context.Id] = numTokens - context.NumTokens

					numUrls++
					updatedContexts = append(updatedContexts, context)
					req[context.Id] = &shared.UpdateContextParams{
						Body: body,
					}
				}

			}(context)
		}
	}

	wg.Wait()

	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to check context outdated: %v", errs)
	}

	var msg string
	var hasConflicts bool

	if len(req) == 0 {
		return &types.ContextOutdatedResult{
			Msg: "Context is up to date",
		}, nil
	} else if doUpdate {
		filesToLoad := map[string]string{}
		for id := range req {
			context := contextsById[id]
			if context.ContextType == shared.ContextFileType {
				filesToLoad[context.FilePath] = context.Body
			}
		}

		var err error
		hasConflicts, err = checkContextConflicts(filesToLoad)
		if err != nil {
			return nil, fmt.Errorf("failed to check context conflicts: %v", err)
		}

		res, apiErr := api.Client.UpdateContext(CurrentPlanId, CurrentBranch, req)
		if apiErr != nil {
			return nil, fmt.Errorf("failed to update context: %v", apiErr)
		}
		msg = res.Msg
	}

	if hasConflicts {
		term.StartSpinner("🏗️  Starting build...")
		_, err := buildPlanInlineFn(nil) // don't pass in outdated contexts -- nil value causes them to be refetched, which is what we want since they were just updated

		if err != nil {
			return nil, fmt.Errorf("failed to build plan: %v", err)
		}

		fmt.Println()
	}

	return &types.ContextOutdatedResult{
		Msg:             msg,
		UpdatedContexts: updatedContexts,
		TokenDiffsById:  tokenDiffsById,
		NumFiles:        numFiles,
		NumUrls:         numUrls,
		NumTrees:        numTrees,
	}, nil
}

func tableForContextOutdated(updateRes *types.ContextOutdatedResult) string {
	updatedContexts := updateRes.UpdatedContexts
	tokenDiffsById := updateRes.TokenDiffsById

	if len(updatedContexts) == 0 {
		return ""
	}

	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Name", "Type", "🪙"})
	table.SetAutoWrapText(false)

	for _, context := range updatedContexts {
		t, icon := GetContextTypeAndIcon(context)
		diff := tokenDiffsById[context.Id]

		diffStr := "+" + strconv.Itoa(diff)
		tableColor := tablewriter.FgHiGreenColor

		if diff < 0 {
			diffStr = strconv.Itoa(diff)
			tableColor = tablewriter.FgHiRedColor
		}

		row := []string{
			" " + icon + " " + context.Name,
			t,
			diffStr,
		}

		table.Rich(row, []tablewriter.Colors{
			{tableColor, tablewriter.Bold},
			{tableColor},
			{tableColor},
		})
	}

	table.Render()

	return tableString.String()
}
