package shared

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
)

type ContextUpdateResult struct {
	UpdatedContexts []*Context
	TokenDiffsById  map[string]int
	TokensDiff      int
	TotalTokens     int
	NumFiles        int
	NumUrls         int
	NumTrees        int
	MaxTokens       int
}

func (c *Context) TypeAndIcon() (string, string) {
	var icon string
	var t string
	switch c.ContextType {
	case ContextFileType:
		icon = "📄"
		t = "file"
	case ContextURLType:
		icon = "🌎"
		t = "url"
	case ContextDirectoryTreeType:
		icon = "🗂 "
		t = "tree"
	case ContextNoteType:
		icon = "✏️ "
		t = "note"
	case ContextPipedDataType:
		icon = "↔️ "
		t = "piped"
	}

	return t, icon
}

func TableForLoadContext(contexts []*Context) string {
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Name", "Type", "🪙"})
	table.SetAutoWrapText(false)

	for _, context := range contexts {
		t, icon := context.TypeAndIcon()
		row := []string{
			" " + icon + " " + context.Name,
			t,
			"+" + strconv.Itoa(context.NumTokens),
		}

		table.Rich(row, []tablewriter.Colors{
			{tablewriter.FgHiGreenColor, tablewriter.Bold},
			{tablewriter.FgHiGreenColor},
			{tablewriter.FgHiGreenColor},
		})
	}

	table.Render()

	return tableString.String()
}

func SummaryForLoadContext(contexts []*Context, tokensAdded, totalTokens int) string {

	var hasNote bool
	var hasPiped bool

	var numFiles int
	var numTrees int
	var numUrls int

	for _, context := range contexts {
		switch context.ContextType {
		case ContextFileType:
			numFiles++
		case ContextURLType:
			numUrls++
		case ContextDirectoryTreeType:
			numTrees++
		case ContextNoteType:
			hasNote = true
		case ContextPipedDataType:
			hasPiped = true
		}
	}

	var added []string

	if hasNote {
		added = append(added, "a note")
	}
	if hasPiped {
		added = append(added, "piped data")
	}
	if numFiles > 0 {
		label := "file"
		if numFiles > 1 {
			label = "files"
		}
		added = append(added, fmt.Sprintf("%d %s", numFiles, label))
	}
	if numTrees > 0 {
		label := "directory tree"
		if numTrees > 1 {
			label = "directory trees"
		}
		added = append(added, fmt.Sprintf("%d %s", numTrees, label))
	}
	if numUrls > 0 {
		label := "url"
		if numUrls > 1 {
			label = "urls"
		}
		added = append(added, fmt.Sprintf("%d %s", numUrls, label))
	}

	msg := "Loaded "

	if len(added) <= 2 {
		msg += strings.Join(added, " and ")
	} else {
		for i, add := range added {
			if i == len(added)-1 {
				msg += ", and " + add
			} else {
				msg += ", " + add
			}
		}
	}

	msg += fmt.Sprintf(" into context | added → %d 🪙 |  total → %d 🪙", tokensAdded, totalTokens)

	return msg
}

func TableForRemoveContext(contexts []*Context) string {
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Name", "Type", "🪙"})
	table.SetAutoWrapText(false)

	for _, context := range contexts {
		t, icon := context.TypeAndIcon()
		row := []string{
			" " + icon + " " + context.Name,
			t,
			"-" + strconv.Itoa(context.NumTokens),
		}

		table.Rich(row, []tablewriter.Colors{
			{tablewriter.FgHiRedColor, tablewriter.Bold},
			{tablewriter.FgHiRedColor},
			{tablewriter.FgHiRedColor},
		})
	}

	table.Render()

	return tableString.String()
}

func SummaryForRemoveContext(contexts []*Context, previousTotalTokens int) string {
	removedTokens := 0

	for _, context := range contexts {
		removedTokens += context.NumTokens
	}

	totalTokens := previousTotalTokens - removedTokens

	suffix := ""
	if len(contexts) > 1 {
		suffix = "s"
	}

	return fmt.Sprintf("Removed %d piece%s of context | removed → %d 🪙 | total → %d 🪙", len(contexts), suffix, removedTokens, totalTokens)
}

func SummaryForUpdateContext(updateRes *ContextUpdateResult) string {
	numFiles := updateRes.NumFiles
	numTrees := updateRes.NumTrees
	numUrls := updateRes.NumUrls
	tokensDiff := updateRes.TokensDiff
	totalTokens := updateRes.TotalTokens

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

	return msg
}

func TableForContextUpdate(updateRes *ContextUpdateResult) string {
	contexts := updateRes.UpdatedContexts
	tokenDiffsById := updateRes.TokenDiffsById

	if len(contexts) == 0 {
		return ""
	}

	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Name", "Type", "🪙"})
	table.SetAutoWrapText(false)

	for _, context := range contexts {
		t, icon := context.TypeAndIcon()
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
