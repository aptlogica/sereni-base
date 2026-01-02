package tenant

import (
	"fmt"
	"godbgrest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type GlobalAuditLog struct {
	ID            uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	UserID        uuid.UUID `db:"user_id" json:"user_id,omitempty" mapstructure:"user_id"`
	EventType     string    `db:"event_type" json:"event_type,omitempty" mapstructure:"event_type"`
	EventCategory string    `db:"event_category" json:"event_category,omitempty" mapstructure:"event_category"`
	ResourceType  *string   `db:"resource_type" json:"resource_type,omitempty" mapstructure:"resource_type"`
	ResourceID    *string   `db:"resource_id" json:"resource_id,omitempty" mapstructure:"resource_id"`

	// Context
	IPAddress *string   `db:"ip_address" json:"ip_address,omitempty" mapstructure:"ip_address"`
	UserAgent *string   `db:"user_agent" json:"user_agent,omitempty" mapstructure:"user_agent"`
	RequestID uuid.UUID `db:"request_id" json:"request_id,omitempty" mapstructure:"request_id"`
	EventData *string   `db:"event_data" json:"event_data,omitempty" mapstructure:"event_data"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
}

func (GlobalAuditLog) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".global_audit_logs", prefix)
}

func (tbl GlobalAuditLog) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "user_id", DataType: "uuid"},
			{Name: "event_type", DataType: "varchar", NotNull: true},
			{Name: "event_category", DataType: "varchar", NotNull: true},
			{Name: "resource_type", DataType: "varchar"},
			{Name: "resource_id", DataType: "uuid"},
			{Name: "ip_address", DataType: "varchar"},
			{Name: "user_agent", DataType: "text"},
			{Name: "request_id", DataType: "varchar"},
			{Name: "event_data", DataType: "text"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_audit_event_type", Columns: []string{"event_type"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_audit_user",
				Columns:           []string{"user_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".users", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "SET NULL",
			},
		},
	}
}
