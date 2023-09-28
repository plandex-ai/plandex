package cmd

import (
	"plandex/lib"

	"github.com/spf13/cobra"
)

var note string

var contextLoadCmd = &cobra.Command{
	Use:   "load [files-or-urls...]",
	Short: "Load context from various inputs",
	Long:  `Load context from a file path, a directory, a URL, text, or piped data.`,
	Args:  cobra.RangeArgs(0, 1),
	Run:   contextLoad,
}

func init() {
	addSharedContextFlags(contextLoadCmd)
	contextLoadCmd.Flags().StringVarP(&note, "note", "n", "", "Add a note to the context")

	// can be called via plandex load or plandex context load
	RootCmd.AddCommand(contextLoadCmd)
	contextCmd.AddCommand(contextLoadCmd)
}

func contextLoad(cmd *cobra.Command, args []string) {
	lib.LoadContextOrDie(&lib.LoadContextParams{
		Note:      note,
		MaxTokens: maxTokens,
		Recursive: recursive,
		MaxDepth:  maxDepth,
		NamesOnly: namesOnly,
		Truncate:  truncate,
		Resources: args,
	})
}
