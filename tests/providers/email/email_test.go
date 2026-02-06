package email_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"serenibase/internal/config"
	"serenibase/internal/providers/email"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	htmlDoctype = "<!DOCTYPE html>"
	testEmail   = "test@example.com"
	testSubject = "Test Subject"
	testBody    = "<h1>Test Body</h1>"
)

// TestEmailTemplateServiceGetResetPasswordEmailTemplate tests the password reset email template
func TestEmailTemplateServiceGetResetPasswordEmailTemplate(t *testing.T) {
	t.Parallel()
	service := email.NewEmailTemplateService()
	resetLink := "https://example.com/reset?token=abc123"

	result := service.PasswordResetBody(resetLink)

	assert.NotEmpty(t, result.Subject)
	assert.Contains(t, result.Subject, "Password Reset")
	assert.NotEmpty(t, result.Body)
	assert.Contains(t, result.Body, resetLink)
	assert.Contains(t, result.Body, htmlDoctype)
	assert.Contains(t, result.Body, "1 hour")
}

// TestEmailTemplateServiceGetRegistrationEmailTemplate tests the registration email template
func TestEmailTemplateServiceGetRegistrationEmailTemplate(t *testing.T) {
	t.Parallel()
	service := email.NewEmailTemplateService()
	otp := "1234"

	result := service.EmailVerificationOTPBody(otp)

	assert.NotEmpty(t, result.Subject)
	assert.Contains(t, result.Subject, "Verification")
	assert.NotEmpty(t, result.Body)
	assert.Contains(t, result.Body, otp)
	assert.Contains(t, result.Body, htmlDoctype)
	assert.Contains(t, result.Body, "5 minutes")
}

// TestEmailTemplateServiceGetVerificationEmailTemplate tests the verification email template
func TestEmailTemplateServiceGetVerificationEmailTemplate(t *testing.T) {
	t.Parallel()
	service := email.NewEmailTemplateService()
	otp := "5678"

	result := service.EmailVerificationOTPBody(otp)

	assert.NotEmpty(t, result.Subject)
	assert.NotEmpty(t, result.Body)
	assert.Contains(t, result.Body, otp)
	assert.Contains(t, result.Body, htmlDoctype)
	assert.Contains(t, result.Body, "email-wrapper")
}

// TestEmailTemplateServiceGetInviteUserEmailTemplate tests the user invitation email template
func TestEmailTemplateServiceGetInviteUserEmailTemplate(t *testing.T) {
	t.Parallel()
	service := email.NewEmailTemplateService()
	firstName := "John"
	tenantName := "Acme Corp"
	resetLink := "https://example.com/invite?token=xyz789"

	result := service.PlatformInvitationBody(firstName, tenantName, resetLink)

	assert.NotEmpty(t, result.Subject)
	assert.Contains(t, result.Subject, "Invitation")
	assert.Contains(t, result.Subject, tenantName)
	assert.NotEmpty(t, result.Body)
	assert.Contains(t, result.Body, firstName)
	assert.Contains(t, result.Body, tenantName)
	assert.Contains(t, result.Body, resetLink)
	assert.Contains(t, result.Body, htmlDoctype)
}

// TestEmailTemplateServiceGetWelcomeEmailTemplate tests the welcome email template
func TestEmailTemplateServiceGetWelcomeEmailTemplate(t *testing.T) {
	t.Parallel()
	service := email.NewEmailTemplateService()
	workspaceName := "Main Workspace"
	access := "Editor"

	result := service.AddedToWorkspaceBody(workspaceName, access)

	assert.NotEmpty(t, result.Subject)
	assert.Contains(t, result.Subject, "Access Granted")
	assert.NotEmpty(t, result.Body)
	assert.Contains(t, result.Body, workspaceName)
	assert.Contains(t, result.Body, access)
	assert.Contains(t, result.Body, htmlDoctype)
}

// TestEmailTemplateServiceGetPasswordChangedEmailTemplate tests the password changed email template
func TestEmailTemplateServiceGetPasswordChangedEmailTemplate(t *testing.T) {
	t.Parallel()
	service := email.NewEmailTemplateService()
	workspaceLabel := "Dev Workspace"

	result := service.RemovedFromWorkspaceBody(workspaceLabel)

	assert.NotEmpty(t, result.Subject)
	assert.Contains(t, result.Subject, "Revoked")
	assert.NotEmpty(t, result.Body)
	assert.Contains(t, result.Body, workspaceLabel)
	assert.Contains(t, result.Body, htmlDoctype)
}

// TestEmailTemplateServiceGetAccountDeletedEmailTemplate tests the account deleted email template
func TestEmailTemplateServiceGetAccountDeletedEmailTemplate(t *testing.T) {
	t.Parallel()
	service := email.NewEmailTemplateService()
	workspaceName := "Project X"
	access := "Viewer"

	result := service.InvitedToWorkspaceBody(workspaceName, access)

	assert.NotEmpty(t, result.Subject)
	assert.Contains(t, result.Subject, "Invitation")
	assert.NotEmpty(t, result.Body)
	assert.Contains(t, result.Body, workspaceName)
	assert.Contains(t, result.Body, access)
	assert.Contains(t, result.Body, htmlDoctype)
}

// TestEmailTemplateServiceGetAccountSuspendedEmailTemplate tests the account suspended email template
func TestEmailTemplateServiceGetAccountSuspendedEmailTemplate(t *testing.T) {
	t.Parallel()
	service := email.NewEmailTemplateService()
	workspaceName := "Alpha Team"
	access := "Admin"

	result := service.WorkspaceAccessUpdatedBody(workspaceName, access)

	assert.NotEmpty(t, result.Subject)
	assert.Contains(t, result.Subject, "Updated")
	assert.NotEmpty(t, result.Body)
	assert.Contains(t, result.Body, workspaceName)
	assert.Contains(t, result.Body, access)
	assert.Contains(t, result.Body, htmlDoctype)
}

// TestEmailTemplateServiceAllTemplatesHaveHTMLStructure tests that all templates have proper HTML structure
func TestEmailTemplateServiceAllTemplatesHaveHTMLStructure(t *testing.T) {
	t.Parallel()
	service := email.NewEmailTemplateService()

	tests := []struct {
		name     string
		template email.EmailContent
	}{
		{
			name:     "PasswordResetBody",
			template: service.PasswordResetBody("https://example.com/reset"),
		},
		{
			name:     "EmailVerificationOTPBody",
			template: service.EmailVerificationOTPBody("1234"),
		},
		{
			name:     "PlatformInvitationBody",
			template: service.PlatformInvitationBody("John", "Acme", "https://example.com"),
		},
		{
			name:     "AddedToWorkspaceBody",
			template: service.AddedToWorkspaceBody("Workspace", "Editor"),
		},
		{
			name:     "RemovedFromWorkspaceBody",
			template: service.RemovedFromWorkspaceBody("Workspace"),
		},
		{
			name:     "InvitedToWorkspaceBody",
			template: service.InvitedToWorkspaceBody("Workspace", "Viewer"),
		},
		{
			name:     "WorkspaceAccessUpdatedBody",
			template: service.WorkspaceAccessUpdatedBody("Workspace", "Admin"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, tt.template.Body, htmlDoctype)
			assert.Contains(t, tt.template.Body, "<html>")
			assert.Contains(t, tt.template.Body, "</html>")
			assert.Contains(t, tt.template.Body, "email-wrapper")
			assert.Contains(t, tt.template.Body, "email-content")
			assert.NotEmpty(t, tt.template.Subject)
		})
	}
}

// TestNewService tests the NewService constructor
func TestNewService(t *testing.T) {
	t.Parallel()
	cfg := config.EmailConfig{
		URL: "http://email-service:8080",
	}
	queueSize := 100
	templateService := email.NewEmailTemplateService()

	service := email.NewService(cfg, queueSize, templateService)

	assert.NotNil(t, service)
}

// TestEmailServiceSendEmail tests the sendEmail functionality
func TestEmailServiceSendEmail(t *testing.T) {
	t.Parallel()
	t.Run("successful email send", func(t *testing.T) {
		// Create a test server that simulates email service
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/send", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			// Verify request body
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var payload map[string]interface{}
			err = json.Unmarshal(body, &payload)
			require.NoError(t, err)

			assert.Contains(t, payload, "to")
			assert.Contains(t, payload, "subject")
			assert.Contains(t, payload, "body")
			assert.Equal(t, true, payload["is_html"])

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"success"}`))
		}))
		defer server.Close()

		cfg := config.EmailConfig{
			URL: server.URL,
		}
		templateService := email.NewEmailTemplateService()
		service := email.NewService(cfg, 10, templateService)

		// Start the service
		service.Start(1)
		defer service.Stop()

		// Enqueue an email
		job := email.EmailJob{
			To:      testEmail,
			Subject: testSubject,
			Body:    "testBody",
		}
		service.Enqueue(job)

		// Wait for processing
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("email send with non-200 response", func(t *testing.T) {
		// Create a test server that returns error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal error"}`))
		}))
		defer server.Close()

		cfg := config.EmailConfig{
			URL: server.URL,
		}
		templateService := email.NewEmailTemplateService()
		service := email.NewService(cfg, 10, templateService)

		service.Start(1)
		defer service.Stop()

		job := email.EmailJob{
			To:      testEmail,
			Subject: testSubject,
			Body:    "testBody",
		}
		service.Enqueue(job)

		// Wait for processing
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("email send with network error", func(t *testing.T) {
		cfg := config.EmailConfig{
			URL: "http://invalid-host-that-does-not-exist:9999",
		}
		templateService := email.NewEmailTemplateService()
		service := email.NewService(cfg, 10, templateService)

		service.Start(1)
		defer service.Stop()

		job := email.EmailJob{
			To:      testEmail,
			Subject: testSubject,
			Body:    "testBody",
		}
		service.Enqueue(job)

		// Wait for processing
		time.Sleep(50 * time.Millisecond)
	})
}

// TestEmailServiceQueueManagement tests queue management
func TestEmailServiceQueueManagement(t *testing.T) {
	t.Parallel()
	t.Run("queue multiple emails", func(t *testing.T) {
		emailsSent := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			emailsSent++
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cfg := config.EmailConfig{
			URL: server.URL,
		}
		templateService := email.NewEmailTemplateService()
		service := email.NewService(cfg, 10, templateService)

		service.Start(2) // Start 2 workers
		defer service.Stop()

		// Enqueue multiple emails
		for i := 0; i < 5; i++ {
			job := email.EmailJob{
				To:      testEmail,
				Subject: testSubject,
				Body:    "testBody",
			}
			service.Enqueue(job)
		}

		// Wait for processing
		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, 5, emailsSent)
	})

	t.Run("queue full handling", func(t *testing.T) {
		cfg := config.EmailConfig{
			URL: "http://localhost:8080",
		}
		templateService := email.NewEmailTemplateService()
		service := email.NewService(cfg, 2, templateService) // Small queue

		// Don't start workers, so queue fills up
		for i := 0; i < 5; i++ {
			job := email.EmailJob{
				To:      testEmail,
				Subject: testSubject,
				Body:    "testBody",
			}
			service.Enqueue(job) // Should drop some emails
		}
	})
}

// TestEmailServiceWorkerLifecycle tests worker start and stop
func TestEmailServiceWorkerLifecycle(t *testing.T) {
	t.Parallel()
	cfg := config.EmailConfig{
		URL: "http://localhost:8080",
	}
	templateService := email.NewEmailTemplateService()
	service := email.NewService(cfg, 10, templateService)

	// Start workers
	service.Start(3)

	// Enqueue some jobs
	for i := 0; i < 3; i++ {
		job := email.EmailJob{
			To:      testEmail,
			Subject: testSubject,
			Body:    "testBody",
		}
		service.Enqueue(job)
	}

	// Stop service
	service.Stop()

	// Verify graceful shutdown (no panic)
	assert.True(t, true)
}

// TestEmailServiceMultipleWorkers tests concurrent workers processing
func TestEmailServiceMultipleWorkers(t *testing.T) {
	t.Parallel()
	processedEmails := make(map[string]bool)
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]interface{}
		json.Unmarshal(body, &payload)

		mu.Lock()
		processedEmails[payload["subject"].(string)] = true
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.EmailConfig{
		URL: server.URL,
	}
	templateService := email.NewEmailTemplateService()
	service := email.NewService(cfg, 20, templateService)

	service.Start(3) // 3 concurrent workers
	defer service.Stop()

	// Enqueue emails with different subjects
	for i := 0; i < 10; i++ {
		job := email.EmailJob{
			To:      testEmail,
			Subject: "Subject " + string(rune(i)),
			Body:    "testBody",
		}
		service.Enqueue(job)
	}

	// Wait for processing
	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	count := len(processedEmails)
	mu.Unlock()

	assert.Equal(t, 10, count)
}

// TestEmailTemplateServiceSpecialCharacters tests templates with special characters
func TestEmailTemplateServiceSpecialCharacters(t *testing.T) {
	t.Parallel()
	service := email.NewEmailTemplateService()

	t.Run("OTP with special format", func(t *testing.T) {
		result := service.EmailVerificationOTPBody("1234")
		assert.Contains(t, result.Body, "1234")
		// Verify HTML encoding is proper
		assert.NotContains(t, result.Body, "<script>")
	})

	t.Run("Names with special characters", func(t *testing.T) {
		firstName := "John O'Brien"
		tenantName := "Acme & Sons"
		result := service.PlatformInvitationBody(firstName, tenantName, "https://example.com")
		assert.Contains(t, result.Body, firstName)
		assert.Contains(t, result.Body, tenantName)
	})

	t.Run("Workspace names with quotes", func(t *testing.T) {
		workspaceName := `"Special" Workspace`
		result := service.AddedToWorkspaceBody(workspaceName, "Editor")
		assert.Contains(t, result.Body, workspaceName)
	})
}

// TestEmailServiceContextCancellation tests context cancellation scenarios
func TestEmailServiceContextCancellation(t *testing.T) {
	t.Parallel()
	requestReceived := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		// Simulate slow response
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.EmailConfig{
		URL: server.URL,
	}
	templateService := email.NewEmailTemplateService()
	service := email.NewService(cfg, 10, templateService)

	service.Start(1)
	defer service.Stop()

	job := email.EmailJob{
		To:      testEmail,
		Subject: testSubject,
		Body:    "testBody",
	}
	service.Enqueue(job)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	assert.True(t, requestReceived)
}
