package email_test

import (
	"testing"

	"github.com/aptlogica/sereni-base/internal/providers/email"
	"github.com/stretchr/testify/assert"
)

func TestExportReadyEmail(t *testing.T) {
	mail := email.ExportReadyEmail("Sales <CRM>", "https://app.example.com/exports/1?a=1&b=2", "24 hours")
	assert.Equal(t, "Your export of Sales <CRM> is ready", mail.Subject)
	assert.Contains(t, mail.Body, "https://app.example.com/exports/1?a=1&amp;b=2", "the link is HTML-escaped in the body")
	assert.Contains(t, mail.Body, "24 hours")
	assert.NotContains(t, mail.Body, "attached", "the file is never attached; only the link is sent")
	assert.NotContains(t, mail.Body, "<CRM>", "the base title is HTML-escaped in the body")
}
