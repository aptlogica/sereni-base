package table_test

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/aptlogica/go-postgres-rest/pkg"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	services "github.com/aptlogica/sereni-base/internal/services/table"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ---------------------------------------------------------------------------
// Fixture
// ---------------------------------------------------------------------------

const relSchema = "schema"

type relFixture struct {
	svc    interfaces.TableManagementService
	table  *MockTableService
	bulk   *MockBulkService
	model  *MockModelService
	column *MockColumnService
	view   *MockViewService
	rel    *MockRelationshipService
}

func newRelFixture() *relFixture {
	f := &relFixture{
		table:  &MockTableService{},
		bulk:   &MockBulkService{},
		model:  &MockModelService{},
		column: &MockColumnService{},
		view:   &MockViewService{},
		rel:    &MockRelationshipService{},
	}
	repo := &pkg.DatabaseService{TableService: f.table, BulkService: f.bulk}
	f.svc = services.NewTableManagementService("postgres", repo, f.model, f.column, f.view, f.rel, &MockAssetManagementService{})
	return f
}

// linkCol builds a "links" column; with is the model ID of the other side.
func linkCol(id uuid.UUID, modelID, relationID, role, with, relType, name, dt string) tenant.Column {
	return tenant.Column{
		ID: id, ModelID: modelID, BaseID: uuid.NewString(), ColumnName: name, Title: name,
		UIDT: "links", DT: helpers.StringPtr(dt),
		Meta: map[string]interface{}{
			"relation_id": relationID,
			"entity_role": role,
			"relation":    map[string]interface{}{"with": with, "type": relType},
		},
	}
}

func plainCol(id uuid.UUID, modelID, name, uidt string) tenant.Column {
	return tenant.Column{ID: id, ModelID: modelID, BaseID: uuid.NewString(), ColumnName: name, Title: name, UIDT: uidt, DT: helpers.StringPtr("TEXT")}
}

func lookupCol(id uuid.UUID, modelID, name string, meta map[string]interface{}) tenant.Column {
	return tenant.Column{ID: id, ModelID: modelID, BaseID: uuid.NewString(), ColumnName: name, Title: name, UIDT: "lookup", DT: helpers.StringPtr("lookup"), Meta: meta}
}

func toResponse(c tenant.Column) dto.ColumnResponse {
	var r dto.ColumnResponse
	if err := helpers.StructToStruct(c, &r); err != nil {
		panic(err)
	}
	return r
}

type recordsGetter interface {
	GetRecordsWithLookups(ctx context.Context, schemaName string, tableName string, columnsData []dto.ColumnResponse) (dto.RecordsResponse, error)
}

// ---------------------------------------------------------------------------
// Creating link columns
// ---------------------------------------------------------------------------

func TestAddLinkColumn_SelfLinkCreatesSingleColumn(t *testing.T) {
	f := newRelFixture()
	modelID := uuid.New()
	baseID := uuid.New()
	srcID := uuid.New()

	var created dto.ColumnInsertion
	f.column.On("Create", mock.Anything, mock.Anything, relSchema).Run(func(args mock.Arguments) {
		created = args.Get(1).(dto.ColumnInsertion)
	}).Return(tenant.Column{ID: srcID, ModelID: modelID.String(), BaseID: baseID.String(), ColumnName: "reports_1", UIDT: "links", DT: helpers.StringPtr("INT[]")}, nil).Once()
	f.model.On("GetModelByID", mock.Anything, relSchema, modelID.String()).Return(tenant.Model{ID: modelID, Alias: "emp", Title: "Employees"}, nil)
	f.table.On("AddColumn", mock.Anything, mock.Anything).Return(nil).Once()
	var relation dto.RelationInsertion
	f.rel.On("Create", mock.Anything, mock.Anything, relSchema).Run(func(args mock.Arguments) {
		relation = args.Get(1).(dto.RelationInsertion)
	}).Return(tenant.Relation{}, nil)

	meta := map[string]interface{}{"relation": map[string]interface{}{"with": modelID.String(), "type": "has-many", "inverse_title": "Manager"}}
	resp, err := f.svc.AddColumn(context.Background(), relSchema, dto.AddColumnRequest{ModelID: modelID, BaseID: baseID, Title: "Reports", UIDT: "links", Meta: meta})

	assert.NoError(t, err)
	assert.Equal(t, srcID, resp.ID)
	f.column.AssertNumberOfCalls(t, "Create", 1)
	f.column.AssertNotCalled(t, "GetMaxOrderIndexOfColumn", mock.Anything, mock.Anything, mock.Anything)
	f.table.AssertNumberOfCalls(t, "AddColumn", 1)

	assert.Equal(t, true, created.Meta["is_self_link"])
	assert.Equal(t, "source", created.Meta["entity_role"])
	assert.NotContains(t, created.Meta["relation"].(map[string]interface{}), "inverse_title")
	assert.Regexp(t, regexp.MustCompile(`^reports_\d+_[0-9a-f]{4}$`), created.ColumnName)

	assert.Equal(t, srcID.String(), relation.SourceColumnID)
	assert.Equal(t, relation.SourceColumnID, relation.TargetColumnID)
	assert.Equal(t, relation.SourceModelID, relation.TargetModelID)
	assert.Equal(t, "has-many", relation.RelationType)
}

func TestAddLinkColumn_TwoTablesInverseTitle(t *testing.T) {
	cases := []struct {
		name          string
		inverseTitle  string
		expectedTitle string
	}{
		{"custom inverse title", "Owner", "Owner"},
		{"default to source table title", "", "Src"},
		{"whitespace treated as empty", "   ", "Src"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newRelFixture()
			srcModel, tgtModel, baseID := uuid.New(), uuid.New(), uuid.New()
			srcCol := linkCol(uuid.New(), srcModel.String(), "", "source", tgtModel.String(), "one-to-one", "rel", "INT")
			tgtCol := linkCol(uuid.New(), tgtModel.String(), "", "target", srcModel.String(), "one-to-one", "owner", "INT")

			var sourceReq, targetReq dto.ColumnInsertion
			f.column.On("Create", mock.Anything, mock.MatchedBy(func(r dto.ColumnInsertion) bool { return r.ModelID == srcModel }), relSchema).
				Run(func(a mock.Arguments) { sourceReq = a.Get(1).(dto.ColumnInsertion) }).Return(srcCol, nil)
			f.column.On("Create", mock.Anything, mock.MatchedBy(func(r dto.ColumnInsertion) bool { return r.ModelID == tgtModel }), relSchema).
				Run(func(a mock.Arguments) { targetReq = a.Get(1).(dto.ColumnInsertion) }).Return(tgtCol, nil)
			f.column.On("GetMaxOrderIndexOfColumn", mock.Anything, relSchema, tgtModel.String()).Return(float64(4), nil)
			f.model.On("GetModelByID", mock.Anything, relSchema, srcModel.String()).Return(tenant.Model{ID: srcModel, Alias: "src", Title: "Src"}, nil)
			f.model.On("GetModelByID", mock.Anything, relSchema, tgtModel.String()).Return(tenant.Model{ID: tgtModel, Alias: "tgt", Title: "Tgt"}, nil)
			f.table.On("AddColumn", mock.Anything, mock.Anything).Return(nil)
			var relation dto.RelationInsertion
			f.rel.On("Create", mock.Anything, mock.Anything, relSchema).Run(func(a mock.Arguments) { relation = a.Get(1).(dto.RelationInsertion) }).Return(tenant.Relation{}, nil)

			meta := map[string]interface{}{"relation": map[string]interface{}{"with": tgtModel.String(), "type": "one-to-one", "inverse_title": c.inverseTitle}}
			_, err := f.svc.AddColumn(context.Background(), relSchema, dto.AddColumnRequest{ModelID: srcModel, BaseID: baseID, Title: "Rel", UIDT: "links", Meta: meta, CreatedBy: "u"})

			assert.NoError(t, err)
			assert.Equal(t, c.expectedTitle, targetReq.Title)
			assert.Equal(t, "target", targetReq.Meta["entity_role"])
			assert.Equal(t, float64(5), *targetReq.OrderIndex)
			assert.Equal(t, "u", targetReq.CreatedBy)
			assert.Equal(t, "u", sourceReq.CreatedBy)
			assert.Equal(t, false, sourceReq.Meta["is_self_link"])
			assert.NotContains(t, sourceReq.Meta["relation"].(map[string]interface{}), "inverse_title")
			assert.NotEqual(t, relation.SourceColumnID, relation.TargetColumnID)
		})
	}
}

func TestAddLinkColumn_RollbackOnFailure(t *testing.T) {
	setup := func(f *relFixture, srcModel, tgtModel uuid.UUID, srcCol, tgtCol tenant.Column, targetErr, relationErr error) {
		f.column.On("Create", mock.Anything, mock.MatchedBy(func(r dto.ColumnInsertion) bool { return r.ModelID == srcModel }), relSchema).Return(srcCol, nil)
		f.column.On("Create", mock.Anything, mock.MatchedBy(func(r dto.ColumnInsertion) bool { return r.ModelID == tgtModel && srcModel != tgtModel }), relSchema).Return(tgtCol, targetErr)
		f.column.On("GetMaxOrderIndexOfColumn", mock.Anything, relSchema, tgtModel.String()).Return(float64(1), nil)
		f.column.On("DeleteColumn", mock.Anything, relSchema, mock.Anything).Return(nil)
		f.model.On("GetModelByID", mock.Anything, relSchema, srcModel.String()).Return(tenant.Model{ID: srcModel, Alias: "src", Title: "Src"}, nil)
		f.model.On("GetModelByID", mock.Anything, relSchema, tgtModel.String()).Return(tenant.Model{ID: tgtModel, Alias: "tgt", Title: "Tgt"}, nil)
		f.table.On("AddColumn", mock.Anything, mock.Anything).Return(nil)
		f.table.On("AlterTable", mock.Anything, mock.Anything).Return(nil)
		f.rel.On("Create", mock.Anything, mock.Anything, relSchema).Return(tenant.Relation{}, relationErr)
	}

	t.Run("target column fails: source removed", func(t *testing.T) {
		f := newRelFixture()
		srcModel, tgtModel := uuid.New(), uuid.New()
		srcCol := linkCol(uuid.New(), srcModel.String(), "", "source", tgtModel.String(), "has-many", "rel", "INT[]")
		setup(f, srcModel, tgtModel, srcCol, tenant.Column{}, errors.New("boom"), nil)

		meta := map[string]interface{}{"relation": map[string]interface{}{"with": tgtModel.String(), "type": "has-many"}}
		_, err := f.svc.AddColumn(context.Background(), relSchema, dto.AddColumnRequest{ModelID: srcModel, BaseID: uuid.New(), Title: "Rel", UIDT: "links", Meta: meta})

		assert.Error(t, err)
		f.column.AssertCalled(t, "DeleteColumn", mock.Anything, relSchema, srcCol.ID.String())
		f.column.AssertNumberOfCalls(t, "DeleteColumn", 1)
		f.table.AssertNumberOfCalls(t, "AlterTable", 1)
		f.rel.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("relation fails: both columns removed", func(t *testing.T) {
		f := newRelFixture()
		srcModel, tgtModel := uuid.New(), uuid.New()
		srcCol := linkCol(uuid.New(), srcModel.String(), "", "source", tgtModel.String(), "has-many", "rel", "INT[]")
		tgtCol := linkCol(uuid.New(), tgtModel.String(), "", "target", srcModel.String(), "has-many", "back", "INT")
		setup(f, srcModel, tgtModel, srcCol, tgtCol, nil, errors.New("boom"))

		meta := map[string]interface{}{"relation": map[string]interface{}{"with": tgtModel.String(), "type": "has-many"}}
		_, err := f.svc.AddColumn(context.Background(), relSchema, dto.AddColumnRequest{ModelID: srcModel, BaseID: uuid.New(), Title: "Rel", UIDT: "links", Meta: meta})

		assert.Error(t, err)
		f.column.AssertCalled(t, "DeleteColumn", mock.Anything, relSchema, srcCol.ID.String())
		f.column.AssertCalled(t, "DeleteColumn", mock.Anything, relSchema, tgtCol.ID.String())
		f.table.AssertNumberOfCalls(t, "AlterTable", 2)
	})

	t.Run("self-link relation fails: single column removed", func(t *testing.T) {
		f := newRelFixture()
		modelID := uuid.New()
		srcCol := linkCol(uuid.New(), modelID.String(), "", "source", modelID.String(), "many-to-many", "rel", "INT[]")
		setup(f, modelID, modelID, srcCol, tenant.Column{}, nil, errors.New("boom"))

		meta := map[string]interface{}{"relation": map[string]interface{}{"with": modelID.String(), "type": "many-to-many"}}
		_, err := f.svc.AddColumn(context.Background(), relSchema, dto.AddColumnRequest{ModelID: modelID, BaseID: uuid.New(), Title: "Rel", UIDT: "links", Meta: meta})

		assert.Error(t, err)
		f.column.AssertNumberOfCalls(t, "DeleteColumn", 1)
		f.table.AssertNumberOfCalls(t, "AlterTable", 1)
	})
}

// ---------------------------------------------------------------------------
// Creating lookups
// ---------------------------------------------------------------------------

func TestAddLookupColumn(t *testing.T) {
	type env struct {
		f                 *relFixture
		modelID, linkedID string
		relationID        string
		created           *dto.ColumnInsertion
		updates           *[]dto.RelationUpdate
	}
	newEnv := func(selfLink bool) env {
		f := newRelFixture()
		e := env{f: f, modelID: uuid.NewString(), relationID: uuid.NewString(), created: &dto.ColumnInsertion{}, updates: &[]dto.RelationUpdate{}}
		e.linkedID = uuid.NewString()
		if selfLink {
			e.linkedID = e.modelID
		}
		f.column.On("Create", mock.Anything, mock.Anything, relSchema).Run(func(a mock.Arguments) {
			*e.created = a.Get(1).(dto.ColumnInsertion)
		}).Return(lookupCol(uuid.New(), e.modelID, "lk", nil), nil)
		f.rel.On("GetRelationByID", mock.Anything, e.relationID, relSchema).Return(tenant.Relation{SourceLookupColumns: []string{"a"}, TargetLookupColumns: []string{"b"}}, nil)
		f.rel.On("UpdateRelation", mock.Anything, e.relationID, mock.Anything, relSchema).Run(func(a mock.Arguments) {
			*e.updates = append(*e.updates, a.Get(2).(dto.RelationUpdate))
		}).Return(tenant.Relation{}, nil)
		return e
	}
	add := func(e env, meta map[string]interface{}) error {
		_, err := e.f.svc.AddColumn(context.Background(), relSchema, dto.AddColumnRequest{ModelID: uuid.MustParse(e.modelID), BaseID: uuid.New(), Title: "L", UIDT: "lookup", Meta: meta})
		return err
	}

	t.Run("through target-side link records on target side", func(t *testing.T) {
		e := newEnv(false)
		link := linkCol(uuid.New(), e.modelID, e.relationID, "target", e.linkedID, "has-many", "parent", "INT")
		foreign := plainCol(uuid.New(), e.linkedID, "name_123", "text")
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelID).Return([]tenant.Column{link}, nil)
		e.f.column.On("GetColumnByID", mock.Anything, relSchema, foreign.ID.String()).Return(foreign, nil)

		err := add(e, map[string]interface{}{"relation_id": e.relationID, "lookup_column_id": foreign.ID.String(), "link_column_id": link.ID.String()})

		assert.NoError(t, err)
		assert.True(t, strings.HasPrefix(e.created.ColumnName, "lk_"))
		assert.True(t, strings.HasSuffix(e.created.ColumnName, "_name_123"))
		assert.Equal(t, link.ID.String(), e.created.Meta["link_column_id"])
		if assert.Len(t, *e.updates, 1) {
			assert.Nil(t, (*e.updates)[0].SourceLookupColumns)
			assert.Equal(t, []string{"b", "name_123"}, (*e.updates)[0].TargetLookupColumns)
		}
	})

	t.Run("legacy self-link without link_column_id uses source side", func(t *testing.T) {
		e := newEnv(true)
		src := linkCol(uuid.New(), e.modelID, e.relationID, "source", e.modelID, "has-many", "reports", "INT[]")
		tgt := linkCol(uuid.New(), e.modelID, e.relationID, "target", e.modelID, "has-many", "manager", "INT")
		foreign := plainCol(uuid.New(), e.modelID, "name", "text")
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelID).Return([]tenant.Column{tgt, src}, nil)
		e.f.column.On("GetColumnByID", mock.Anything, relSchema, foreign.ID.String()).Return(foreign, nil)

		err := add(e, map[string]interface{}{"relation_id": e.relationID, "lookup_column_id": foreign.ID.String()})

		assert.NoError(t, err)
		assert.Equal(t, src.ID.String(), e.created.Meta["link_column_id"])
		if assert.Len(t, *e.updates, 1) {
			assert.Equal(t, []string{"a", "name"}, (*e.updates)[0].SourceLookupColumns)
			assert.Nil(t, (*e.updates)[0].TargetLookupColumns)
		}
	})

	t.Run("single-column self-link", func(t *testing.T) {
		e := newEnv(true)
		link := linkCol(uuid.New(), e.modelID, e.relationID, "source", e.modelID, "one-to-one", "mentor", "INT")
		foreign := plainCol(uuid.New(), e.modelID, "name", "text")
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelID).Return([]tenant.Column{link}, nil)
		e.f.column.On("GetColumnByID", mock.Anything, relSchema, foreign.ID.String()).Return(foreign, nil)

		err := add(e, map[string]interface{}{"relation_id": e.relationID, "lookup_column_id": foreign.ID.String(), "link_column_id": link.ID.String()})

		assert.NoError(t, err)
		if assert.Len(t, *e.updates, 1) {
			assert.NotNil(t, (*e.updates)[0].SourceLookupColumns)
		}
	})

	t.Run("link column from another table rejected", func(t *testing.T) {
		e := newEnv(false)
		link := linkCol(uuid.New(), e.modelID, e.relationID, "source", e.linkedID, "has-many", "orders", "INT[]")
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelID).Return([]tenant.Column{link}, nil)

		err := add(e, map[string]interface{}{"relation_id": e.relationID, "lookup_column_id": uuid.NewString(), "link_column_id": uuid.NewString()})

		assert.ErrorIs(t, err, app_errors.InvalidLookupLinkColumn)
		e.f.column.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("link column with different relation rejected", func(t *testing.T) {
		e := newEnv(false)
		link := linkCol(uuid.New(), e.modelID, uuid.NewString(), "source", e.linkedID, "has-many", "orders", "INT[]")
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelID).Return([]tenant.Column{link}, nil)

		err := add(e, map[string]interface{}{"relation_id": e.relationID, "lookup_column_id": uuid.NewString(), "link_column_id": link.ID.String()})

		assert.ErrorIs(t, err, app_errors.InvalidLookupLinkColumn)
	})

	t.Run("looked-up column not on the linked table rejected", func(t *testing.T) {
		e := newEnv(false)
		link := linkCol(uuid.New(), e.modelID, e.relationID, "source", e.linkedID, "has-many", "orders", "INT[]")
		foreign := plainCol(uuid.New(), uuid.NewString(), "name", "text")
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelID).Return([]tenant.Column{link}, nil)
		e.f.column.On("GetColumnByID", mock.Anything, relSchema, foreign.ID.String()).Return(foreign, nil)

		err := add(e, map[string]interface{}{"relation_id": e.relationID, "lookup_column_id": foreign.ID.String(), "link_column_id": link.ID.String()})

		assert.ErrorIs(t, err, app_errors.InvalidLookupLinkColumn)
	})

	for _, uidt := range []string{"links", "lookup", "rollup"} {
		t.Run("lookup of "+uidt+" rejected", func(t *testing.T) {
			e := newEnv(false)
			link := linkCol(uuid.New(), e.modelID, e.relationID, "source", e.linkedID, "has-many", "orders", "INT[]")
			foreign := plainCol(uuid.New(), e.linkedID, "virtual", uidt)
			e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelID).Return([]tenant.Column{link}, nil)
			e.f.column.On("GetColumnByID", mock.Anything, relSchema, foreign.ID.String()).Return(foreign, nil)

			err := add(e, map[string]interface{}{"relation_id": e.relationID, "lookup_column_id": foreign.ID.String(), "link_column_id": link.ID.String()})

			assert.ErrorIs(t, err, app_errors.InvalidLookupTargetColumn)
		})
	}

	t.Run("no link column for relation rejected", func(t *testing.T) {
		e := newEnv(false)
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelID).Return([]tenant.Column{}, nil)

		err := add(e, map[string]interface{}{"relation_id": e.relationID, "lookup_column_id": uuid.NewString()})

		assert.ErrorIs(t, err, app_errors.InvalidLookupLinkColumn)
	})
}

// ---------------------------------------------------------------------------
// Updating link and lookup columns
// ---------------------------------------------------------------------------

func TestUpdateLinkColumn_InverseTitle(t *testing.T) {
	cases := []struct {
		name          string
		role          string
		singleColumn  bool
		expectPartner bool
	}{
		{"source renames target partner", "source", false, true},
		{"target renames source partner", "target", false, true},
		{"single-column self-link has no partner", "source", true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newRelFixture()
			modelID, relationID := uuid.NewString(), uuid.NewString()
			col := linkCol(uuid.New(), modelID, relationID, c.role, uuid.NewString(), "many-to-many", "projects", "INT[]")
			partnerID := uuid.NewString()
			relation := tenant.Relation{SourceColumnID: partnerID, TargetColumnID: partnerID}
			if c.singleColumn {
				relation = tenant.Relation{SourceColumnID: col.ID.String(), TargetColumnID: col.ID.String()}
			} else if c.role == "source" {
				relation.SourceColumnID = col.ID.String()
			} else {
				relation.TargetColumnID = col.ID.String()
			}

			f.column.On("GetColumnByID", mock.Anything, relSchema, col.ID.String()).Return(col, nil)
			f.column.On("UpdateColumn", mock.Anything, relSchema, col.ID.String(), mock.MatchedBy(func(r dto.ColumnUpdate) bool {
				return r.Title != nil && *r.Title == "Assigned" && r.Meta == nil && r.UIDT == nil
			})).Return(col, nil)
			f.column.On("UpdateColumn", mock.Anything, relSchema, partnerID, mock.MatchedBy(func(r dto.ColumnUpdate) bool {
				return r.Title != nil && *r.Title == "Team"
			})).Return(tenant.Column{}, nil)
			f.rel.On("GetRelationByID", mock.Anything, relationID, relSchema).Return(relation, nil)

			meta := map[string]interface{}{"relation": map[string]interface{}{"inverse_title": "Team"}}
			_, err := f.svc.UpdateColumn(context.Background(), relSchema, col.ID.String(), dto.ColumnUpdate{Title: helpers.StringPtr("Assigned"), UIDT: helpers.StringPtr("text"), Meta: &meta})

			assert.NoError(t, err)
			if c.expectPartner {
				f.column.AssertCalled(t, "UpdateColumn", mock.Anything, relSchema, partnerID, mock.Anything)
			} else {
				f.column.AssertNumberOfCalls(t, "UpdateColumn", 1)
			}
		})
	}
}

func TestUpdateColumn_CannotConvertToLinkOrLookup(t *testing.T) {
	for _, uidt := range []string{"links", "lookup"} {
		t.Run(uidt, func(t *testing.T) {
			f := newRelFixture()
			col := plainCol(uuid.New(), uuid.NewString(), "name", "text")
			f.column.On("GetColumnByID", mock.Anything, relSchema, col.ID.String()).Return(col, nil)

			_, err := f.svc.UpdateColumn(context.Background(), relSchema, col.ID.String(), dto.ColumnUpdate{UIDT: helpers.StringPtr(uidt)})

			assert.ErrorIs(t, err, app_errors.UpdateNotAllowed)
			f.column.AssertNotCalled(t, "UpdateColumn", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestUpdateLookupColumn(t *testing.T) {
	type env struct {
		f                           *relFixture
		modelID, linkedID, relation string
		link                        tenant.Column
		foreign, other              tenant.Column
		lookup                      tenant.Column
		update                      *dto.ColumnUpdate
	}
	newEnv := func() env {
		f := newRelFixture()
		e := env{f: f, modelID: uuid.NewString(), linkedID: uuid.NewString(), relation: uuid.NewString(), update: &dto.ColumnUpdate{}}
		e.link = linkCol(uuid.New(), e.modelID, e.relation, "source", e.linkedID, "has-many", "orders", "INT[]")
		e.foreign = plainCol(uuid.New(), e.linkedID, "name", "text")
		e.other = plainCol(uuid.New(), e.linkedID, "email", "email")
		e.lookup = lookupCol(uuid.New(), e.modelID, "lk_old_name", map[string]interface{}{
			"relation_id": e.relation, "lookup_column_id": e.foreign.ID.String(), "link_column_id": e.link.ID.String(),
		})
		f.column.On("GetColumnByID", mock.Anything, relSchema, e.lookup.ID.String()).Return(e.lookup, nil)
		f.column.On("GetColumnByID", mock.Anything, relSchema, e.foreign.ID.String()).Return(e.foreign, nil)
		f.column.On("GetColumnByID", mock.Anything, relSchema, e.other.ID.String()).Return(e.other, nil)
		f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelID).Return([]tenant.Column{e.link}, nil)
		f.column.On("UpdateColumn", mock.Anything, relSchema, e.lookup.ID.String(), mock.Anything).Run(func(a mock.Arguments) {
			*e.update = a.Get(3).(dto.ColumnUpdate)
		}).Return(e.lookup, nil)
		f.rel.On("GetRelationByID", mock.Anything, e.relation, relSchema).Return(tenant.Relation{SourceLookupColumns: []string{"name"}}, nil)
		f.rel.On("UpdateRelation", mock.Anything, e.relation, mock.Anything, relSchema).Return(tenant.Relation{}, nil)
		return e
	}
	update := func(e env, req dto.ColumnUpdate) error {
		_, err := e.f.svc.UpdateColumn(context.Background(), relSchema, e.lookup.ID.String(), req)
		return err
	}

	t.Run("title only leaves definition alone", func(t *testing.T) {
		e := newEnv()
		err := update(e, dto.ColumnUpdate{Title: helpers.StringPtr("Boss")})

		assert.NoError(t, err)
		assert.Equal(t, "Boss", *e.update.Title)
		assert.Nil(t, e.update.Meta)
		assert.Nil(t, e.update.ColumnName)
		e.f.rel.AssertNotCalled(t, "UpdateRelation", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("same definition does not touch relation", func(t *testing.T) {
		e := newEnv()
		meta := map[string]interface{}{"relation_id": e.relation, "lookup_column_id": e.foreign.ID.String(), "link_column_id": e.link.ID.String()}
		err := update(e, dto.ColumnUpdate{Meta: &meta})

		assert.NoError(t, err)
		assert.Nil(t, e.update.ColumnName)
		e.f.rel.AssertNotCalled(t, "UpdateRelation", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("new looked-up column gets new name and moves relation entry", func(t *testing.T) {
		e := newEnv()
		meta := map[string]interface{}{"relation_id": e.relation, "lookup_column_id": e.other.ID.String(), "link_column_id": e.link.ID.String()}
		err := update(e, dto.ColumnUpdate{Meta: &meta})

		assert.NoError(t, err)
		if assert.NotNil(t, e.update.ColumnName) {
			assert.True(t, strings.HasPrefix(*e.update.ColumnName, "lk_"))
			assert.True(t, strings.HasSuffix(*e.update.ColumnName, "_email"))
		}
		e.f.rel.AssertNumberOfCalls(t, "UpdateRelation", 2) // remove old + add new
	})

	t.Run("invalid new definition changes nothing", func(t *testing.T) {
		e := newEnv()
		bad := plainCol(uuid.New(), e.linkedID, "orders", "links")
		e.f.column.On("GetColumnByID", mock.Anything, relSchema, bad.ID.String()).Return(bad, nil)
		meta := map[string]interface{}{"relation_id": e.relation, "lookup_column_id": bad.ID.String(), "link_column_id": e.link.ID.String()}

		err := update(e, dto.ColumnUpdate{Meta: &meta})

		assert.ErrorIs(t, err, app_errors.InvalidLookupTargetColumn)
		e.f.rel.AssertNotCalled(t, "UpdateRelation", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		e.f.column.AssertNotCalled(t, "UpdateColumn", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

// ---------------------------------------------------------------------------
// Read path
// ---------------------------------------------------------------------------

func TestGetRecordsWithLookups_BuildsRelationData(t *testing.T) {
	f := newRelFixture()
	modelID, customersID := uuid.NewString(), uuid.NewString()
	billing := linkCol(uuid.New(), modelID, uuid.NewString(), "source", customersID, "one-to-one", "billing", "INT")
	shipping := linkCol(uuid.New(), modelID, uuid.NewString(), "source", customersID, "one-to-one", "shipping", "INT")
	items := linkCol(uuid.New(), modelID, uuid.NewString(), "source", customersID, "many-to-many", "items", "INT[]")
	name := plainCol(uuid.New(), customersID, "name", "text")
	email := plainCol(uuid.New(), customersID, "email", "text")
	elsewhere := plainCol(uuid.New(), uuid.NewString(), "x", "text")

	lk := func(colName string, link tenant.Column, foreign tenant.Column) tenant.Column {
		return lookupCol(uuid.New(), modelID, colName, map[string]interface{}{
			"relation_id": link.Meta["relation_id"], "lookup_column_id": foreign.ID.String(), "link_column_id": link.ID.String(),
		})
	}
	columns := []dto.ColumnResponse{
		toResponse(billing), toResponse(shipping), toResponse(items),
		toResponse(lk("lk_1_name", billing, name)),
		toResponse(lk("lk_2_name", shipping, name)),
		toResponse(lk("lk_3_name", items, name)),
		toResponse(lk("lk_4_email", items, email)),
		toResponse(lk("lk_5_x", items, elsewhere)), // foreign column on another table: skipped
		toResponse(lookupCol(uuid.New(), modelID, "bad_meta", map[string]interface{}{"relation_id": "nope"})),
		toResponse(lookupCol(uuid.New(), modelID, "no_link", map[string]interface{}{"relation_id": uuid.NewString(), "lookup_column_id": name.ID.String()})),
	}

	f.column.On("GetColumnByID", mock.Anything, relSchema, name.ID.String()).Return(name, nil)
	f.column.On("GetColumnByID", mock.Anything, relSchema, email.ID.String()).Return(email, nil)
	f.column.On("GetColumnByID", mock.Anything, relSchema, elsewhere.ID.String()).Return(elsewhere, nil)
	f.model.On("GetModelByID", mock.Anything, relSchema, customersID).Return(tenant.Model{Alias: "customers"}, nil).Once()

	var relationData []map[string]interface{}
	f.table.On("GetByFunction", mock.Anything, "public.get_table_data_with_lookups", mock.Anything).Run(func(a mock.Arguments) {
		relationData = a.Get(2).(map[string]interface{})["relation_data"].([]map[string]interface{})
	}).Return([]map[string]interface{}{{"get_table_data_with_lookups": []interface{}{map[string]interface{}{"id": 1}}}}, nil)

	records, err := f.svc.(recordsGetter).GetRecordsWithLookups(context.Background(), relSchema, "orders", columns)

	assert.NoError(t, err)
	assert.Len(t, records.Records, 1)
	if assert.Len(t, relationData, 3) {
		bySource := map[string]map[string]interface{}{}
		for _, entry := range relationData {
			bySource[entry["source_column_name"].(string)] = entry
		}
		assert.Equal(t, []map[string]interface{}{{"column": "name", "alias": "lk_1_name"}}, bySource["billing"]["targets"])
		assert.Equal(t, []map[string]interface{}{{"column": "name", "alias": "lk_2_name"}}, bySource["shipping"]["targets"])
		assert.Equal(t, []map[string]interface{}{{"column": "name", "alias": "lk_3_name"}, {"column": "email", "alias": "lk_4_email"}}, bySource["items"]["targets"])
		assert.Equal(t, true, bySource["items"]["is_array"])
		assert.Equal(t, false, bySource["billing"]["is_array"])
		assert.Equal(t, "customers", bySource["items"]["target_table_name"])
		assert.Equal(t, "id", bySource["items"]["target_column_name"])
	}
}

func TestGetRecordsWithLookups_NoLookupsAndErrors(t *testing.T) {
	t.Run("no lookups sends empty relation data", func(t *testing.T) {
		f := newRelFixture()
		var relationData []map[string]interface{}
		f.table.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Run(func(a mock.Arguments) {
			relationData = a.Get(2).(map[string]interface{})["relation_data"].([]map[string]interface{})
		}).Return([]map[string]interface{}{}, nil)

		records, err := f.svc.(recordsGetter).GetRecordsWithLookups(context.Background(), relSchema, "t", []dto.ColumnResponse{toResponse(plainCol(uuid.New(), uuid.NewString(), "a", "text"))})

		assert.NoError(t, err)
		assert.Nil(t, records.Records)
		assert.Empty(t, relationData)
	})

	t.Run("function error is returned", func(t *testing.T) {
		f := newRelFixture()
		f.table.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db down"))

		_, err := f.svc.(recordsGetter).GetRecordsWithLookups(context.Background(), relSchema, "t", nil)

		assert.Error(t, err)
	})

	t.Run("missing model or foreign column skips the lookup", func(t *testing.T) {
		f := newRelFixture()
		modelID, otherID := uuid.NewString(), uuid.NewString()
		link := linkCol(uuid.New(), modelID, uuid.NewString(), "source", otherID, "has-many", "l", "INT[]")
		foreignOK := plainCol(uuid.New(), otherID, "n", "text")
		missing := uuid.NewString()
		columns := []dto.ColumnResponse{
			toResponse(link),
			toResponse(lookupCol(uuid.New(), modelID, "a", map[string]interface{}{"relation_id": link.Meta["relation_id"], "lookup_column_id": missing, "link_column_id": link.ID.String()})),
			toResponse(lookupCol(uuid.New(), modelID, "b", map[string]interface{}{"relation_id": link.Meta["relation_id"], "lookup_column_id": foreignOK.ID.String(), "link_column_id": link.ID.String()})),
		}
		f.column.On("GetColumnByID", mock.Anything, relSchema, missing).Return(tenant.Column{}, app_errors.ColumnNotFound)
		f.column.On("GetColumnByID", mock.Anything, relSchema, foreignOK.ID.String()).Return(foreignOK, nil)
		f.model.On("GetModelByID", mock.Anything, relSchema, otherID).Return(tenant.Model{}, errors.New("gone"))
		var relationData []map[string]interface{}
		f.table.On("GetByFunction", mock.Anything, mock.Anything, mock.Anything).Run(func(a mock.Arguments) {
			relationData = a.Get(2).(map[string]interface{})["relation_data"].([]map[string]interface{})
		}).Return([]map[string]interface{}{}, nil)

		_, err := f.svc.(recordsGetter).GetRecordsWithLookups(context.Background(), relSchema, "t", columns)

		assert.NoError(t, err)
		assert.Empty(t, relationData)
	})
}

// ---------------------------------------------------------------------------
// Linking rows (public.link_rows)
// ---------------------------------------------------------------------------

type linkEnv struct {
	f        *relFixture
	modelA   string
	modelB   string
	relation tenant.Relation
	colA     tenant.Column
	colB     tenant.Column
}

// newTwoTableLink: table A (alias "a", column "a_link", role source) linked to table B ("b", "b_link").
func newTwoTableLink(relType, dtA, dtB string) linkEnv {
	f := newRelFixture()
	e := linkEnv{f: f, modelA: uuid.NewString(), modelB: uuid.NewString()}
	relationID := uuid.New()
	e.colA = linkCol(uuid.New(), e.modelA, relationID.String(), "source", e.modelB, relType, "a_link", dtA)
	e.colB = linkCol(uuid.New(), e.modelB, relationID.String(), "target", e.modelA, relType, "b_link", dtB)
	e.relation = tenant.Relation{ID: relationID, SourceModelID: e.modelA, TargetModelID: e.modelB, SourceColumnID: e.colA.ID.String(), TargetColumnID: e.colB.ID.String(), RelationType: relType}
	f.column.On("GetColumnByID", mock.Anything, relSchema, e.colA.ID.String()).Return(e.colA, nil)
	f.column.On("GetColumnByID", mock.Anything, relSchema, e.colB.ID.String()).Return(e.colB, nil)
	f.model.On("GetModelByID", mock.Anything, relSchema, e.modelA).Return(tenant.Model{ID: uuid.MustParse(e.modelA), Alias: "a"}, nil)
	f.model.On("GetModelByID", mock.Anything, relSchema, e.modelB).Return(tenant.Model{ID: uuid.MustParse(e.modelB), Alias: "b"}, nil)
	f.rel.On("GetRelationByID", mock.Anything, relationID.String(), relSchema).Return(e.relation, nil)
	f.table.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{{"id": int64(1)}}, nil)
	return e
}

// newSelfLink: a single-column self-link on table "emp", column "emp_link".
func newSelfLink(relType, dt string) linkEnv {
	f := newRelFixture()
	e := linkEnv{f: f, modelA: uuid.NewString()}
	e.modelB = e.modelA
	relationID := uuid.New()
	e.colA = linkCol(uuid.New(), e.modelA, relationID.String(), "source", e.modelA, relType, "emp_link", dt)
	e.colA.Meta["is_self_link"] = true
	e.relation = tenant.Relation{ID: relationID, SourceModelID: e.modelA, TargetModelID: e.modelA, SourceColumnID: e.colA.ID.String(), TargetColumnID: e.colA.ID.String(), RelationType: relType}
	f.column.On("GetColumnByID", mock.Anything, relSchema, e.colA.ID.String()).Return(e.colA, nil)
	f.model.On("GetModelByID", mock.Anything, relSchema, e.modelA).Return(tenant.Model{ID: uuid.MustParse(e.modelA), Alias: "emp"}, nil)
	f.rel.On("GetRelationByID", mock.Anything, relationID.String(), relSchema).Return(e.relation, nil)
	f.table.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{{"id": int64(1)}}, nil)
	return e
}

func (e linkEnv) link(source, target int, action string) (dto.RecordResponse, error) {
	return e.f.svc.UpdateRawDataForLinks(context.Background(), relSchema, dto.UpdateRowDataLinksRequest{
		ModelID: e.modelA, ColumnId: e.colA.ID.String(), SourceRowId: source, TargetRowId: target, Action: action, UpdatedBy: "u",
	})
}

// expectFunction makes the named SQL function return result and records the arguments of each call.
func (f *relFixture) expectFunction(name string, result interface{}) *[]map[string]interface{} {
	calls := &[]map[string]interface{}{}
	f.table.On("GetByFunction", mock.Anything, "public."+name, mock.Anything).Run(func(a mock.Arguments) {
		*calls = append(*calls, a.Get(2).(map[string]interface{}))
	}).Return([]map[string]interface{}{{name: result}}, nil)
	return calls
}

// assertArgsMatchFunction checks the Go call passes exactly the parameters the SQL function declares.
func assertArgsMatchFunction(t *testing.T, name string, args map[string]interface{}) {
	t.Helper()
	var params []string
	for _, fn := range constant.DefinedFunctions {
		if fn.FunctionName != name {
			continue
		}
		for _, p := range strings.Split(fn.FunctionParams, ",") {
			params = append(params, strings.Fields(p)[0])
		}
	}
	var keys []string
	for k := range args {
		keys = append(keys, k)
	}
	assert.ElementsMatch(t, params, keys, "arguments for %s", name)
}

func TestUpdateRawDataForLinks_TwoTables(t *testing.T) {
	t.Run("link sends both sides to link_rows", func(t *testing.T) {
		e := newTwoTableLink("many-to-many", "INT[]", "INT[]")
		calls := e.f.expectFunction("link_rows", "ok")

		resp, err := e.link(1, 2, "link")

		assert.NoError(t, err)
		assert.NotNil(t, resp.Record)
		assert.NotNil(t, resp.RelatedRecord)
		if assert.Len(t, *calls, 1) {
			args := (*calls)[0]
			assertArgsMatchFunction(t, "link_rows", args)
			assert.Equal(t, map[string]interface{}{
				"p_relation_type":   "many-to-many",
				"p_single_column":   false,
				"p_is_self":         false,
				"p_action":          "link",
				"p_source_table":    `"schema"."a"`,
				"p_source_column":   "a_link",
				"p_source_is_array": true,
				"p_target_table":    `"schema"."b"`,
				"p_target_column":   "b_link",
				"p_target_is_array": true,
				"p_source_row":      int64(1),
				"p_target_row":      int64(2),
				"p_updated_by":      "u",
			}, args)
		}
	})

	t.Run("unlink", func(t *testing.T) {
		e := newTwoTableLink("one-to-one", "INT", "INT")
		calls := e.f.expectFunction("link_rows", "ok")

		_, err := e.link(1, 2, "unlink")

		assert.NoError(t, err)
		assert.Equal(t, "unlink", (*calls)[0]["p_action"])
		assert.Equal(t, false, (*calls)[0]["p_source_is_array"])
	})

	t.Run("has-many from the child side", func(t *testing.T) {
		// A holds the children (INT[]), B holds the parent (INT). Link from the child column on B.
		e := newTwoTableLink("has-many", "INT[]", "INT")
		calls := e.f.expectFunction("link_rows", "ok")

		_, err := e.f.svc.UpdateRawDataForLinks(context.Background(), relSchema, dto.UpdateRowDataLinksRequest{
			ModelID: e.modelB, ColumnId: e.colB.ID.String(), SourceRowId: 1, TargetRowId: 2, Action: "link",
		})

		assert.NoError(t, err)
		args := (*calls)[0]
		assert.Equal(t, `"schema"."b"`, args["p_source_table"])
		assert.Equal(t, "b_link", args["p_source_column"])
		assert.Equal(t, false, args["p_source_is_array"])
		assert.Equal(t, `"schema"."a"`, args["p_target_table"])
		assert.Equal(t, true, args["p_target_is_array"])
		assert.Equal(t, "", args["p_updated_by"])
	})

	statuses := []struct {
		result interface{}
		want   error
	}{
		{"source_not_found", app_errors.RowNotFound},
		{"target_not_found", app_errors.LinkTargetRowNotFound},
		{"cycle", app_errors.LinkCycleDetected},
		{"something else", app_errors.DatabaseError},
		{nil, app_errors.DatabaseError},
	}
	for _, s := range statuses {
		t.Run("status "+functionStatusName(s.result), func(t *testing.T) {
			e := newTwoTableLink("many-to-many", "INT[]", "INT[]")
			e.f.expectFunction("link_rows", s.result)

			_, err := e.link(1, 2, "link")

			assert.ErrorIs(t, err, s.want)
			e.f.table.AssertNotCalled(t, "GetTableData", mock.Anything, mock.Anything)
		})
	}

	t.Run("byte status from the driver", func(t *testing.T) {
		e := newTwoTableLink("many-to-many", "INT[]", "INT[]")
		e.f.expectFunction("link_rows", []byte("ok"))

		_, err := e.link(1, 2, "link")

		assert.NoError(t, err)
	})

	t.Run("function error", func(t *testing.T) {
		e := newTwoTableLink("many-to-many", "INT[]", "INT[]")
		e.f.table.On("GetByFunction", mock.Anything, "public.link_rows", mock.Anything).Return(nil, errors.New("db down"))

		_, err := e.link(1, 2, "link")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("function returned no row", func(t *testing.T) {
		e := newTwoTableLink("many-to-many", "INT[]", "INT[]")
		e.f.table.On("GetByFunction", mock.Anything, "public.link_rows", mock.Anything).Return([]map[string]interface{}{}, nil)

		_, err := e.link(1, 2, "link")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("linked row reload failure is not fatal", func(t *testing.T) {
		e := newTwoTableLink("many-to-many", "INT[]", "INT[]")
		e.f.expectFunction("link_rows", "ok")
		e.f.table.ExpectedCalls = filterCalls(e.f.table.ExpectedCalls, "GetTableData")
		e.f.table.On("GetTableData", `"schema"."a"`, mock.Anything).Return([]map[string]interface{}{{"id": int64(1)}}, nil)
		e.f.table.On("GetTableData", `"schema"."b"`, mock.Anything).Return([]map[string]interface{}{}, nil)

		resp, err := e.link(1, 2, "link")

		assert.NoError(t, err)
		assert.NotNil(t, resp.Record)
		assert.Nil(t, resp.RelatedRecord)
	})

	t.Run("column on another table rejected", func(t *testing.T) {
		e := newTwoTableLink("many-to-many", "INT[]", "INT[]")
		_, err := e.f.svc.UpdateRawDataForLinks(context.Background(), relSchema, dto.UpdateRowDataLinksRequest{
			ModelID: e.modelB, ColumnId: e.colA.ID.String(), SourceRowId: 1, TargetRowId: 2, Action: "link",
		})
		assert.ErrorIs(t, err, app_errors.InvalidColumnMetaForLinkType)
		e.f.table.AssertNotCalled(t, "GetByFunction", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("non-link column rejected", func(t *testing.T) {
		f := newRelFixture()
		col := plainCol(uuid.New(), uuid.NewString(), "name", "text")
		f.column.On("GetColumnByID", mock.Anything, relSchema, col.ID.String()).Return(col, nil)
		_, err := f.svc.UpdateRawDataForLinks(context.Background(), relSchema, dto.UpdateRowDataLinksRequest{
			ModelID: col.ModelID, ColumnId: col.ID.String(), SourceRowId: 1, TargetRowId: 2, Action: "link",
		})
		assert.ErrorIs(t, err, app_errors.InvalidColumnMetaForLinkType)
	})
}

func TestUpdateRawDataForLinks_SelfLink(t *testing.T) {
	t.Run("row cannot link to itself", func(t *testing.T) {
		e := newSelfLink("many-to-many", "INT[]")
		_, err := e.link(3, 3, "link")

		assert.ErrorIs(t, err, app_errors.SelfReferenceNotAllowed)
		e.f.table.AssertNotCalled(t, "GetByFunction", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("unlinking a row from itself is allowed", func(t *testing.T) {
		e := newSelfLink("many-to-many", "INT[]")
		e.f.expectFunction("link_rows", "ok")

		_, err := e.link(3, 3, "unlink")

		assert.NoError(t, err)
	})

	for _, c := range []struct{ relType, dt string }{{"one-to-one", "INT"}, {"has-many", "INT[]"}, {"many-to-many", "INT[]"}} {
		t.Run("single column "+c.relType, func(t *testing.T) {
			e := newSelfLink(c.relType, c.dt)
			calls := e.f.expectFunction("link_rows", "ok")

			_, err := e.link(1, 2, "link")

			assert.NoError(t, err)
			args := (*calls)[0]
			assertArgsMatchFunction(t, "link_rows", args)
			assert.Equal(t, true, args["p_single_column"])
			assert.Equal(t, true, args["p_is_self"])
			assert.Equal(t, c.relType, args["p_relation_type"])
			assert.Equal(t, args["p_source_table"], args["p_target_table"])
			assert.Equal(t, args["p_source_column"], args["p_target_column"])
			assert.Equal(t, c.dt == "INT[]", args["p_source_is_array"])
		})
	}

	t.Run("legacy two-column self-link", func(t *testing.T) {
		e := newTwoTableLink("has-many", "INT[]", "INT")
		e.relation.TargetModelID = e.modelA
		e.f.rel.ExpectedCalls = nil
		e.f.rel.On("GetRelationByID", mock.Anything, e.relation.ID.String(), relSchema).Return(e.relation, nil)
		calls := e.f.expectFunction("link_rows", "cycle")

		_, err := e.link(1, 2, "link")

		assert.ErrorIs(t, err, app_errors.LinkCycleDetected)
		assert.Equal(t, false, (*calls)[0]["p_single_column"])
		assert.Equal(t, true, (*calls)[0]["p_is_self"])
	})
}

func functionStatusName(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return "nil"
}

func filterCalls(calls []*mock.Call, method string) []*mock.Call {
	var out []*mock.Call
	for _, c := range calls {
		if c.Method != method {
			out = append(out, c)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Generic writes, row deletes (public.remove_link_back_references)
// ---------------------------------------------------------------------------

func TestInsertRowData_RejectsLinkAndLookup(t *testing.T) {
	for _, uidt := range []string{"links", "lookup"} {
		t.Run(uidt, func(t *testing.T) {
			f := newRelFixture()
			col := plainCol(uuid.New(), uuid.NewString(), "c", uidt)
			f.column.On("GetColumnByID", mock.Anything, relSchema, col.ID.String()).Return(col, nil)
			val := interface{}([]int{1})

			_, err := f.svc.InsertRowData(context.Background(), relSchema, dto.InsertRowDataRequest{ModelID: col.ModelID, ColumnId: col.ID.String(), RowId: 1, Value: &val})

			assert.ErrorIs(t, err, app_errors.LinkColumnNotWritable)
			f.table.AssertNotCalled(t, "UpdateRecord", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestDeleteRow_RemovesBackReferences(t *testing.T) {
	t.Run("two-table link partner", func(t *testing.T) {
		e := newTwoTableLink("many-to-many", "INT[]", "INT[]")
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelA).Return([]tenant.Column{e.colA, plainCol(uuid.New(), e.modelA, "name", "text")}, nil)
		calls := e.f.expectFunction("remove_link_back_references", nil)
		e.f.table.On("DeleteRecord", mock.Anything, 1).Return(nil)

		err := e.f.svc.DeleteRow(context.Background(), relSchema, dto.DeleteRowDataRequest{ModelID: e.modelA, RowId: 1})

		assert.NoError(t, err)
		if assert.Len(t, *calls, 1) {
			args := (*calls)[0]
			assertArgsMatchFunction(t, "remove_link_back_references", args)
			assert.Equal(t, []map[string]interface{}{{"table": `"schema"."b"`, "column": "b_link", "is_array": true}}, args["p_partners"])
			assert.Equal(t, []int64{1}, args["p_row_ids"])
			assert.Equal(t, "", args["p_updated_by"])
		}
		e.f.table.AssertCalled(t, "DeleteRecord", mock.Anything, 1)
	})

	t.Run("target-side link points back at the source column", func(t *testing.T) {
		e := newTwoTableLink("has-many", "INT[]", "INT")
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelB).Return([]tenant.Column{e.colB}, nil)
		calls := e.f.expectFunction("remove_link_back_references", nil)
		e.f.table.On("DeleteRecord", mock.Anything, 5).Return(nil)

		err := e.f.svc.DeleteRow(context.Background(), relSchema, dto.DeleteRowDataRequest{ModelID: e.modelB, RowId: 5})

		assert.NoError(t, err)
		assert.Equal(t, []map[string]interface{}{{"table": `"schema"."a"`, "column": "a_link", "is_array": true}}, (*calls)[0]["p_partners"])
	})

	t.Run("single-column self-link uses its own column", func(t *testing.T) {
		e := newSelfLink("one-to-one", "INT")
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelA).Return([]tenant.Column{e.colA}, nil)
		calls := e.f.expectFunction("remove_link_back_references", nil)
		e.f.table.On("DeleteRecord", mock.Anything, 1).Return(nil)

		err := e.f.svc.DeleteRow(context.Background(), relSchema, dto.DeleteRowDataRequest{ModelID: e.modelA, RowId: 1})

		assert.NoError(t, err)
		assert.Equal(t, []map[string]interface{}{{"table": `"schema"."emp"`, "column": "emp_link", "is_array": false}}, (*calls)[0]["p_partners"])
	})

	t.Run("table without links skips the function", func(t *testing.T) {
		f := newRelFixture()
		modelID := uuid.New()
		f.model.On("GetModelByID", mock.Anything, relSchema, modelID.String()).Return(tenant.Model{ID: modelID, Alias: "t"}, nil)
		f.table.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{{"id": int64(1)}}, nil)
		f.column.On("GetColumnByModelID", mock.Anything, relSchema, modelID.String()).Return([]tenant.Column{plainCol(uuid.New(), modelID.String(), "name", "text")}, nil)
		f.table.On("DeleteRecord", mock.Anything, 1).Return(nil)

		err := f.svc.DeleteRow(context.Background(), relSchema, dto.DeleteRowDataRequest{ModelID: modelID.String(), RowId: 1})

		assert.NoError(t, err)
		f.table.AssertNotCalled(t, "GetByFunction", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("row not found", func(t *testing.T) {
		f := newRelFixture()
		modelID := uuid.NewString()
		f.model.On("GetModelByID", mock.Anything, relSchema, modelID).Return(tenant.Model{Alias: "t"}, nil)
		f.table.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

		err := f.svc.DeleteRow(context.Background(), relSchema, dto.DeleteRowDataRequest{ModelID: modelID, RowId: 1})

		assert.ErrorIs(t, err, app_errors.RowNotFound)
		f.table.AssertNotCalled(t, "GetByFunction", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("cleanup failure keeps the row", func(t *testing.T) {
		e := newTwoTableLink("many-to-many", "INT[]", "INT[]")
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelA).Return([]tenant.Column{e.colA}, nil)
		e.f.table.On("GetByFunction", mock.Anything, "public.remove_link_back_references", mock.Anything).Return(nil, errors.New("write failed"))

		err := e.f.svc.DeleteRow(context.Background(), relSchema, dto.DeleteRowDataRequest{ModelID: e.modelA, RowId: 1})

		assert.ErrorIs(t, err, app_errors.DatabaseError)
		e.f.table.AssertNotCalled(t, "DeleteRecord", mock.Anything, mock.Anything)
	})

	t.Run("partner lookup failure keeps the row", func(t *testing.T) {
		e := newTwoTableLink("many-to-many", "INT[]", "INT[]")
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelA).Return([]tenant.Column{e.colA}, nil)
		e.f.rel.ExpectedCalls = nil
		e.f.rel.On("GetRelationByID", mock.Anything, e.relation.ID.String(), relSchema).Return(tenant.Relation{}, errors.New("gone"))

		err := e.f.svc.DeleteRow(context.Background(), relSchema, dto.DeleteRowDataRequest{ModelID: e.modelA, RowId: 1})

		assert.Error(t, err)
		e.f.table.AssertNotCalled(t, "DeleteRecord", mock.Anything, mock.Anything)
	})
}

func TestBulkDeleteRows_RemovesBackReferences(t *testing.T) {
	t.Run("one cleanup call for all rows", func(t *testing.T) {
		e := newTwoTableLink("has-many", "INT[]", "INT")
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelA).Return([]tenant.Column{e.colA}, nil)
		calls := e.f.expectFunction("remove_link_back_references", nil)
		e.f.bulk.On("BulkDelete", mock.Anything, []interface{}{1, 2}, "id").Return(int64(2), nil)

		count, err := e.f.svc.BulkDeleteRows(context.Background(), relSchema, dto.BulkDeleteRowsRequest{ModelID: e.modelA, RowIds: []int{1, 2}})

		assert.NoError(t, err)
		assert.Equal(t, 2, count)
		if assert.Len(t, *calls, 1) {
			assert.Equal(t, []int64{1, 2}, (*calls)[0]["p_row_ids"])
			assert.Equal(t, []map[string]interface{}{{"table": `"schema"."b"`, "column": "b_link", "is_array": false}}, (*calls)[0]["p_partners"])
		}
	})

	t.Run("cleanup failure skips bulk delete", func(t *testing.T) {
		e := newTwoTableLink("has-many", "INT[]", "INT")
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelA).Return([]tenant.Column{e.colA}, nil)
		e.f.table.On("GetByFunction", mock.Anything, "public.remove_link_back_references", mock.Anything).Return(nil, errors.New("write failed"))

		_, err := e.f.svc.BulkDeleteRows(context.Background(), relSchema, dto.BulkDeleteRowsRequest{ModelID: e.modelA, RowIds: []int{1}})

		assert.Error(t, err)
		e.f.bulk.AssertNotCalled(t, "BulkDelete", mock.Anything, mock.Anything, mock.Anything)
	})
}

// ---------------------------------------------------------------------------
// Deleting columns and tables
// ---------------------------------------------------------------------------

func TestDeleteLinkColumn_OnlyItsLookups(t *testing.T) {
	t.Run("two-table link", func(t *testing.T) {
		e := newTwoTableLink("has-many", "INT[]", "INT")
		order := 3.0
		mine := lookupCol(uuid.New(), e.modelA, "lk_mine", map[string]interface{}{"relation_id": e.relation.ID.String(), "lookup_column_id": uuid.NewString()})
		mine.OrderIndex = &order
		other := lookupCol(uuid.New(), e.modelA, "lk_other", map[string]interface{}{"relation_id": uuid.NewString(), "lookup_column_id": uuid.NewString()})
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelA).Return([]tenant.Column{e.colA, mine, other}, nil)
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelB).Return([]tenant.Column{e.colB}, nil)
		e.f.column.On("DeleteColumn", mock.Anything, relSchema, mock.Anything).Return(nil)
		e.f.table.On("AlterTable", mock.Anything, mock.Anything).Return(nil)
		e.f.table.On("GetByFunction", mock.Anything, "public.reorder_columns_after_delete", mock.Anything).Return([]map[string]interface{}{}, nil)

		err := e.f.svc.DeleteColumn(context.Background(), relSchema, e.colA.ID.String())

		assert.NoError(t, err)
		e.f.column.AssertCalled(t, "DeleteColumn", mock.Anything, relSchema, e.colA.ID.String())
		e.f.column.AssertCalled(t, "DeleteColumn", mock.Anything, relSchema, e.colB.ID.String())
		e.f.column.AssertCalled(t, "DeleteColumn", mock.Anything, relSchema, mine.ID.String())
		e.f.column.AssertNotCalled(t, "DeleteColumn", mock.Anything, relSchema, other.ID.String())
		e.f.table.AssertNumberOfCalls(t, "AlterTable", 2)
	})

	t.Run("single-column self-link has no partner", func(t *testing.T) {
		e := newSelfLink("has-many", "INT[]")
		e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.modelA).Return([]tenant.Column{e.colA}, nil)
		e.f.column.On("DeleteColumn", mock.Anything, relSchema, e.colA.ID.String()).Return(nil)
		e.f.table.On("AlterTable", mock.Anything, mock.Anything).Return(nil)

		err := e.f.svc.DeleteColumn(context.Background(), relSchema, e.colA.ID.String())

		assert.NoError(t, err)
		e.f.column.AssertNumberOfCalls(t, "DeleteColumn", 1)
		e.f.table.AssertNumberOfCalls(t, "AlterTable", 1)
	})
}

func TestDeleteLookupColumn(t *testing.T) {
	t.Run("reorders remaining columns", func(t *testing.T) {
		f := newRelFixture()
		modelID, relationID := uuid.NewString(), uuid.NewString()
		order := 2.0
		foreign := plainCol(uuid.New(), uuid.NewString(), "name", "text")
		lk := lookupCol(uuid.New(), modelID, "lk", map[string]interface{}{"relation_id": relationID, "lookup_column_id": foreign.ID.String()})
		lk.OrderIndex = &order
		f.column.On("GetColumnByID", mock.Anything, relSchema, lk.ID.String()).Return(lk, nil)
		f.column.On("GetColumnByID", mock.Anything, relSchema, foreign.ID.String()).Return(foreign, nil)
		f.column.On("GetColumnByModelID", mock.Anything, relSchema, modelID).Return([]tenant.Column{}, nil)
		f.column.On("DeleteColumn", mock.Anything, relSchema, lk.ID.String()).Return(nil)
		f.rel.On("GetRelationByID", mock.Anything, relationID, relSchema).Return(tenant.Relation{SourceLookupColumns: []string{"name"}}, nil)
		f.rel.On("UpdateRelation", mock.Anything, relationID, mock.Anything, relSchema).Return(tenant.Relation{}, nil)
		f.table.On("GetByFunction", mock.Anything, "public.reorder_columns_after_delete", mock.Anything).Return([]map[string]interface{}{}, nil)

		err := f.svc.DeleteColumn(context.Background(), relSchema, lk.ID.String())

		assert.NoError(t, err)
		f.table.AssertCalled(t, "GetByFunction", mock.Anything, "public.reorder_columns_after_delete", mock.Anything)
	})

	t.Run("still deleted when the looked-up column is gone", func(t *testing.T) {
		f := newRelFixture()
		modelID, missing := uuid.NewString(), uuid.NewString()
		lk := lookupCol(uuid.New(), modelID, "lk", map[string]interface{}{"relation_id": uuid.NewString(), "lookup_column_id": missing})
		f.column.On("GetColumnByID", mock.Anything, relSchema, lk.ID.String()).Return(lk, nil)
		f.column.On("GetColumnByID", mock.Anything, relSchema, missing).Return(tenant.Column{}, app_errors.ColumnNotFound)
		f.column.On("DeleteColumn", mock.Anything, relSchema, lk.ID.String()).Return(nil)

		err := f.svc.DeleteColumn(context.Background(), relSchema, lk.ID.String())

		assert.NoError(t, err)
		f.rel.AssertNotCalled(t, "UpdateRelation", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		f.column.AssertCalled(t, "DeleteColumn", mock.Anything, relSchema, lk.ID.String())
	})
}

func TestDeleteTable_SkipsColumnsRemovedWithTheirLink(t *testing.T) {
	// A legacy two-column self-link: deleting the first column also deletes the second.
	f := newRelFixture()
	modelID := uuid.New()
	relationID := uuid.NewString()
	src := linkCol(uuid.New(), modelID.String(), relationID, "source", modelID.String(), "has-many", "reports", "INT[]")
	tgt := linkCol(uuid.New(), modelID.String(), relationID, "target", modelID.String(), "has-many", "manager", "INT")

	f.model.On("GetModelByID", mock.Anything, relSchema, modelID.String()).Return(tenant.Model{ID: modelID, Alias: "emp"}, nil)
	f.column.On("GetColumnByModelID", mock.Anything, relSchema, modelID.String()).Return([]tenant.Column{src, tgt}, nil)
	f.column.On("GetColumnByID", mock.Anything, relSchema, src.ID.String()).Return(src, nil)
	f.column.On("GetColumnByID", mock.Anything, relSchema, tgt.ID.String()).Return(tgt, nil).Once()
	f.column.On("GetColumnByID", mock.Anything, relSchema, tgt.ID.String()).Return(tenant.Column{}, app_errors.ColumnNotFound)
	f.column.On("DeleteColumn", mock.Anything, relSchema, mock.Anything).Return(nil)
	f.rel.On("GetRelationByID", mock.Anything, relationID, relSchema).Return(tenant.Relation{SourceColumnID: src.ID.String(), TargetColumnID: tgt.ID.String(), SourceModelID: modelID.String(), TargetModelID: modelID.String()}, nil)
	f.table.On("AlterTable", mock.Anything, mock.Anything).Return(nil)
	f.view.On("GetViewsByModelID", mock.Anything, relSchema, modelID.String()).Return([]tenant.View{}, nil)
	f.model.On("DeleteModel", mock.Anything, relSchema, modelID.String()).Return(nil)
	f.table.On("DropTable", mock.Anything, mock.Anything).Return(nil)

	err := f.svc.DeleteTable(context.Background(), relSchema, modelID.String())

	assert.NoError(t, err)
	f.column.AssertNumberOfCalls(t, "DeleteColumn", 2)
	f.table.AssertCalled(t, "DropTable", mock.Anything, mock.Anything)
}

// linkFunctionsSucceed is a StubTableService.GetByFunctionFn that answers the link SQL functions
// with success ("ok" for link_rows, an empty result for the void functions).
func linkFunctionsSucceed(_ context.Context, functionName string, _ map[string]interface{}) ([]map[string]interface{}, error) {
	name := strings.TrimPrefix(functionName, "public.")
	if name == "link_rows" {
		return []map[string]interface{}{{name: "ok"}}, nil
	}
	return []map[string]interface{}{{name: nil}}, nil
}
