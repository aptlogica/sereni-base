package table_test

import (
	"context"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Fixture: a link from table "src" to table "tgt"
// ---------------------------------------------------------------------------

type displayEnv struct {
	f        *relFixture
	src, tgt uuid.UUID
	// text and other columns of each side
	srcName, tgtName, tgtNotes, tgtAge tenant.Column
	sourceReq, targetReq               dto.ColumnInsertion
}

func orderedCol(c tenant.Column, index float64) tenant.Column {
	c.OrderIndex = &index
	return c
}

func newDisplayEnv() *displayEnv {
	e := &displayEnv{f: newRelFixture(), src: uuid.New(), tgt: uuid.New()}
	e.srcName = orderedCol(plainCol(uuid.New(), e.src.String(), "name", "text"), 1)
	e.tgtName = orderedCol(plainCol(uuid.New(), e.tgt.String(), "name", "text"), 2)
	e.tgtNotes = orderedCol(plainCol(uuid.New(), e.tgt.String(), "notes", "longText"), 5)
	e.tgtAge = orderedCol(plainCol(uuid.New(), e.tgt.String(), "age", "number"), 1)

	f := e.f
	srcLink := linkCol(uuid.New(), e.src.String(), "", "source", e.tgt.String(), "has-many", "rel", "INT[]")
	tgtLink := linkCol(uuid.New(), e.tgt.String(), "", "target", e.src.String(), "has-many", "back", "INT")
	f.column.On("Create", mock.Anything, mock.MatchedBy(func(r dto.ColumnInsertion) bool { return r.ModelID == e.src }), relSchema).
		Run(func(a mock.Arguments) { e.sourceReq = a.Get(1).(dto.ColumnInsertion) }).Return(srcLink, nil).Maybe()
	f.column.On("Create", mock.Anything, mock.MatchedBy(func(r dto.ColumnInsertion) bool { return r.ModelID == e.tgt }), relSchema).
		Run(func(a mock.Arguments) { e.targetReq = a.Get(1).(dto.ColumnInsertion) }).Return(tgtLink, nil).Maybe()
	f.column.On("GetMaxOrderIndexOfColumn", mock.Anything, relSchema, e.tgt.String()).Return(float64(4), nil).Maybe()
	f.model.On("GetModelByID", mock.Anything, relSchema, e.src.String()).Return(tenant.Model{ID: e.src, Alias: "src", Title: "Src"}, nil)
	f.model.On("GetModelByID", mock.Anything, relSchema, e.tgt.String()).Return(tenant.Model{ID: e.tgt, Alias: "tgt", Title: "Tgt"}, nil)
	f.table.On("AddColumn", mock.Anything, mock.Anything).Return(nil).Maybe()
	f.rel.On("Create", mock.Anything, mock.Anything, relSchema).Return(tenant.Relation{}, nil).Maybe()
	e.columns(e.src, e.srcName)
	e.columns(e.tgt, e.tgtAge, e.tgtNotes, e.tgtName)
	return e
}

// columns answers the column list and the column-by-id lookups of a table
func (e *displayEnv) columns(model uuid.UUID, cols ...tenant.Column) {
	e.f.column.On("GetColumnByModelID", mock.Anything, relSchema, model.String()).Return(cols, nil).Maybe()
	for _, c := range cols {
		e.f.column.On("GetColumnByID", mock.Anything, relSchema, c.ID.String()).Return(c, nil).Maybe()
	}
}

func (e *displayEnv) addLink(meta map[string]interface{}) error {
	_, err := e.f.svc.AddColumn(context.Background(), relSchema, dto.AddColumnRequest{ModelID: e.src, BaseID: uuid.New(), Title: "Rel", UIDT: "links", Meta: meta})
	return err
}

func linkMeta(with uuid.UUID, display, inverse string) map[string]interface{} {
	relation := map[string]interface{}{"with": with.String(), "type": "has-many"}
	if inverse != "" {
		relation["inverse_display_column_id"] = inverse
	}
	meta := map[string]interface{}{"relation": relation}
	if display != "" {
		meta["display_column_id"] = display
	}
	return meta
}

// ---------------------------------------------------------------------------
// Creating a link
// ---------------------------------------------------------------------------

func TestAddLinkColumn_DisplayColumns(t *testing.T) {
	t.Run("each side keeps the column it was given", func(t *testing.T) {
		e := newDisplayEnv()

		err := e.addLink(linkMeta(e.tgt, e.tgtNotes.ID.String(), e.srcName.ID.String()))

		require.NoError(t, err)
		assert.Equal(t, e.tgtNotes.ID.String(), e.sourceReq.Meta["display_column_id"], "the source link shows a column of the linked table")
		assert.Equal(t, e.srcName.ID.String(), e.targetReq.Meta["display_column_id"], "the partner link shows a column of the source table")
		assert.NotContains(t, e.sourceReq.Meta["relation"].(map[string]interface{}), "inverse_display_column_id", "the partner's choice is not stored on the source")
	})

	t.Run("without a choice and without an Id column each side shows its first text column", func(t *testing.T) {
		e := newDisplayEnv()

		err := e.addLink(linkMeta(e.tgt, "", ""))

		require.NoError(t, err)
		// "name" has order index 2 and "notes" 5; "age" is first but a number
		assert.Equal(t, e.tgtName.ID.String(), e.sourceReq.Meta["display_column_id"])
		assert.Equal(t, e.srcName.ID.String(), e.targetReq.Meta["display_column_id"])
	})

	t.Run("without a choice each side shows its system Id column", func(t *testing.T) {
		e := newDisplayEnv()
		srcID := tenant.Column{ID: uuid.New(), ModelID: e.src.String(), ColumnName: "id", UIDT: "number", System: true}
		tgtID := tenant.Column{ID: uuid.New(), ModelID: e.tgt.String(), ColumnName: "id", UIDT: "number", System: true}
		e.f.column.ExpectedCalls = nil
		e.f.column.On("Create", mock.Anything, mock.Anything, relSchema).
			Run(func(a mock.Arguments) {
				req := a.Get(1).(dto.ColumnInsertion)
				if req.ModelID == e.src {
					e.sourceReq = req
				} else {
					e.targetReq = req
				}
			}).Return(linkCol(uuid.New(), e.src.String(), "", "source", e.tgt.String(), "has-many", "rel", "INT[]"), nil)
		e.f.column.On("GetMaxOrderIndexOfColumn", mock.Anything, relSchema, e.tgt.String()).Return(float64(4), nil)
		e.columns(e.src, srcID, e.srcName)
		e.columns(e.tgt, tgtID, e.tgtAge, e.tgtName)

		err := e.addLink(linkMeta(e.tgt, "", ""))

		require.NoError(t, err)
		assert.Equal(t, tgtID.ID.String(), e.sourceReq.Meta["display_column_id"])
		assert.Equal(t, srcID.ID.String(), e.targetReq.Meta["display_column_id"])
	})

	t.Run("the Id column can be chosen explicitly", func(t *testing.T) {
		e := newDisplayEnv()
		tgtID := tenant.Column{ID: uuid.New(), ModelID: e.tgt.String(), ColumnName: "id", UIDT: "number", System: true}
		e.f.column.On("GetColumnByID", mock.Anything, relSchema, tgtID.ID.String()).Return(tgtID, nil)

		require.NoError(t, e.addLink(linkMeta(e.tgt, tgtID.ID.String(), "")))

		assert.Equal(t, tgtID.ID.String(), e.sourceReq.Meta["display_column_id"])
	})

	t.Run("a table without a text column leaves it unset", func(t *testing.T) {
		e := newDisplayEnv()
		e.f.column.ExpectedCalls = nil
		e.f.column.On("Create", mock.Anything, mock.Anything, relSchema).
			Run(func(a mock.Arguments) {
				req := a.Get(1).(dto.ColumnInsertion)
				if req.ModelID == e.src {
					e.sourceReq = req
				} else {
					e.targetReq = req
				}
			}).Return(linkCol(uuid.New(), e.src.String(), "", "source", e.tgt.String(), "has-many", "rel", "INT[]"), nil)
		e.f.column.On("GetMaxOrderIndexOfColumn", mock.Anything, relSchema, e.tgt.String()).Return(float64(4), nil)
		e.columns(e.src)
		e.columns(e.tgt, e.tgtAge)

		err := e.addLink(linkMeta(e.tgt, "", ""))

		require.NoError(t, err)
		assert.NotContains(t, e.sourceReq.Meta, "display_column_id")
		assert.NotContains(t, e.targetReq.Meta, "display_column_id")
	})

	t.Run("a self-link has one column and one display column", func(t *testing.T) {
		e := newDisplayEnv()
		meta := linkMeta(e.src, e.srcName.ID.String(), e.tgtName.ID.String())

		err := e.addLink(meta)

		require.NoError(t, err)
		assert.Equal(t, e.srcName.ID.String(), e.sourceReq.Meta["display_column_id"])
		e.f.column.AssertNumberOfCalls(t, "Create", 1)
	})

	t.Run("rejects a column that is not worth showing, before anything is created", func(t *testing.T) {
		cases := map[string]func(e *displayEnv) tenant.Column{
			"a column of the source table": func(e *displayEnv) tenant.Column { return e.srcName },
			"an attachment": func(e *displayEnv) tenant.Column {
				return plainCol(uuid.New(), e.tgt.String(), "files", "attachment")
			},
			"a json field": func(e *displayEnv) tenant.Column {
				return plainCol(uuid.New(), e.tgt.String(), "data", "json")
			},
			"a system column": func(e *displayEnv) tenant.Column {
				return tenant.Column{ID: uuid.New(), ModelID: e.tgt.String(), ColumnName: "created_time", UIDT: "datetime", System: true}
			},
			"a deleted column": func(e *displayEnv) tenant.Column {
				c := plainCol(uuid.New(), e.tgt.String(), "old", "text")
				c.Deleted = true
				return c
			},
		}
		for name, build := range cases {
			t.Run(name, func(t *testing.T) {
				e := newDisplayEnv()
				bad := build(e)
				e.f.column.On("GetColumnByID", mock.Anything, relSchema, bad.ID.String()).Return(bad, nil)

				err := e.addLink(linkMeta(e.tgt, bad.ID.String(), ""))

				assert.ErrorIs(t, err, app_errors.InvalidDisplayColumn)
				e.f.column.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
				e.f.rel.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
			})
		}
	})

	t.Run("rejects an unknown column and a bad partner column", func(t *testing.T) {
		e := newDisplayEnv()
		missing := uuid.NewString()
		e.f.column.On("GetColumnByID", mock.Anything, relSchema, missing).Return(tenant.Column{}, app_errors.ColumnNotFound)

		assert.ErrorIs(t, e.addLink(linkMeta(e.tgt, missing, "")), app_errors.InvalidDisplayColumn)
		// the partner's column must belong to the source table
		assert.ErrorIs(t, e.addLink(linkMeta(e.tgt, "", e.tgtName.ID.String())), app_errors.InvalidDisplayColumn)
		e.f.column.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
	})
}

// ---------------------------------------------------------------------------
// Editing a link
// ---------------------------------------------------------------------------

func TestUpdateLinkColumn_DisplayColumn(t *testing.T) {
	setup := func() (*displayEnv, tenant.Column, *dto.ColumnUpdate) {
		e := newDisplayEnv()
		link := linkCol(uuid.New(), e.src.String(), uuid.NewString(), "source", e.tgt.String(), "has-many", "rel", "INT[]")
		link.Meta["display_column_id"] = e.tgtName.ID.String()
		e.f.column.On("GetColumnByID", mock.Anything, relSchema, link.ID.String()).Return(link, nil)
		var sent dto.ColumnUpdate
		e.f.column.On("UpdateColumn", mock.Anything, relSchema, link.ID.String(), mock.Anything).
			Run(func(a mock.Arguments) { sent = a.Get(3).(dto.ColumnUpdate) }).Return(link, nil)
		return e, link, &sent
	}
	update := func(e *displayEnv, link tenant.Column, req dto.ColumnUpdate) error {
		_, err := e.f.svc.UpdateColumn(context.Background(), relSchema, link.ID.String(), req)
		return err
	}

	t.Run("changes the display column and keeps the rest of meta", func(t *testing.T) {
		e, link, sent := setup()
		meta := map[string]interface{}{"display_column_id": e.tgtNotes.ID.String()}

		require.NoError(t, update(e, link, dto.ColumnUpdate{Meta: &meta}))

		require.NotNil(t, sent.Meta)
		assert.Equal(t, e.tgtNotes.ID.String(), (*sent.Meta)["display_column_id"])
		assert.Equal(t, link.Meta["relation_id"], (*sent.Meta)["relation_id"])
		assert.Equal(t, "source", (*sent.Meta)["entity_role"])
		assert.Equal(t, link.Meta["relation"], (*sent.Meta)["relation"])
		// the stored meta is not changed in place
		assert.Equal(t, e.tgtName.ID.String(), link.Meta["display_column_id"])
	})

	t.Run("title and description can change together with it", func(t *testing.T) {
		e, link, sent := setup()
		meta := map[string]interface{}{"display_column_id": e.tgtNotes.ID.String()}

		require.NoError(t, update(e, link, dto.ColumnUpdate{Title: helpers.StringPtr("Renamed"), Description: helpers.StringPtr("About"), Meta: &meta}))

		assert.Equal(t, "Renamed", *sent.Title)
		assert.Equal(t, "About", *sent.Description)
		assert.Equal(t, e.tgtNotes.ID.String(), (*sent.Meta)["display_column_id"])
	})

	t.Run("an empty id clears it", func(t *testing.T) {
		e, link, sent := setup()
		meta := map[string]interface{}{"display_column_id": ""}

		require.NoError(t, update(e, link, dto.ColumnUpdate{Meta: &meta}))

		require.NotNil(t, sent.Meta)
		assert.NotContains(t, *sent.Meta, "display_column_id")
		assert.Contains(t, *sent.Meta, "relation_id")
	})

	t.Run("a title edit alone leaves meta untouched", func(t *testing.T) {
		e, link, sent := setup()

		require.NoError(t, update(e, link, dto.ColumnUpdate{Title: helpers.StringPtr("Renamed")}))

		assert.Nil(t, sent.Meta)
	})

	t.Run("the relation and the field type stay locked", func(t *testing.T) {
		e, link, sent := setup()
		meta := map[string]interface{}{
			"relation":          map[string]interface{}{"with": uuid.NewString(), "type": "one-to-one"},
			"relation_id":       uuid.NewString(),
			"entity_role":       "target",
			"display_column_id": e.tgtNotes.ID.String(),
		}

		require.NoError(t, update(e, link, dto.ColumnUpdate{UIDT: helpers.StringPtr("text"), Meta: &meta}))

		assert.Nil(t, sent.UIDT)
		assert.Equal(t, link.Meta["relation"], (*sent.Meta)["relation"])
		assert.Equal(t, link.Meta["relation_id"], (*sent.Meta)["relation_id"])
		assert.Equal(t, "source", (*sent.Meta)["entity_role"])
	})

	t.Run("rejects a column that can't label records", func(t *testing.T) {
		e, link, _ := setup()
		attachment := plainCol(uuid.New(), e.tgt.String(), "files", "attachment")
		other := e.srcName // a column of the source table, not of the linked one
		e.f.column.On("GetColumnByID", mock.Anything, relSchema, attachment.ID.String()).Return(attachment, nil)

		for _, id := range []string{attachment.ID.String(), other.ID.String(), uuid.NewString()} {
			if id != attachment.ID.String() && id != other.ID.String() {
				e.f.column.On("GetColumnByID", mock.Anything, relSchema, id).Return(tenant.Column{}, app_errors.ColumnNotFound)
			}
			meta := map[string]interface{}{"display_column_id": id}

			err := update(e, link, dto.ColumnUpdate{Meta: &meta})

			assert.ErrorIs(t, err, app_errors.InvalidDisplayColumn, id)
		}
		e.f.column.AssertNotCalled(t, "UpdateColumn", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("only this side changes", func(t *testing.T) {
		e, link, _ := setup()
		meta := map[string]interface{}{"display_column_id": e.tgtNotes.ID.String()}

		require.NoError(t, update(e, link, dto.ColumnUpdate{Meta: &meta}))

		e.f.column.AssertNumberOfCalls(t, "UpdateColumn", 1)
		e.f.rel.AssertNotCalled(t, "GetRelationByID", mock.Anything, mock.Anything, mock.Anything)
	})
}

// ---------------------------------------------------------------------------
// Deleting a column and changing its type
// ---------------------------------------------------------------------------

func TestDeleteColumn_ClearsDisplayColumnOfLinks(t *testing.T) {
	e := newDisplayEnv()
	f := e.f
	order := 2.0
	shown := e.tgtName
	shown.OrderIndex = &order

	usesIt := linkCol(uuid.New(), e.src.String(), uuid.NewString(), "source", e.tgt.String(), "has-many", "rel", "INT[]")
	usesIt.Meta["display_column_id"] = shown.ID.String()
	partnerUsesIt := linkCol(uuid.New(), e.tgt.String(), uuid.NewString(), "target", e.src.String(), "has-many", "back", "INT")
	partnerUsesIt.Meta["display_column_id"] = shown.ID.String()
	showsOther := linkCol(uuid.New(), e.src.String(), uuid.NewString(), "source", e.tgt.String(), "has-many", "rel2", "INT[]")
	showsOther.Meta["display_column_id"] = e.tgtNotes.ID.String()
	showsNothing := linkCol(uuid.New(), e.src.String(), uuid.NewString(), "source", e.tgt.String(), "has-many", "rel3", "INT[]")

	f.column.ExpectedCalls = nil
	f.column.On("GetColumnByID", mock.Anything, relSchema, shown.ID.String()).Return(shown, nil)
	f.column.On("GetAllColumns", mock.Anything, relSchema).Return([]tenant.Column{usesIt, partnerUsesIt, showsOther, showsNothing, shown}, nil)
	f.column.On("GetColumnByModelID", mock.Anything, relSchema, e.tgt.String()).Return([]tenant.Column{}, nil)
	f.column.On("DeleteColumn", mock.Anything, relSchema, shown.ID.String()).Return(nil)
	var cleared []string
	f.column.On("UpdateColumn", mock.Anything, relSchema, mock.Anything, mock.Anything).Run(func(a mock.Arguments) {
		req := a.Get(3).(dto.ColumnUpdate)
		assert.NotContains(t, *req.Meta, "display_column_id")
		assert.Contains(t, *req.Meta, "relation_id", "the rest of the link's meta is kept")
		cleared = append(cleared, a.String(2))
	}).Return(tenant.Column{}, nil)
	f.table.On("AlterTable", mock.Anything, mock.Anything).Return(nil)
	f.table.On("GetByFunction", mock.Anything, "public.reorder_columns_after_delete", mock.Anything).Return([]map[string]interface{}{}, nil)
	f.table.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil)

	err := f.svc.DeleteColumn(context.Background(), relSchema, shown.ID.String())

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{usesIt.ID.String(), partnerUsesIt.ID.String()}, cleared,
		"links that showed the column, on either table, are cleared; the others are left alone")
}

func TestUpdateColumn_TypeOfDisplayColumn(t *testing.T) {
	build := func(shownByLink bool) (*displayEnv, tenant.Column) {
		e := newDisplayEnv()
		f := e.f
		f.column.ExpectedCalls = nil
		shown := e.tgtName
		link := linkCol(uuid.New(), e.src.String(), uuid.NewString(), "source", e.tgt.String(), "has-many", "rel", "INT[]")
		if shownByLink {
			link.Meta["display_column_id"] = shown.ID.String()
		}
		f.column.On("GetColumnByID", mock.Anything, relSchema, shown.ID.String()).Return(shown, nil)
		f.column.On("GetAllColumns", mock.Anything, relSchema).Return([]tenant.Column{link, shown}, nil)
		f.column.On("UpdateColumn", mock.Anything, relSchema, shown.ID.String(), mock.Anything).Return(shown, nil).Maybe()
		f.model.On("GetModelByID", mock.Anything, relSchema, e.tgt.String()).Return(tenant.Model{ID: e.tgt, Alias: "tgt"}, nil).Maybe()
		f.table.On("AlterTable", mock.Anything, mock.Anything).Return(nil).Maybe()
		f.table.On("GetByFunction", mock.Anything, "public.convert_column_type", mock.Anything).Return([]map[string]interface{}{{}}, nil).Maybe()
		f.table.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil).Maybe()
		return e, shown
	}

	t.Run("can't become a type that has no readable value while a link shows it", func(t *testing.T) {
		for _, uidt := range []string{"attachment", "json"} {
			e, shown := build(true)

			_, err := e.f.svc.UpdateColumn(context.Background(), relSchema, shown.ID.String(), dto.ColumnUpdate{UIDT: helpers.StringPtr(uidt)})

			assert.ErrorIs(t, err, app_errors.DisplayColumnInUse, uidt)
			e.f.column.AssertNotCalled(t, "UpdateColumn", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		}
	})

	t.Run("can become another readable type", func(t *testing.T) {
		e, shown := build(true)

		_, err := e.f.svc.UpdateColumn(context.Background(), relSchema, shown.ID.String(), dto.ColumnUpdate{UIDT: helpers.StringPtr("email")})

		assert.NotErrorIs(t, err, app_errors.DisplayColumnInUse)
	})

	t.Run("a column no link shows can change freely", func(t *testing.T) {
		e, shown := build(false)

		_, err := e.f.svc.UpdateColumn(context.Background(), relSchema, shown.ID.String(), dto.ColumnUpdate{UIDT: helpers.StringPtr("attachment")})

		assert.NotErrorIs(t, err, app_errors.DisplayColumnInUse)
	})
}
