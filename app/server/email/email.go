package email

import (
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/gen2brain/beeep"
)

func SendVerificationEmail(email string, pin string) error {
	// Check if the environment is production
	if os.Getenv("GOENV") == "production" {
		// Production environment - send email using AWS SES
		subject := "Your Verification Pin"
		htmlBody := fmt.Sprintf("<h1>Verification Pin</h1><p>Your verification pin is: <strong>%s</strong></p>", pin)
		textBody := fmt.Sprintf("Your verification pin is: %s", pin)
		return sendEmailViaSES(email, subject, htmlBody, textBody)
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

// sendEmailViaSES sends an email using AWS SES
func sendEmailViaSES(recipient, subject, htmlBody, textBody string) error {
	sess, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("error creating AWS session: %v", err)
	}

	// Create an SES session.
	svc := ses.New(sess)

	// Assemble the email.
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: []*string{
				aws.String(recipient),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(htmlBody),
				},
				Text: &ses.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(textBody),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String("support@plandex.ai"),
	}

	// Attempt to send the email.
	_, err = svc.SendEmail(input)

	return err
}
