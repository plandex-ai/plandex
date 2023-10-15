package cmd

import (
	"github.com/spf13/cobra"
)

var (
	recursive bool
	maxDepth  int
	namesOnly bool
	truncate  bool
	maxTokens int
)

func addSharedContextFlags(cmd *cobra.Command) {

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Search directories recursively")
	cmd.Flags().IntVarP(&maxDepth, "depth", "d", -1, "Maximum depth for recursive directory search (-1 means no limit)")
	cmd.Flags().BoolVar(&namesOnly, "names", false, "Only process file names")
	cmd.Flags().BoolVar(&truncate, "truncate", false, "Truncate contents if tokens exceed maximum")
	cmd.Flags().IntVar(&maxTokens, "max", -1, "Maximum limit for number of tokens")
}
