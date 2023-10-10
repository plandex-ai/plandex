package cmd

import (
	"github.com/plandex/plandex/shared"

	"github.com/spf13/cobra"
)

var (
	recursive bool
	maxDepth  int16
	namesOnly bool
	truncate  bool
	maxTokens int = shared.MaxTokens
)

func addSharedContextFlags(cmd *cobra.Command) {

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Search directories recursively")
	cmd.Flags().Int16VarP(&maxDepth, "depth", "d", -1, "Maximum depth for recursive directory search (-1 means no limit)")
	cmd.Flags().BoolVar(&namesOnly, "names", false, "Only process file names")
	cmd.Flags().BoolVar(&truncate, "truncate", false, "Truncate contents if tokens exceed maximum")
	cmd.Flags().IntVar(&maxTokens, "max", shared.MaxTokens, "Maximum limit for number of tokens")
}
