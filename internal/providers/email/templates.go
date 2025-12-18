package email

import (
	"fmt"
	"serenibase/internal/constant"
)

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
        .footer { padding: 20px; text-align: center; color: #6b6e76; font-size: 12px; white-space: pre-wrap; }
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
            <p>%s</p>
        </div>
    </div>
</body>
</html>
`, title, title, content, constant.EmailFooterNotice)
}

// Email Verification OTP
func (s *service) EmailVerificationOTPBody(otp string) EmailContent {
	subject := "Email Address Verification Required"
	content := fmt.Sprintf(`
		<p>Dear User,</p>
		<p>Thank you for registering with Serenibase.</p>
		<p>To complete the verification of your email address, please use the One-Time Password (OTP) provided below:</p>
		<p style="font-size: 24px; font-weight: bold; text-align: center; margin: 30px 0; letter-spacing: 5px; color: #3869D4;">%s</p>
		<p>This OTP is valid for a period of <strong>5 minutes</strong>.</p>
		<p>If you did not initiate this request, please disregard this email. No further action is required.</p>
		<p>For security reasons, please do not share this OTP with anyone.</p>
	`, otp)

	return EmailContent{
		Subject: subject,
		Body:    s.wrapBody(subject, content),
	}
}

// Password Reset
func (s *service) PasswordResetBody(resetLink string) EmailContent {
	subject := "Password Reset Instructions"
	content := fmt.Sprintf(`
		<p>Dear User,</p>
		<p>We received a request to reset the password associated with your Serenibase account.</p>
		<p>To proceed, please click the button below:</p>
		<p style="text-align: center; margin: 30px 0;">
			<a href="%s" class="button" style="color: #ffffff;">Reset Password</a>
		</p>
		<p>Alternatively, you may copy and paste the following link into your web browser:</p>
		<p><a href="%s">%s</a></p>
		<p>This password reset link will expire in <strong>1 hour</strong> for security purposes.</p>
		<p>If you did not request a password reset, please ignore this email. Your account will remain secure.</p>
	`, resetLink, resetLink, resetLink)

	return EmailContent{
		Subject: subject,
		Body:    s.wrapBody(subject, content),
	}
}

// Tenant/User Invitation
func (s *service) PlatformInvitationBody(firstName, tenantName, resetLink string) EmailContent {
	subject := fmt.Sprintf("Invitation to Join %s on Serenibase", tenantName)
	content := fmt.Sprintf(`
		<p>Dear %s,</p>
		<p>You have been invited to join the organization <strong>%s</strong> on the Serenibase platform.</p>
		<p>To accept this invitation and complete your account setup, please click the button below:</p>
		<p style="text-align: center; margin: 30px 0;">
			<a href="%s" class="button" style="color: #ffffff;">Accept Invitation</a>
		</p>
		<p>Alternatively, you may use the following link:</p>
		<p><a href="%s">%s</a></p>
		<p>This invitation link will remain valid for <strong>1 hour</strong>.</p>
		<p>If you believe this invitation was sent to you in error, you may safely disregard this message.</p>
		<p>We look forward to your participation on Serenibase.</p>
	`, firstName, tenantName, resetLink, resetLink, resetLink)

	return EmailContent{
		Subject: subject,
		Body:    s.wrapBody(subject, content),
	}
}

// Added to Workspace
func (s *service) AddedToWorkspaceBody(workspaceName, access string) EmailContent {
	subject := fmt.Sprintf("Access Granted to Workspace: %s", workspaceName)
	content := fmt.Sprintf(`
		<p>Dear User,</p>
		<p>You have been granted access to the workspace "<strong>%s</strong>" on Serenibase.</p>
		<p>Assigned Access Level: <strong>%s</strong></p>
		<p>You may now log in to begin collaborating within this workspace.</p>
		<p>If you have any questions regarding your access, please contact your workspace administrator.</p>
	`, workspaceName, access)

	return EmailContent{
		Subject: subject,
		Body:    s.wrapBody(subject, content),
	}
}

// Removed from Workspace
func (s *service) RemovedFromWorkspaceBody(workspaceLabel string) EmailContent {
	subject := fmt.Sprintf("Workspace Access Revoked: %s", workspaceLabel)
	content := fmt.Sprintf(`
		<p>Dear User,</p>
		<p>Your access to the workspace "<strong>%s</strong>" on Serenibase has been removed.</p>
		<p>If you believe this change was made in error, please reach out to your workspace administrator for further clarification.</p>
	`, workspaceLabel)

	return EmailContent{
		Subject: subject,
		Body:    s.wrapBody(subject, content),
	}
}

// Invited to Workspace
func (s *service) InvitedToWorkspaceBody(workspaceName, access string) EmailContent {
	subject := fmt.Sprintf("Invitation to Join Workspace: %s", workspaceName)
	content := fmt.Sprintf(`
		<p>Dear User,</p>
		<p>You have been invited to join the workspace "<strong>%s</strong>" on Serenibase.</p>
		<p>Proposed Access Level: <strong>%s</strong></p>
		<p>Please log in to your Serenibase account to review and accept this invitation.</p>
		<p>If you were not expecting this invitation, no action is required.</p>
	`, workspaceName, access)

	return EmailContent{
		Subject: subject,
		Body:    s.wrapBody(subject, content),
	}
}

// Workspace Access Updated
func (s *service) WorkspaceAccessUpdatedBody(workspaceName, access string) EmailContent {
	subject := "Workspace Access Level Updated"
	content := fmt.Sprintf(`
		<p>Dear User,</p>
		<p>Your access permissions for the workspace "<strong>%s</strong>" have been updated.</p>
		<p>New Access Level: <strong>%s</strong></p>
		<p>These changes take effect immediately.</p>
		<p>If you require additional information regarding this update, please contact your workspace administrator.</p>
	`, workspaceName, access)

	return EmailContent{
		Subject: subject,
		Body:    s.wrapBody(subject, content),
	}
}
