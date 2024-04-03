package email

import (
	"fmt"
	"log"
	"os"

	"github.com/atotto/clipboard"
	"github.com/gen2brain/beeep"
)

func SendVerificationEmail(email string, pin string) error {
	// Check if the environment is production
	if os.Getenv("GOENV") == "production" {
		// Production environment - send email using AWS SES
		subject := "Your Plandex Pin"
		htmlBody := fmt.Sprintf("<p>Hi there,</p><p>Welcome to Plandex!</p><p>Your pin is:<br><strong>%s</strong></p><p>It will be valid for the next 5 minutes. Please return to the terminal and paste in your pin.</p>", pin)
		textBody := fmt.Sprintf("Hi there,\n\nWelcome to Plandex!\n\nYour pin is:\n%s\n\nIt will be valid for the next 5 minutes. Please return to the terminal and paste in your pin.", pin)

		if os.Getenv("IS_CLOUD") == "" {
			return sendEmailViaSMTP(email, subject, htmlBody, textBody)
		} else {
			return sendEmailViaSES(email, subject, htmlBody, textBody)
		}
	}

	if os.Getenv("GOENV") == "development" {
		// Development environment
		log.Printf("Development mode: Verification pin is %s for email %s", pin, email)

		// Copy pin to clipboard
		clipboard.WriteAll(pin) // ignore error

		// Send notification
		beeep.Notify("Verification Pin", fmt.Sprintf("Verification pin %s copied to clipboard %s", pin, email), "") // ignore error
	}

	return nil
}
