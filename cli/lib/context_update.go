package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"plandex/term"
	"plandex/types"
	"plandex/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/olekukonko/tablewriter"
	"github.com/plandex/plandex/shared"
)

type contextUpdate struct {
	Path string
	Part *shared.ModelContextPart
}

type updateRes struct {
	UpdatedParts     []*shared.ModelContextPart
	TokenDiffsByName map[string]int
	TokensDiff       int
	TotalTokens      int
	MaxExceeded      bool
	NumFiles         int
	NumUrls          int
	NumTrees         int
}

func MustUpdateContextWithOuput() *updateRes {
	timeStart := time.Now()

	s := spinner.New(spinner.CharSets[33], 100*time.Millisecond)
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

	maxExceeded := updateRes.MaxExceeded
	updatedParts := updateRes.UpdatedParts
	tokensDiff := updateRes.TokensDiff
	totalTokens := updateRes.TotalTokens
	numFiles := updateRes.NumFiles
	numTrees := updateRes.NumTrees
	numUrls := updateRes.NumUrls

	if maxExceeded {
		stopFn()
		overage := totalTokens - shared.MaxContextTokens
		fmt.Printf("🚨 Update would add %d 🪙 and exceed token limit (%d) by %d 🪙", tokensDiff, shared.MaxContextTokens, overage)
		os.Exit(1)
	}

	if len(updatedParts) == 0 {
		stopFn()
		fmt.Println("✅ Context is already up to date")
		os.Exit(1)
	}

	msg := "Updated"

	var toAdd []string
	if numFiles > 0 {
		postfix := "s"
		if numFiles == 1 {
			postfix = ""
		}
		toAdd = append(toAdd, fmt.Sprintf("%d file%s", numFiles, postfix))
	}
	if numTrees > 0 {
		postfix := "s"
		if numTrees == 1 {
			postfix = ""
		}
		toAdd = append(toAdd, fmt.Sprintf("%d tree%s", numTrees, postfix))
	}
	if numUrls > 0 {
		postfix := "s"
		if numUrls == 1 {
			postfix = ""
		}
		toAdd = append(toAdd, fmt.Sprintf("%d url%s", numUrls, postfix))
	}

	if len(toAdd) <= 2 {
		msg += " " + strings.Join(toAdd, " and ")
	} else {
		for i, add := range toAdd {
			if i == len(toAdd)-1 {
				msg += ", and " + add
			} else {
				msg += ", " + add
			}
		}
	}

	msg += " in context"

	action := "added"
	if tokensDiff < 0 {
		action = "removed"
	}
	absTokenDiff := int(math.Abs(float64(tokensDiff)))
	msg += fmt.Sprintf(" | %s → %d 🪙 | total → %d 🪙", action, absTokenDiff, totalTokens)

	err = GitCommitContextUpdate(msg + "\n\n" + TableForContextUpdateRes(updateRes))

	stopFn()

	if err != nil {
		fmt.Println("Error committing context update:", err)
		os.Exit(1)
	}

	fmt.Println("✅ " + msg)

	return updateRes
}

func TableForContextUpdateRes(updateRes *updateRes) string {
	updatedParts := updateRes.UpdatedParts
	tokenDiffsByName := updateRes.TokenDiffsByName

	if len(updatedParts) == 0 {
		return ""
	}

	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Name", "Type", "🪙"})
	table.SetAutoWrapText(false)

	for _, part := range updatedParts {
		t, icon := GetContextTypeAndIcon(part)
		diff := tokenDiffsByName[part.Name]

		diffStr := "+" + strconv.Itoa(diff)
		tableColor := tablewriter.FgHiGreenColor

		if diff < 0 {
			diffStr = strconv.Itoa(diff)
			tableColor = tablewriter.FgHiRedColor
		}

		row := []string{
			" " + icon + " " + part.Name,
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

func UpdateContext() (*updateRes, error) {
	return checkOutdatedAndMaybeUpdateContext(true)
}

func CheckOutdatedContext() (*updateRes, error) {
	return checkOutdatedAndMaybeUpdateContext(false)
}

func checkOutdatedAndMaybeUpdateContext(doUpdate bool) (*updateRes, error) {
	maxTokens := shared.MaxContextTokens
	planState, err := GetPlanState()
	if err != nil {
		return nil, fmt.Errorf("error retrieving plan state: %w", err)
	}

	tokenDiffsByName := make(map[string]int)
	tokensDiff := 0
	totalTokens := planState.ContextTokens
	totalUpdatableTokens := planState.ContextUpdatableTokens
	var mu sync.Mutex

	context, err := GetAllContext(true)
	if err != nil {
		return nil, fmt.Errorf("error retrieving context: %w", err)
	}

	ts := shared.StringTs()

	updateCh := make(chan *contextUpdate, len(context))
	errCh := make(chan error, len(context))

	var wg sync.WaitGroup

	for _, part := range context {
		contextPath := CreateContextFileName(part.Name, part.Sha)

		if part.Type == shared.ContextFileType {
			wg.Add(1)
			go func(part *shared.ModelContextPart) {
				defer wg.Done()
				fileContent, err := os.ReadFile(part.FilePath)
				if err != nil {
					errCh <- fmt.Errorf("failed to read the file %s: %v", part.FilePath, err)
					return
				}

				hash := sha256.Sum256(fileContent)
				sha := hex.EncodeToString(hash[:])

				if sha == part.Sha {
					updateCh <- nil
				} else {
					body := string(fileContent)
					numTokens := shared.GetNumTokens(body)

					mu.Lock()
					tokensDiff += numTokens - part.NumTokens
					tokenDiffsByName[part.Name] = numTokens - part.NumTokens
					mu.Unlock()

					updateCh <- &contextUpdate{Path: contextPath, Part: &shared.ModelContextPart{
						Type:      shared.ContextFileType,
						Name:      part.Name,
						FilePath:  part.FilePath,
						Body:      body,
						Sha:       sha,
						NumTokens: numTokens,
						AddedAt:   part.AddedAt,
						UpdatedAt: ts,
					}}
				}
			}(part)

		} else if part.Type == shared.ContextDirectoryTreeType {
			wg.Add(1)
			go func(part *shared.ModelContextPart) {
				defer wg.Done()
				flattenedPaths, err := ParseInputPaths([]string{part.FilePath}, &types.LoadContextParams{
					NamesOnly: true,
				})
				if err != nil {
					errCh <- fmt.Errorf("failed to get the directory tree %s: %v", part.FilePath, err)
					return
				}
				body := strings.Join(flattenedPaths, "\n")
				bytes := []byte(body)

				hash := sha256.Sum256(bytes)
				sha := hex.EncodeToString(hash[:])

				if sha == part.Sha {
					updateCh <- nil
				} else {
					numTokens := shared.GetNumTokens(body)
					mu.Lock()
					tokensDiff += numTokens - part.NumTokens
					tokenDiffsByName[part.Name] = numTokens - part.NumTokens
					mu.Unlock()

					updateCh <- &contextUpdate{Path: contextPath, Part: &shared.ModelContextPart{
						Type:      shared.ContextDirectoryTreeType,
						Name:      part.Name,
						FilePath:  part.FilePath,
						Body:      body,
						Sha:       sha,
						NumTokens: numTokens,
						AddedAt:   part.AddedAt,
						UpdatedAt: ts,
					}}
				}
			}(part)

		} else if part.Type == shared.ContextURLType {
			wg.Add(1)
			go func(part *shared.ModelContextPart) {
				defer wg.Done()
				body, err := url.FetchURLContent(part.Url)
				if err != nil {
					errCh <- fmt.Errorf("failed to fetch the URL %s: %v", part.Url, err)
					return
				}

				hash := sha256.Sum256([]byte(body))
				sha := hex.EncodeToString(hash[:])

				if sha == part.Sha {
					updateCh <- nil
				} else {
					numTokens := shared.GetNumTokens(body)
					mu.Lock()
					tokensDiff += numTokens - part.NumTokens
					tokenDiffsByName[part.Name] = numTokens - part.NumTokens
					mu.Unlock()

					updateCh <- &contextUpdate{Path: contextPath, Part: &shared.ModelContextPart{
						Type:      shared.ContextURLType,
						Name:      part.Name,
						Url:       part.Url,
						Body:      body,
						Sha:       sha,
						NumTokens: numTokens,
						AddedAt:   part.AddedAt,
						UpdatedAt: ts,
					}}
				}

			}(part)
		} else {
			updateCh <- nil
		}
	}

	wg.Wait()

	updatedPaths := []string{}
	updatedParts := []*shared.ModelContextPart{}

	for range context {
		select {
		case err := <-errCh:
			return nil, err
		case update := <-updateCh:
			if update != nil {
				updatedPaths = append(updatedPaths, update.Path)
				updatedParts = append(updatedParts, update.Part)
			}
		}
	}

	totalTokens += tokensDiff
	totalUpdatableTokens += tokensDiff

	if doUpdate {
		if totalTokens > maxTokens {
			return &updateRes{
				UpdatedParts: nil,
				TokensDiff:   tokensDiff,
				TotalTokens:  totalTokens,
				MaxExceeded:  true,
			}, fmt.Errorf("context update would exceed the token limit of %d", maxTokens)
		}

		err = ContextRemoveFiles(updatedPaths)
		if err != nil {
			return nil, fmt.Errorf("error removing updated context paths: %w", err)
		}

		err = writeContextParts(updatedParts)
		if err != nil {
			return nil, fmt.Errorf("error writing updated context parts: %w", err)
		}

		planState.ContextTokens = totalTokens
		planState.ContextUpdatableTokens = totalUpdatableTokens
		err = SetPlanState(planState, ts)

		if err != nil {
			return nil, fmt.Errorf("error writing plan state: %w", err)
		}
	}

	var numFiles int
	var numUrls int
	var numTrees int
	for _, part := range updatedParts {
		if part.Type == shared.ContextFileType {
			numFiles++
		}
		if part.Type == shared.ContextURLType {
			numUrls++
		}
		if part.Type == shared.ContextDirectoryTreeType {
			numTrees++
		}
	}

	return &updateRes{
		UpdatedParts:     updatedParts,
		TokenDiffsByName: tokenDiffsByName,
		TokensDiff:       tokensDiff,
		TotalTokens:      totalTokens,
		MaxExceeded:      totalTokens > maxTokens,
		NumFiles:         numFiles,
		NumUrls:          numUrls,
		NumTrees:         numTrees,
	}, nil

}
