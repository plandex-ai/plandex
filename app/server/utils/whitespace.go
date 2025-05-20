package utils

import "strings"

func StripAddedBlankLines(orig, upd string) string {
	origLines := strings.Split(orig, "\n")
	updLines := strings.Split(upd, "\n")

	leadingOrig := 0
	for leadingOrig < len(origLines) && strings.TrimSpace(origLines[leadingOrig]) == "" {
		leadingOrig++
	}

	leadingUpd := 0
	for leadingUpd < len(updLines) && strings.TrimSpace(updLines[leadingUpd]) == "" {
		leadingUpd++
	}

	if leadingUpd > leadingOrig {
		updLines = updLines[leadingUpd-leadingOrig:] // trim surplus
	}

	trailingOrig := 0
	for trailingOrig < len(origLines) && strings.TrimSpace(origLines[len(origLines)-1-trailingOrig]) == "" {
		trailingOrig++
	}

	trailingUpd := 0
	for trailingUpd < len(updLines) && strings.TrimSpace(updLines[len(updLines)-1-trailingUpd]) == "" {
		trailingUpd++
	}

	if trailingUpd > trailingOrig {
		updLines = updLines[:len(updLines)-(trailingUpd-trailingOrig)]
	}

	return strings.Join(updLines, "\n")
}
