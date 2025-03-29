package services

import (
	"api/config"
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"time"
)

// Define email service errors
var (
    ErrMissingRecipient   = errors.New("missing email recipient")
    ErrMissingCredentials = errors.New("missing email credentials")
    ErrSendingEmail       = errors.New("error sending email")
)

// EmailService handles sending emails through SMTP
type EmailService struct {
    host     string
    port     string
    username string
    password string
    fromAddr string
}

// NewEmailService creates a new email service with credentials from config
func NewEmailService() *EmailService {
    return &EmailService{
        host:     config.MailHost,
        port:     config.MailPort,
        username: config.MailUsername,
        password: config.MailPassword,
        fromAddr: config.MailUsername,
    }
}

// SendPasswordResetEmail sends a password reset email with a token link
func (s *EmailService) SendPasswordResetEmail(to, resetToken string) error {
    // Validate inputs
    if to == "" {
        return fmt.Errorf("%w: recipient email is required", ErrMissingRecipient)
    }
    if s.username == "" || s.password == "" {
        return fmt.Errorf("%w: SMTP credentials not configured", ErrMissingCredentials)
    }
    
    // Create authentication
    smtp.PlainAuth("", s.username, s.password, s.host)
    
    // Build reset link with proper URL construction
    resetLink := fmt.Sprintf("%s/reset-password?token=%s", strings.TrimRight(config.ClientUrl, "/"), resetToken)
    
    // Email subject
    subject := "Reset Your AlgoHive Password"
    
    // Prepare email content
    data := map[string]interface{}{
        "ResetLink":  resetLink,
        "Year":       time.Now().Year(),
        "Expiration": "1 hour",
    }
    
    // Generate email content
    content, err := s.generateEmailContent(resetPasswordTemplate, data)
    if err != nil {
        return fmt.Errorf("failed to generate email content: %w", err)
    }
    
    // Build email headers
    headers := map[string]string{
        "To":           to,
        "From":         s.fromAddr,
        "Subject":      subject,
        "MIME-Version": "1.0",
        "Content-Type": "text/html; charset=\"UTF-8\"",
    }
    
    // Send the email
    if err := s.sendMail(to, headers, content); err != nil {
        return fmt.Errorf("%w: %s", ErrSendingEmail, err)
    }
    
    return nil
}

// SendSupportEmail sends a support request email
func (s *EmailService) SendSupportEmail(name, email, issueType, subject, message string) error {
    // Validate inputs
    if name == "" || email == "" || message == "" {
        return fmt.Errorf("%w: name, email and message are required", ErrMissingRecipient)
    }
    
    // Support email recipient - should be configurable
    to := "ericphlpp@proton.me"
    
    // Create authentication
    smtp.PlainAuth("", s.username, s.password, s.host)
    
    // Prepare email content
    data := map[string]interface{}{
        "Name":      name,
        "Email":     email,
        "IssueType": issueType,
        "Subject":   subject,
        "Message":   message,
        "Year":      time.Now().Year(),
    }
    
    // Generate email content
    content, err := s.generateEmailContent(supportEmailTemplate, data)
    if err != nil {
        return fmt.Errorf("failed to generate email content: %w", err)
    }
    
    // Build email headers
    headers := map[string]string{
        "To":           to,
        "From":         s.fromAddr,
        "Reply-To":     email,
        "Subject":      fmt.Sprintf("Support Request from %s: %s", name, subject),
        "MIME-Version": "1.0",
        "Content-Type": "text/html; charset=\"UTF-8\"",
    }
    
    // Send the email
    if err := s.sendMail(to, headers, content); err != nil {
        return fmt.Errorf("%w: %s", ErrSendingEmail, err)
    }
    
    return nil
}

// sendMail is a helper function to send an email
func (s *EmailService) sendMail(recipient string, headers map[string]string, body string) error {
    // Build the message with headers
    var message bytes.Buffer
    
    // Add headers
    for key, value := range headers {
        message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
    }
    
    // Add empty line between headers and body
    message.WriteString("\r\n")
    
    // Add body
    message.WriteString(body)
    
    // Send the email
    err := smtp.SendMail(
        s.host+":"+s.port,
        smtp.PlainAuth("", s.username, s.password, s.host),
        s.fromAddr,
        []string{recipient},
        message.Bytes(),
    )
    
    if err != nil {
        return err
    }
    
    return nil
}

// generateEmailContent processes a template with data
func (s *EmailService) generateEmailContent(templateContent string, data map[string]interface{}) (string, error) {
    // Parse the template
    tmpl, err := template.New("email").Parse(templateContent)
    if err != nil {
        return "", err
    }
    
    // Execute the template with the data
    var content bytes.Buffer
    if err := tmpl.Execute(&content, data); err != nil {
        return "", err
    }
    
    return content.String(), nil
}

// Email templates
const resetPasswordTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset Your Password</title>
</head>
<body style="background-color: #f9fafb; margin: 0; padding: 0; font-family: Arial, sans-serif;">
    <table width="100%" cellpadding="0" cellspacing="0" style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <tr>
            <td style="background: linear-gradient(to right, #1a1a1a, #2d2d2d); padding: 40px 20px; text-align: center; border-radius: 12px;">
                <h1 style="color: #ffffff; margin-bottom: 30px; font-size: 24px;">Reset Your Password</h1>
                <p style="color: #9ca3af; margin-bottom: 30px; font-size: 16px;">Click the button below to reset your password. This link will expire in {{.Expiration}}.</p>
                <a href="{{.ResetLink}}" style="display: inline-block; background-color: #d97706; color: #ffffff; text-decoration: none; padding: 12px 30px; border-radius: 25px; font-weight: bold; margin-bottom: 30px;">Reset Password</a>
                <p style="color: #9ca3af; font-size: 14px;">If you didn't request this password reset, please ignore this email.</p>
            </td>
        </tr>
        <tr>
            <td style="text-align: center; padding-top: 20px;">
                <p style="color: #6b7280; font-size: 14px;">© {{.Year}} AlgoHive. All rights reserved.</p>
            </td>
        </tr>
    </table>
</body>
</html>`

const supportEmailTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Support Request</title>
</head>
<body style="background-color: #f9fafb; margin: 0; padding: 0; font-family: Arial, sans-serif;">
    <table width="100%" cellpadding="0" cellspacing="0" style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <tr>
            <td style="background: linear-gradient(to right, #1a1a1a, #2d2d2d); padding: 40px 20px; text-align: center; border-radius: 12px;">
                <h1 style="color: #ffffff; margin-bottom: 30px; font-size: 24px;">Support Request from {{.Name}}</h1>
                <p style="color: #9ca3af; margin-bottom: 15px; font-size: 16px;"><strong>From:</strong> {{.Name}} ({{.Email}})</p>
                <p style="color: #9ca3af; margin-bottom: 15px; font-size: 16px;"><strong>Issue Type:</strong> {{.IssueType}}</p>
                <p style="color: #9ca3af; margin-bottom: 15px; font-size: 16px;"><strong>Subject:</strong> {{.Subject}}</p>
                <p style="color: #9ca3af; margin-bottom: 30px; font-size: 16px; text-align: left;"><strong>Message:</strong><br>{{.Message}}</p>
                <p style="color: #9ca3af; font-size: 14px;">This is an automated message. To reply, email the sender directly.</p>
            </td>
        </tr>
        <tr>
            <td style="text-align: center; padding-top: 20px;">
                <p style="color: #6b7280; font-size: 14px;">© {{.Year}} AlgoHive. All rights reserved.</p>
            </td>
        </tr>
    </table>
</body>
</html>`