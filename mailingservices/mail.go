package mailingservices

import (
	"context"
	"github.com/mailgun/mailgun-go/v4"
	"os"
	"time"
)

type Mailgun struct {
	Client *mailgun.MailgunImpl
}

type Mailer interface {
	SendWelcomeMessage(userEmail, link string) (string, error)
	SendVerifyAccount(userEmail, link string) (string, error)
	SendResetPassword(userEmail, link string) (string, error)
}

func (mail *Mailgun) Init() {
	domain := os.Getenv("MG_DOMAIN")
	apiKey := os.Getenv("MG_PUBLIC_API_KEY")
	mail.Client = mailgun.NewMailgun(domain, apiKey)
}

func (mail Mailgun) SendWelcomeMessage(userEmail, link string) (string, error) {
	EmailFrom := os.Getenv("MG_EMAIL_FROM")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	m := mail.Client.NewMessage(EmailFrom, "Welcome to CX", "")
	m.SetTemplate("welcome")
	if err := m.AddRecipient(userEmail); err != nil {
		return "", err
	}

	res, _, errr := mail.Client.Send(ctx, m)
	if errr != nil {
		return "", errr
	}
	return res, nil
}

func (mail *Mailgun) SendVerifyAccount(userEmail, link string) (string, error) {
	EmailFrom := os.Getenv("MG_EMAIL_FROM")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	m := mail.Client.NewMessage(EmailFrom, "Verify Account", "")
	m.SetTemplate("verify.account")
	if err := m.AddRecipient(userEmail); err != nil {
		return "", err
	}

	err := m.AddVariable("link", link)
	if err != nil {
		return "", err
	}

	res, _, errr := mail.Client.Send(ctx, m)
	return res, errr
}

func (mail *Mailgun) SendResetPassword(userEmail, live string) (string, error) {
    EmailFrom := os.Getenv("MG_EMAIL_FROM")

    ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
    defer cancel()

    // Create a new message
    m := mail.Client.NewMessage(EmailFrom, "Reset Password", "")
    
    // Use the "live" template
    m.SetTemplate("live")

    // Add the recipient email
    if err := m.AddRecipient(userEmail); err != nil {
        return "", err
    }

    // Inject the reset link into the template variable
    err := m.AddVariable("live", live)
    if err != nil {
        return "", err
    }

    // Send the email
    res, _, errr := mail.Client.Send(ctx, m)
    if errr != nil {
        return "", errr
    }

    return res, nil
}
