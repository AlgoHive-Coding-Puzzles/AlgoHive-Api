package services

import (
	"api/config"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmailService_UsesConfig(t *testing.T) {
	prevUser, prevHost := config.MailUsername, config.MailHost
	config.MailUsername = "bot@algohive.dev"
	config.MailHost = "smtp.algohive.dev"
	t.Cleanup(func() {
		config.MailUsername = prevUser
		config.MailHost = prevHost
	})

	svc := NewEmailService()

	assert.Equal(t, "smtp.algohive.dev", svc.host)
	assert.Equal(t, "bot@algohive.dev", svc.username)
	assert.Equal(t, "bot@algohive.dev", svc.fromAddr)
}

func TestSendPasswordResetEmail_MissingRecipient(t *testing.T) {
	svc := &EmailService{username: "u", password: "p"}

	err := svc.SendPasswordResetEmail("", "token")

	assert.ErrorIs(t, err, ErrMissingRecipient)
}

func TestSendPasswordResetEmail_MissingCredentials(t *testing.T) {
	svc := &EmailService{}

	err := svc.SendPasswordResetEmail("user@example.com", "token")

	assert.ErrorIs(t, err, ErrMissingCredentials)
}

func TestSendSupportEmail_MissingFields(t *testing.T) {
	svc := &EmailService{username: "u", password: "p"}

	err := svc.SendSupportEmail("", "", "bug", "subject", "")

	assert.ErrorIs(t, err, ErrMissingRecipient)
}

func TestGenerateEmailContent_RendersData(t *testing.T) {
	svc := &EmailService{}

	content, err := svc.generateEmailContent(resetPasswordTemplate, map[string]interface{}{
		"ResetLink":  "https://algohive.dev/reset?token=abc",
		"Year":       2026,
		"Expiration": "1 hour",
	})

	require.NoError(t, err)
	assert.Contains(t, content, "https://algohive.dev/reset?token=abc")
	assert.Contains(t, content, "2026")
}

func TestGenerateEmailContent_InvalidTemplate(t *testing.T) {
	svc := &EmailService{}

	_, err := svc.generateEmailContent("{{ .Broken", nil)

	assert.Error(t, err)
}
