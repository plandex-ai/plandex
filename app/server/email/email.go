package email

import (
	"fmt"
	"net/smtp"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
)

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
		Source: aws.String("Plandex <support@plandex.ai>"),
	}

	// Attempt to send the email.
	_, err = svc.SendEmail(input)

	return err
}

func sendEmailViaSMTP(recipient, subject, htmlBody, textBody string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPassword := os.Getenv("SMTP_PASSWORD")

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPassword == "" {
		return fmt.Errorf("SMTP settings not found in environment variables")
	}

	smtpAddress := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)

	// Generate a MIME boundary
	boundary := "BOUNDARY1234567890"
	header := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", smtpUser, recipient, subject, boundary)

	// Prepare the text body part
	textPart := fmt.Sprintf("--%s\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\n%s\r\n", boundary, textBody)

	// Prepare the HTML body part
	htmlPart := fmt.Sprintf("--%s\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s\r\n", boundary, htmlBody)

	// End marker for the boundary
	endBoundary := fmt.Sprintf("--%s--", boundary)

	// Combine the parts to form the full email message
	message := []byte(header + textPart + htmlPart + endBoundary)

	err := smtp.SendMail(smtpAddress, auth, smtpUser, []string{recipient}, message)
	if err != nil {
		return fmt.Errorf("error sending email via SMTP: %v", err)
	}

	return nil
}
