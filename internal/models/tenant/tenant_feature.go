package tenant

import (
	"fmt"
	"godbgrest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type TenantFeature struct {
	ID            uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	TenantID      string    `db:"tenant_id" json:"tenant_id,omitempty" mapstructure:"tenant_id"`
	FeatureFlagID string    `db:"feature_flag_id" json:"feature_flag_id,omitempty" mapstructure:"feature_flag_id"`
	Enabled       bool      `db:"enabled" json:"enabled,omitempty" mapstructure:"enabled"`
	Config        *string   `db:"config" json:"config,omitempty" mapstructure:"config"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (TenantFeature) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".shared.tenant_features", prefix)
}

func (tbl TenantFeature) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "tenant_id", DataType: "varchar", NotNull: true},
			{Name: "feature_flag_id", DataType: "varchar", NotNull: true},
			{Name: "enabled", DataType: "boolean", NotNull: true},
			{Name: "config", DataType: "text"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_tenant_features_tenant_flag", Columns: []string{"tenant_id", "feature_flag_id"}, Unique: true},
			{Name: "idx_tenant_features_tenant_id", Columns: []string{"tenant_id"}},
			{Name: "idx_tenant_features_feature_flag_id", Columns: []string{"feature_flag_id"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_tenant_features_feature_flag_id",
				Columns:           []string{"feature_flag_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".shared.feature_flags", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}
