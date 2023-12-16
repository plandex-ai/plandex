package lib

import (
	"fmt"
	"plandex-server/db"
	"strings"

	"github.com/plandex/plandex/shared"
)

func FormatModelContext(context []*db.Context) (string, int, error) {
	var contextMessages []string
	var numTokens int
	for _, part := range context {
		var message string
		var fmtStr string
		var args []any

		if part.ContextType == shared.ContextDirectoryTreeType {
			fmtStr = "\n\n- %s | directory tree:\n\n```\n%s\n```"
			args = append(args, part.FilePath, part.Body)
		} else if part.ContextType == shared.ContextFileType {
			fmtStr = "\n\n- %s:\n\n```\n%s\n```"
			args = append(args, part.FilePath, part.Body)
		} else if part.Url != "" {
			fmtStr = "\n\n- %s:\n\n```\n%s\n```"
			args = append(args, part.Url, part.Body)
		} else {
			fmtStr = "\n\n- content%s:\n\n```\n%s\n```"
			args = append(args, part.Name, part.Body)
		}

		numContextTokens, err := shared.GetNumTokens(fmt.Sprintf(fmtStr, ""))
		if err != nil {
			err = fmt.Errorf("failed to get the number of tokens in the context: %v", err)
			return "", 0, err
		}

		numTokens += part.NumTokens + numContextTokens

		message = fmt.Sprintf(fmtStr, args...)

		contextMessages = append(contextMessages, message)
	}
	return strings.Join(contextMessages, "\n"), numTokens, nil
}
