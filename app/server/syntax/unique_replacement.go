package syntax

import "strings"

func FindUniqueReplacement(originalFile, old string) string {
	oldCount := strings.Count(originalFile, old)

	if oldCount == 1 {
		// perfect match
		return old
	}

	// try to find a unique match. we're forgiving of errors in the middle if we can still identify the block uniquely by checking from both ends
	startMatch := ""
	endMatch := ""
	n := 0
	oldLength := len(old)

	for {
		n++
		if n > oldLength {
			// Prevent slice bounds error
			return ""
		}

		startMatch = old[0:n]
		endMatch = old[oldLength-n : oldLength]

		startOccurrences := strings.Count(originalFile, startMatch)

		if startOccurrences == 0 {
			// lost the match from the start
			return ""
		}

		var endOccurrences int
		var afterStart string
		if startOccurrences == 1 {
			startSplit := strings.Split(originalFile, startMatch)
			afterStart = startSplit[1]

			endOccurrences = strings.Count(afterStart, endMatch)
		} else {
			endOccurrences = strings.Count(originalFile, endMatch)
		}

		if endOccurrences == 0 {
			// lost the match from the end
			return ""
		}

		var beforeEnd string
		if endOccurrences == 1 {
			endSplit := strings.Split(originalFile, endMatch)
			beforeEnd = endSplit[0]
			if startOccurrences > 1 {
				startOccurrences = strings.Count(beforeEnd, startMatch)
			}
		}

		if startOccurrences == 1 && endOccurrences == 1 {
			afterStartIndex := strings.Index(originalFile, afterStart)
			beforeEndIndex := strings.Index(originalFile, beforeEnd)

			startIndex := beforeEndIndex + strings.Index(beforeEnd, startMatch)
			endIndex := afterStartIndex + strings.Index(afterStart, endMatch) + len(endMatch)
			return originalFile[startIndex:endIndex]
		} else {
			// couldn't get a unique match on both ends, keep narrowing down
			continue
		}

	}
}
