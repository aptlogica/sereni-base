package master

import (
	"fmt"
	"godbgrest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type TenantDomain struct {
	ID                 uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	TenantID           uuid.UUID `db:"tenant_id" json:"tenant_id,omitempty" mapstructure:"tenant_id"`
	Domain             string    `db:"domain" json:"domain,omitempty" mapstructure:"domain"`
	IsPrimary          bool      `db:"is_primary" json:"is_primary,omitempty" mapstructure:"is_primary"`
	SSLEnabled         bool      `db:"ssl_enabled" json:"ssl_enabled,omitempty" mapstructure:"ssl_enabled"`
	SSLCertificate     *string   `db:"ssl_certificate" json:"ssl_certificate,omitempty" mapstructure:"ssl_certificate"`
	VerificationStatus string    `db:"verification_status" json:"verification_status,omitempty" mapstructure:"verification_status"`
	VerificationToken  *string   `db:"verification_token" json:"verification_token,omitempty" mapstructure:"verification_token"`
	DNSRecords         *string   `db:"dns_records" json:"dns_records,omitempty" mapstructure:"dns_records"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (TenantDomain) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".tenant_domains", prefix)
}

func (tbl TenantDomain) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "tenant_id", DataType: "uuid", NotNull: true},
			{Name: "domain", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "is_primary", DataType: "boolean", DefaultValue: strPtr("false")},
			{Name: "ssl_enabled", DataType: "boolean", DefaultValue: strPtr("false")},
			{Name: "ssl_certificate", DataType: "text"},
			{Name: "verification_status", DataType: "varchar", DefaultValue: strPtr("'pending'")},
			{Name: "verification_token", DataType: "varchar"},
			{Name: "dns_records", DataType: "text"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Columns:           []string{"tenant_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".tenants", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}
