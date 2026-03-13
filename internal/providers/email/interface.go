// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package email

// ServiceInterface defines the contract for the email service
type EmailService interface {
	Start(workers int)
	Stop()
	Enqueue(job EmailJob)
}

// EmailJob represents an email to be sent
type EmailJob struct {
	To      string
	Subject string
	Body    string
}

// EmailTemplateService defines the contract for generating email subjects and bodies for various scenarios
type EmailTemplateService interface {
	EmailVerificationOTPBody(otp string) EmailContent
	PasswordResetBody(resetLink string) EmailContent
	PlatformInvitationBody(firstName, tenantName, resetLink string) EmailContent
	AddedToWorkspaceBody(workspaceName, access string) EmailContent
	RemovedFromWorkspaceBody(workspaceLabel string) EmailContent
	InvitedToWorkspaceBody(workspaceName, access string) EmailContent
	WorkspaceAccessUpdatedBody(workspaceName, access string) EmailContent
}

// EmailContent represents the subject and body of an email
type EmailContent struct {
	Subject string
	Body    string
}
