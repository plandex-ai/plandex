package auth

import (
	"fmt"
	"plandex/term"
	"time"
)

func ConvertTrial() {
	openAuthenticatedURL("Opening Plandex Cloud upgrade flow in your browser.", "/settings/billing?upgrade=1&cliUpgrade=1")

	fmt.Println("\nCommand will continue automatically once you've upgraded...")
	fmt.Println()
	term.StartSpinner("")

	startTime := time.Now()
	expirationTime := startTime.Add(1 * time.Hour)

	for time.Now().Before(expirationTime) {
		org, apiErr := apiClient.GetOrgSession()

		if apiErr != nil {
			term.StopSpinner()
			term.OutputErrorAndExit("error getting org session: %s", apiErr.Msg)
		}

		if org != nil && !org.IsTrial {
			term.StopSpinner()
			fmt.Println("🚀 Trial upgraded")
			fmt.Println()
			return
		}

		time.Sleep(1500 * time.Millisecond)
	}

	term.StopSpinner()
	term.OutputErrorAndExit("Timed out waiting for upgrade. Please try again. Email support@plandex.ai if the problem persists.")
}

func startTrial() {
	term.StartSpinner("")
	cliTrialToken, apiErr := apiClient.CreateCliTrialSession()
	term.StopSpinner()

	if apiErr != nil {
		term.OutputErrorAndExit("error starting trial: %s", apiErr.Msg)
	}

	openUnauthenticatedCloudURL(
		"Opening Plandex Cloud trial flow in your browser.",
		fmt.Sprintf("/start?cliTrialToken=%s", cliTrialToken),
	)

	fmt.Println("\nCommand will continue automatically once you've started your trial...")
	fmt.Println()
	term.StartSpinner("")

	startTime := time.Now()
	expirationTime := startTime.Add(1 * time.Hour)

	for time.Now().Before(expirationTime) {
		cliTrialSession, apiErr := apiClient.GetCliTrialSession(cliTrialToken)

		if apiErr != nil {
			term.StopSpinner()
			term.OutputErrorAndExit("error getting cli trial session: %s", apiErr.Msg)
		}

		if cliTrialSession != nil {
			// Trial session is valid, break the loop and sign in
			term.StopSpinner()
			fmt.Println("🚀 Trial started")
			fmt.Println()
			err := handleSignInResponse(cliTrialSession, "")
			if err != nil {
				term.OutputErrorAndExit("error signing in after trial started: %s", err)
			}
			return
		}

		time.Sleep(1500 * time.Millisecond)
	}

	term.StopSpinner()
	term.OutputErrorAndExit("Timed out waiting for trial to start. Please try again. Email support@plandex.ai if the problem persists.")
}
