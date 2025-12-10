package email

import "fmt"

// service implements EmailTemplateService.
type service struct{}

// NewEmailTemplateService returns an implementation of EmailTemplateService.
func NewEmailTemplateService() EmailTemplateService {
	return &service{}
}

// Helper function to wrap content in a professional HTML template
func (s *service) wrapBody(title string, content string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        body { font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; background-color: #f4f4f7; color: #51545E; margin: 0; padding: 0; -webkit-text-size-adjust: none; height: 100%%; line-height: 1.4; }
        .email-wrapper { width: 100%%; background-color: #f4f4f7; padding: 20px; }
        .email-content { max-width: 600px; margin: 0 auto; background-color: #ffffff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05); }
        .email-header { background-color: #3869D4; padding: 20px; text-align: center; color: #ffffff; }
        .email-header h1 { margin: 0; font-size: 24px; font-weight: bold; }
        .email-body { padding: 40px; }
        .email-body p { margin-bottom: 20px; font-size: 16px; color: #333333; }
        .button { display: inline-block; background-color: #3869D4; color: #ffffff; text-decoration: none; padding: 12px 24px; border-radius: 4px; font-weight: bold; text-align: center; }
        .footer { padding: 20px; text-align: center; color: #6b6e76; font-size: 12px; }
        a { color: #3869D4; text-decoration: none; }
        strong { color: #333333; }
    </style>
</head>
<body>
    <div class="email-wrapper">
        <div class="email-content">
            <div class="email-header">
                <h1>%s</h1>
            </div>
            <div class="email-body">
                %s
            </div>
        </div>
        <div class="footer">
            <p>&copy; Serenibase. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`, title, title, content)
}

// Email Verification OTP
func (s *service) EmailVerificationOTPBody(otp string) EmailContent {
	subject := "Verify Your Email"
	content := fmt.Sprintf(`
		<p>Hello,</p>
		<p>Thank you for registering with Serenibase. To verify your email address, please use the One-Time Password (OTP) below:</p>
		<p style="font-size: 24px; font-weight: bold; text-align: center; margin: 30px 0; letter-spacing: 5px; color: #3869D4;">%s</p>
		<p>This OTP is valid for <strong>5 minutes</strong>.</p>
		<p>If you did not request this verification, please ignore this email.</p>
	`, otp)

	return EmailContent{
		Subject: subject,
		Body:    s.wrapBody(subject, content),
	}
}

// Password Reset
func (s *service) PasswordResetBody(resetLink string) EmailContent {
	subject := "Reset Your Password"
	content := fmt.Sprintf(`
		<p>Hello,</p>
		<p>We received a request to reset your password for your Serenibase account.</p>
		<p style="text-align: center; margin: 30px 0;">
			<a href="%s" class="button" style="color: #ffffff;">Reset Password</a>
		</p>
		<p>Or copy and paste this link into your browser:</p>
		<p><a href="%s">%s</a></p>
		<p>This link will expire in <strong>1 hour</strong>.</p>
		<p>If you did not request a password reset, you can safely ignore this email.</p>
	`, resetLink, resetLink, resetLink)

	return EmailContent{
		Subject: subject,
		Body:    s.wrapBody(subject, content),
	}
}

// Tenant/User Invitation
func (s *service) PlatformInvitationBody(firstName, tenantName, resetLink string) EmailContent {
	subject := fmt.Sprintf("You're Invited to %s", tenantName)
	content := fmt.Sprintf(`
		<p>Hello %s,</p>
		<p>You have been invited to join the <strong>%s</strong> tenant on Serenibase.</p>
		<p>To accept your invitation and set up your account password, please click the button below:</p>
		<p style="text-align: center; margin: 30px 0;">
			<a href="%s" class="button" style="color: #ffffff;">Join %s</a>
		</p>
		<p>Or use this link:</p>
		<p><a href="%s">%s</a></p>
		<p>This invitation link is valid for <strong>1 hour</strong>.</p>
		<p>We look forward to having you on board!</p>
	`, firstName, tenantName, resetLink, tenantName, resetLink, resetLink)

	return EmailContent{
		Subject: subject,
		Body:    s.wrapBody(subject, content),
	}
}

// Added to Workspace
func (s *service) AddedToWorkspaceBody(workspaceName, access string) EmailContent {
	subject := "Added to Workspace"
	content := fmt.Sprintf(`
		<p>Hello,</p>
		<p>You have been added to the workspace <strong>%s</strong>.</p>
		<p>Your access level is: <strong>%s</strong></p>
		<p>You can now start collaborating with your team in this workspace.</p>
	`, workspaceName, access)

	return EmailContent{
		Subject: subject,
		Body:    s.wrapBody(subject, content),
	}
}

// Removed from Workspace
func (s *service) RemovedFromWorkspaceBody(workspaceLabel string) EmailContent {
	subject := "Removed from Workspace"
	content := fmt.Sprintf(`
		<p>Hello,</p>
		<p>You have been removed from the workspace <strong>%s</strong>.</p>
		<p>If you believe this is an error, please contact your workspace administrator.</p>
	`, workspaceLabel)

	return EmailContent{
		Subject: subject,
		Body:    s.wrapBody(subject, content),
	}
}

// Invited to Workspace
func (s *service) InvitedToWorkspaceBody(workspaceName, access string) EmailContent {
	subject := "Workspace Invitation"
	content := fmt.Sprintf(`
		<p>Hello,</p>
		<p>You have been invited to join the workspace <strong>%s</strong>.</p>
		<p>Access Level: <strong>%s</strong></p>
		<p>Log in to your account to view and accept this invitation.</p>
	`, workspaceName, access)

	return EmailContent{
		Subject: subject,
		Body:    s.wrapBody(subject, content),
	}
}

// Workspace Access Updated
func (s *service) WorkspaceAccessUpdatedBody(workspaceName, access string) EmailContent {
	subject := "Access Rights Updated"
	content := fmt.Sprintf(`
		<p>Hello,</p>
		<p>Your access rights for the workspace <strong>%s</strong> have been updated.</p>
		<p>New Access Level: <strong>%s</strong></p>
	`, workspaceName, access)

	return EmailContent{
		Subject: subject,
		Body:    s.wrapBody(subject, content),
	}
}
