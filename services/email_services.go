package services

import (
	"api/config"
	"fmt"
	"net/smtp"
	"strings"
)

type EmailService struct {
    host     string
    port     string
    username string
    password string
}

func NewEmailService() *EmailService {
    return &EmailService{
        host:     config.MailHost,
        port:     config.MailPort,
        username: config.MailUsername,
        password: config.MailPassword,
    }
}

func (s *EmailService) SendPasswordResetEmail(to, resetToken string) error {
    auth := smtp.PlainAuth("", s.username, s.password, s.host)
    
    resetLink := fmt.Sprintf(config.ClientUrl + "/reset-password?token=%s", resetToken)
    
    htmlTemplate := strings.TrimSpace(`
To: %s
MIME-version: 1.0
Content-Type: text/html; charset="UTF-8"
Subject: Reset Your AlgoHive Password

<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Reset Your Password</title>
</head>
<body style="background-color: #f9fafb; margin: 0; padding: 0; font-family: Arial, sans-serif;">
    <table width="100%%" cellpadding="0" cellspacing="0" style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <tr>
            <td style="background: linear-gradient(to right, #1a1a1a, #2d2d2d); padding: 40px 20px; text-align: center; border-radius: 12px;">
                <h1 style="color: #ffffff; margin-bottom: 30px; font-size: 24px;">Reset Your Password</h1>
                <p style="color: #9ca3af; margin-bottom: 30px; font-size: 16px;">Click the button below to reset your password. This link will expire in 1 hour.</p>
                <a href="%s" style="display: inline-block; background-color: #d97706; color: #ffffff; text-decoration: none; padding: 12px 30px; border-radius: 25px; font-weight: bold; margin-bottom: 30px;">Reset Password</a>
                <p style="color: #9ca3af; font-size: 14px;">If you didn't request this password reset, please ignore this email.</p>
            </td>
        </tr>
        <tr>
            <td style="text-align: center; padding-top: 20px;">
                <p style="color: #6b7280; font-size: 14px;">© 2025 AlgoHive. All rights reserved.</p>
            </td>
        </tr>
    </table>
</body>
</html>
`)

    msg := []byte(fmt.Sprintf(htmlTemplate, to, resetLink))
    return smtp.SendMail(s.host+":"+s.port, auth, s.username, []string{to}, msg)
}