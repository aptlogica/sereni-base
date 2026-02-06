package models_test

import (
	"testing"
	"time"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRelation_TableName(t *testing.T) {
	relation := tenant.Relation{}
	schema := "test_schema"

	tableName := relation.TableName(schema)

	assert.Equal(t, `"test_schema".relations`, tableName)
}

func TestRelation_OneToMany(t *testing.T) {
	relationID := uuid.New()
	baseID := uuid.New().String()
	now := time.Now().UTC()

	relation := tenant.Relation{
		ID:             relationID,
		BaseID:         baseID,
		SourceModelID:  "orders",
		SourceColumnID: "customer_id",
		TargetModelID:  "customers",
		TargetColumnID: "id",
		RelationType:   "one_to_many",
		UpdateRule:     "CASCADE",
		DeleteRule:     "CASCADE",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	assert.Equal(t, relationID, relation.ID)
	assert.Equal(t, baseID, relation.BaseID)
	assert.Equal(t, "orders", relation.SourceModelID)
	assert.Equal(t, "customers", relation.TargetModelID)
	assert.Equal(t, "one_to_many", relation.RelationType)
	assert.Equal(t, "CASCADE", relation.UpdateRule)
	assert.Equal(t, "CASCADE", relation.DeleteRule)
}

func TestRelation_ManyToMany(t *testing.T) {
	junctionModelID := "user_groups_junction"

	relation := tenant.Relation{
		ID:              uuid.New(),
		BaseID:          uuid.New().String(),
		SourceModelID:   "users",
		SourceColumnID:  "id",
		TargetModelID:   "groups",
		TargetColumnID:  "id",
		RelationType:    "many_to_many",
		JunctionModelID: &junctionModelID,
		UpdateRule:      "CASCADE",
		DeleteRule:      "CASCADE",
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	assert.Equal(t, "many_to_many", relation.RelationType)
	assert.NotNil(t, relation.JunctionModelID)
	assert.Equal(t, "user_groups_junction", *relation.JunctionModelID)
}

func TestRelation_WithLookupColumns(t *testing.T) {
	sourceLookup := []string{"name", "email"}
	targetLookup := []string{"title", "description"}

	relation := tenant.Relation{
		ID:                  uuid.New(),
		BaseID:              uuid.New().String(),
		SourceModelID:       "tasks",
		SourceColumnID:      "assignee_id",
		SourceLookupColumns: sourceLookup,
		TargetModelID:       "users",
		TargetColumnID:      "id",
		TargetLookupColumns: targetLookup,
		RelationType:        "many_to_one",
		UpdateRule:          "NO ACTION",
		DeleteRule:          "SET NULL",
	}

	assert.Equal(t, 2, len(relation.SourceLookupColumns))
	assert.Equal(t, "name", relation.SourceLookupColumns[0])
	assert.Equal(t, "email", relation.SourceLookupColumns[1])
	assert.Equal(t, 2, len(relation.TargetLookupColumns))
	assert.Equal(t, "title", relation.TargetLookupColumns[0])
}

func TestRelation_DeleteRules(t *testing.T) {
	testCases := []struct {
		name       string
		deleteRule string
		updateRule string
	}{
		{"cascade", "CASCADE", "CASCADE"},
		{"set_null", "SET NULL", "SET NULL"},
		{"no_action", "NO ACTION", "NO ACTION"},
		{"restrict", "RESTRICT", "RESTRICT"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			relation := tenant.Relation{
				ID:             uuid.New(),
				BaseID:         uuid.New().String(),
				SourceModelID:  "child",
				SourceColumnID: "parent_id",
				TargetModelID:  "parent",
				TargetColumnID: "id",
				RelationType:   "many_to_one",
				UpdateRule:     tc.updateRule,
				DeleteRule:     tc.deleteRule,
			}

			assert.Equal(t, tc.deleteRule, relation.DeleteRule)
			assert.Equal(t, tc.updateRule, relation.UpdateRule)
		})
	}
}

func TestRelation_TableSchema(t *testing.T) {
	relation := tenant.Relation{}
	schema := "test_schema"

	tableSchema := relation.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".relations`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
}
