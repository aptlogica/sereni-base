package models_test

import (
	"encoding/json"
	"testing"

	"github.com/aptlogica/go-postgres-rest/pkg/services"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The table is only created at startup, and a definition the database layer rejects is only printed to the log, so
// the definition is checked here with the layer's own validator.
func TestTransferJobTableDefinitionIsAccepted(t *testing.T) {
	svc := services.NewTableService(nil).(*services.TableService)
	require.NoError(t, svc.ValidateCreateTableRequest(tenant.TransferJob{}.TableSchema("public")))
}

// A job read back from the database goes through JSON: the size is a double precision column, so it arrives as a
// number that must still land in an int64.
func TestTransferJobSizeReadFromAFloatColumn(t *testing.T) {
	var job tenant.TransferJob
	require.NoError(t, json.Unmarshal([]byte(`{"id":"a","size_bytes":734003200}`), &job))
	assert.Equal(t, int64(734003200), job.SizeBytes)

	raw, err := json.Marshal(map[string]interface{}{"id": "a", "size_bytes": float64(5368709120)}) // 5 GiB, as pq returns it
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &job))
	assert.Equal(t, int64(5368709120), job.SizeBytes)
}

// An import has no base when it starts, so base_id may be empty; the job type decides what the other columns mean
func TestTransferJobTableHasTheColumnsOfBothTypes(t *testing.T) {
	names := map[string]bool{}
	for _, col := range (tenant.TransferJob{}).TableSchema("public").Columns {
		names[col.Name] = true
		if col.Name == "base_id" {
			assert.False(t, col.NotNull, "an import has no base yet")
		}
	}
	for _, want := range []string{"id", "type", "base_id", "workspace_id", "user_id", "status", "file_name", "file_url", "size_bytes", "error", "result", "expires_at"} {
		assert.True(t, names[want], "missing column %s", want)
	}
	for _, gone := range []string{"include_automations", "schema_name", "worker_id", "heartbeat_at", "attempts", "email_status"} {
		assert.False(t, names[gone], "column %s is not used", gone)
	}
}
