package master

import (
	"fmt"
	"godbgrest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type TenantMembership struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	TenantID    uuid.UUID `db:"tenant_id" json:"tenant_id,omitempty" mapstructure:"tenant_id"`
	UserID      uuid.UUID `db:"user_id" json:"user_id,omitempty" mapstructure:"user_id"`
	RoleID      uuid.UUID `db:"role_id" json:"role_id,omitempty" mapstructure:"role_id"`
	Status      string    `db:"status" json:"status,omitempty" mapstructure:"status"`
	Permissions string    `db:"permissions" json:"permissions,omitempty" mapstructure:"permissions"` // stored as JSON string

	// Invitation details
	InvitedBy           *string    `db:"invited_by" json:"invited_by,omitempty" mapstructure:"invited_by"`
	InvitedAt           *time.Time `db:"invited_at" json:"invited_at,omitempty" mapstructure:"invited_at"`
	InvitationToken     *string    `db:"invitation_token" json:"invitation_token,omitempty" mapstructure:"invitation_token"`
	InvitationExpiresAt *time.Time `db:"invitation_expires_at" json:"invitation_expires_at,omitempty" mapstructure:"invitation_expires_at"`
	JoinedAt            *time.Time `db:"joined_at" json:"joined_at,omitempty" mapstructure:"joined_at"`
	LastAccessAt        *time.Time `db:"last_access_at" json:"last_access_at,omitempty" mapstructure:"last_access_at"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (TenantMembership) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".tenant_memberships", prefix)
}

func (tbl TenantMembership) TableSchema(prefix string) models.CreateTableRequest {
	null := "NULL"

	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			// Core fields
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "tenant_id", DataType: "uuid", NotNull: true},
			{Name: "user_id", DataType: "uuid", NotNull: true},
			{Name: "role_id", DataType: "uuid", NotNull: true},
			{Name: "status", DataType: "varchar", DefaultValue: strPtr("'active'")},
			{Name: "permissions", DataType: "text"},

			// Invitation details
			{Name: "invited_by", DataType: "varchar", DefaultValue: &null},
			{Name: "invited_at", DataType: "timestamp", DefaultValue: &null},
			{Name: "invitation_token", DataType: "varchar", DefaultValue: &null},
			{Name: "invitation_expires_at", DataType: "timestamp", DefaultValue: &null},
			{Name: "joined_at", DataType: "timestamp", DefaultValue: &null},
			{Name: "last_access_at", DataType: "timestamp", DefaultValue: &null},

			// Timestamps
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
		},

		Indexes: []models.IndexDefinition{
			{Name: "uq_tenant_memberships_tenant_user", Columns: []string{"tenant_id", "user_id"}, Unique: true},
			{Name: "idx_control_memberships_tenant", Columns: []string{"tenant_id"}},
			{Name: "idx_control_memberships_user", Columns: []string{"user_id"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_tenant_memberships_tenant",
				Columns:           []string{"tenant_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".tenants", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
			{
				Name:              "fk_tenant_memberships_user",
				Columns:           []string{"user_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".users", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
			{
				Name:              "fk_tenant_memberships_role",
				Columns:           []string{"role_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".roles", prefix),
				ReferencedColumns: []string{"id"},
			},
		},
	}
}
