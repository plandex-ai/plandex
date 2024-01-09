package email

import (
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	"github.com/gen2brain/beeep"
)

func SendVerificationEmail(email string, pin string) error {
	// Check if the environment is production
	if os.Getenv("GOENV") == "production" {
		// TODO: send email in production
		return nil
	}

	if os.Getenv("GOENV") == "development" {
		// Development environment
		// Copy pin to clipboard
		if err := clipboard.WriteAll(pin); err != nil {
			return fmt.Errorf("error copying pin to clipboard in dev: %v", err)
		}

		// Send notification
		err := beeep.Notify("Verification Pin", fmt.Sprintf("Verification pin %s copied to clipboard %s", pin, email), "")
		if err != nil {
			return fmt.Errorf("error sending notification in dev: %v", err)
		}

	}

	return nil
}
