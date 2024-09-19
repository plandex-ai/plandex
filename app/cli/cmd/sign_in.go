package cmd

import (
	"plandex/auth"
	"plandex/term"

	"github.com/spf13/cobra"
)

var signInCmd = &cobra.Command{
	Use:   "sign-in",
	Short: "Sign in to a Plandex account",
	Args:  cobra.NoArgs,
	Run:   signIn,
}

func init() {
	RootCmd.AddCommand(signInCmd)

	signInCmd.Flags().String("code", "", "Sign in code from the Plandex web UI")
}

func signIn(cmd *cobra.Command, args []string) {
	code, err := cmd.Flags().GetString("code")
	if err != nil {
		term.OutputErrorAndExit("Error getting code: %v", err)
	}

	if code != "" {
		err = auth.SignInWithCode(code, "")

		if err != nil {
			term.OutputErrorAndExit("Error signing in: %v", err)
		}

		return
	}

	err = auth.SelectOrSignInOrCreate()

	if err != nil {
		term.OutputErrorAndExit("Error signing in: %v", err)
	}
}
