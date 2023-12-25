package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"plandex/api"
	"plandex/term"
	"plandex/types"
	"plandex/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
)

func MustCheckOutdatedContextWithOutput() {
	s := term.Spinner
	start := time.Now()
	s.Prefix = "🔬 Checking context... "
	s.Start()

	stopSpinner := func() {
		s.Stop()
		term.ClearCurrentLine()
	}

	outdatedRes, err := CheckOutdatedContext()
	if err != nil {
		stopSpinner()
		panic(fmt.Errorf("failed to check outdated context: %s", err))
	}

	elapsed := time.Since(start)
	if elapsed < 700*time.Millisecond {
		time.Sleep(700*time.Millisecond - elapsed)
	}

	stopSpinner()

	if len(outdatedRes.UpdatedContexts) == 0 {
		fmt.Println("✅ Context is up to date")
		return
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
	color.New(color.FgHiCyan, color.Bold).Printf("%s in context %s modified 👇\n\n", msg, phrase)

	tableString := tableForContextOutdated(outdatedRes)
	fmt.Println(tableString)

	fmt.Println()

	confirmed, canceled, err := term.ConfirmYesNoCancel("Update context now?")

	if err != nil {
		panic(fmt.Errorf("failed to get user input: %s", err))
	}

	if confirmed {
		MustUpdateContextWithOuput()
	}

	if canceled {
		os.Exit(0)
	}

}

func MustUpdateContextWithOuput() {
	timeStart := time.Now()

	s := term.Spinner
	s.Prefix = "🔄 Updating context... "
	s.Start()

	stopFn := func() {
		elapsed := time.Since(timeStart)
		if elapsed < 700*time.Millisecond {
			time.Sleep(700*time.Millisecond - elapsed)
		}
		s.Stop()
		term.ClearCurrentLine()
	}

	updateRes, err := UpdateContext()

	if err != nil {
		stopFn()
		fmt.Fprintln(os.Stderr, "Error updating context:", err)
		os.Exit(1)
	}

	stopFn()

	fmt.Println("✅ " + updateRes.Msg)

}

func UpdateContext() (*types.ContextOutdatedResult, error) {
	return checkOutdatedAndMaybeUpdateContext(true)
}

func CheckOutdatedContext() (*types.ContextOutdatedResult, error) {
	return checkOutdatedAndMaybeUpdateContext(false)
}

func checkOutdatedAndMaybeUpdateContext(doUpdate bool) (*types.ContextOutdatedResult, error) {

	contexts, err := api.Client.ListContext(CurrentPlanId)
	if err != nil {
		return nil, fmt.Errorf("error retrieving context: %w", err)
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

	for _, context := range contexts {

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
					NamesOnly: true,
				})

				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					errs = append(errs, fmt.Errorf("failed to get the directory tree %s: %v", context.FilePath, err))
					return
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

	if len(req) == 0 {
		return &types.ContextOutdatedResult{
			Msg: "Context is up to date",
		}, nil
	} else if doUpdate {
		res, err := api.Client.UpdateContext(CurrentPlanId, req)
		if err != nil {
			return nil, fmt.Errorf("failed to update context: %w", err)
		}
		msg = res.Msg
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
