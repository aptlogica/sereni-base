package middleware_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	ownModelID     = "model-own"
	victimModelID  = "model-victim"
	ownColumnID    = "col-own"
	victimColumnID = "col-victim"
	ownViewID      = "view-own"
	victimViewID   = "view-victim"
	formContent    = "application/x-www-form-urlencoded"
	jsonContent    = "application/json"
)

var (
	ownBaseTarget    = middleware.ScopeTarget{WorkspaceID: ownWorkspaceID, BaseID: ownBaseID}
	victimBaseTarget = middleware.ScopeTarget{WorkspaceID: victimWorkspace, BaseID: victimBaseID}
)

func lookupFrom(m map[string]string) func(context.Context, string, string) (string, error) {
	return func(_ context.Context, _, id string) (string, error) {
		if v, ok := m[id]; ok {
			return v, nil
		}
		return "", errors.New("not found")
	}
}

// fixtureLookups models two workspaces, each with one base, table, column and view, plus
// access rows of every scope.
func fixtureLookups() middleware.ScopeLookups {
	return middleware.ScopeLookups{
		BaseWorkspace: lookupFrom(map[string]string{ownBaseID: ownWorkspaceID, victimBaseID: victimWorkspace, "base-orphan": ""}),
		ModelBase:     lookupFrom(map[string]string{ownModelID: ownBaseID, victimModelID: victimBaseID}),
		ColumnModel:   lookupFrom(map[string]string{ownColumnID: ownModelID, victimColumnID: victimModelID, "col-orphan": ""}),
		ViewModel:     lookupFrom(map[string]string{ownViewID: ownModelID, victimViewID: victimModelID}),
		AccessMember: func(_ context.Context, _, id string) (dto.AccessMemberDTO, error) {
			switch id {
			case "am-ws":
				return dto.AccessMemberDTO{ScopeType: constant.ScopeLevels.Workspace, ScopeID: strPtr(victimWorkspace)}, nil
			case "am-base":
				return dto.AccessMemberDTO{ScopeType: constant.ScopeLevels.Base, ScopeID: strPtr(victimBaseID)}, nil
			case "am-base-orphan":
				return dto.AccessMemberDTO{ScopeType: constant.ScopeLevels.Base, ScopeID: strPtr("base-unknown")}, nil
			case "am-system":
				return dto.AccessMemberDTO{ScopeType: constant.ScopeLevels.System}, nil
			case "am-system-scoped":
				return dto.AccessMemberDTO{ScopeType: constant.ScopeLevels.System, ScopeID: strPtr("tenant")}, nil
			case "am-no-scope":
				return dto.AccessMemberDTO{ScopeType: constant.ScopeLevels.Workspace}, nil
			}
			return dto.AccessMemberDTO{}, errors.New("not found")
		},
	}
}

func testContext(method, contentType, body string, params gin.Params) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(method, "/", strings.NewReader(body))
	if contentType != "" {
		c.Request.Header.Set("Content-Type", contentType)
	}
	c.Params = params
	return c
}

func form(kv ...string) string {
	v := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Set(kv[i], kv[i+1])
	}
	return v.Encode()
}

func idParam(id string) gin.Params { return gin.Params{{Key: "id", Value: id}} }

// Each resolver maps a request to its scope targets, or rejects it when the request is malformed
// or references objects outside the target scope.
func TestScopeLookups_Resolvers(t *testing.T) {
	l := fixtureLookups()
	empty := middleware.ScopeLookups{}

	tests := []struct {
		name        string
		resolve     middleware.TargetResolver
		contentType string
		body        string
		params      gin.Params
		want        []middleware.ScopeTarget // nil means the resolver must fail
	}{
		// Path parameters
		{"workspace param", l.WorkspaceParam("id"), "", "", idParam(victimWorkspace), []middleware.ScopeTarget{{WorkspaceID: victimWorkspace}}},
		{"workspace param missing", l.WorkspaceParam("id"), "", "", nil, nil},
		{"base param", l.BaseParam("id"), "", "", idParam(victimBaseID), []middleware.ScopeTarget{victimBaseTarget}},
		{"base param unknown", l.BaseParam("id"), "", "", idParam("base-unknown"), nil},
		{"base param without workspace", l.BaseParam("id"), "", "", idParam("base-orphan"), nil},
		{"base param without lookup", empty.BaseParam("id"), "", "", idParam(victimBaseID), nil},
		{"model param", l.ModelParam("id"), "", "", idParam(victimModelID), []middleware.ScopeTarget{victimBaseTarget}},
		{"model param unknown", l.ModelParam("id"), "", "", idParam("model-unknown"), nil},
		{"model param without lookup", empty.ModelParam("id"), "", "", idParam(victimModelID), nil},
		{"column param", l.ColumnParam("id"), "", "", idParam(victimColumnID), []middleware.ScopeTarget{victimBaseTarget}},
		{"column param unknown", l.ColumnParam("id"), "", "", idParam("col-unknown"), nil},
		{"column param without table", l.ColumnParam("id"), "", "", idParam("col-orphan"), nil},
		{"column param without lookup", empty.ColumnParam("id"), "", "", idParam(victimColumnID), nil},
		{"view param", l.ViewParam("id"), "", "", idParam(victimViewID), []middleware.ScopeTarget{victimBaseTarget}},
		{"view param unknown", l.ViewParam("id"), "", "", idParam("view-unknown"), nil},
		{"view param without lookup", empty.ViewParam("id"), "", "", idParam(victimViewID), nil},

		// Form fields
		{"workspace form", l.WorkspaceForm("workspace_id"), formContent, form("workspace_id", victimWorkspace), nil, []middleware.ScopeTarget{{WorkspaceID: victimWorkspace}}},
		{"workspace form missing", l.WorkspaceForm("workspace_id"), formContent, "", nil, nil},
		{"model form", l.ModelForm(), formContent, form("model_id", ownModelID, "column_id", ownColumnID), nil, []middleware.ScopeTarget{ownBaseTarget}},
		{"model form without column", l.ModelForm(), formContent, form("model_id", ownModelID), nil, []middleware.ScopeTarget{ownBaseTarget}},
		{"model form column of another table", l.ModelForm(), formContent, form("model_id", ownModelID, "column_id", victimColumnID), nil, nil},
		{"model form missing model", l.ModelForm(), formContent, form("column_id", ownColumnID), nil, nil},
		{"import into base", l.TableImportForm(), formContent, form("base_id", ownBaseID, "workspace_id", ownWorkspaceID), nil, []middleware.ScopeTarget{ownBaseTarget}},
		{"import into base without workspace", l.TableImportForm(), formContent, form("base_id", ownBaseID), nil, []middleware.ScopeTarget{ownBaseTarget}},
		{"import into new base", l.TableImportForm(), formContent, form("workspace_id", victimWorkspace), nil, []middleware.ScopeTarget{{WorkspaceID: victimWorkspace}}},
		{"import base outside workspace", l.TableImportForm(), formContent, form("base_id", victimBaseID, "workspace_id", ownWorkspaceID), nil, nil},
		{"import without target", l.TableImportForm(), formContent, "", nil, nil},

		// JSON bodies
		{"model body", l.ModelBody(), jsonContent, `{"model_id":"model-own","column_id":"col-own","values":{"col-own":1}}`, nil, []middleware.ScopeTarget{ownBaseTarget}},
		{"model body matching base_id", l.ModelBody(), jsonContent, `{"model_id":"model-own","base_id":"base-own"}`, nil, []middleware.ScopeTarget{ownBaseTarget}},
		{"model body foreign base_id", l.ModelBody(), jsonContent, `{"model_id":"model-own","base_id":"base-victim"}`, nil, nil},
		{"model body foreign column", l.ModelBody(), jsonContent, `{"model_id":"model-own","column_id":"col-victim"}`, nil, nil},
		{"model body foreign values key", l.ModelBody(), jsonContent, `{"model_id":"model-own","values":{"col-own":1,"col-victim":2}}`, nil, nil},
		{"model body links column", l.ModelBody(), jsonContent, `{"model_id":"model-own","meta":{"relation":{"with":"model-victim"}}}`, nil, []middleware.ScopeTarget{ownBaseTarget, victimBaseTarget}},
		{"model body link to unknown table", l.ModelBody(), jsonContent, `{"model_id":"model-own","meta":{"relation":{"with":"model-unknown"}}}`, nil, nil},
		{"model body meta without relation", l.ModelBody(), jsonContent, `{"model_id":"model-own","meta":{"x":1}}`, nil, []middleware.ScopeTarget{ownBaseTarget}},
		{"model body missing model", l.ModelBody(), jsonContent, `{}`, nil, nil},
		{"model body invalid json", l.ModelBody(), jsonContent, `{`, nil, nil},
		{"column pair same table", l.ColumnPairBody("a", "b"), jsonContent, `{"a":"col-own","b":"col-own"}`, nil, []middleware.ScopeTarget{ownBaseTarget}},
		{"column pair across tables", l.ColumnPairBody("a", "b"), jsonContent, `{"a":"col-own","b":"col-victim"}`, nil, nil},
		{"column pair unknown first", l.ColumnPairBody("a", "b"), jsonContent, `{"a":"col-unknown","b":"col-own"}`, nil, nil},
		{"column pair invalid json", l.ColumnPairBody("a", "b"), jsonContent, `[`, nil, nil},
		{"table create", l.TableCreateBody(), jsonContent, `{"base_id":"base-own","workspace_id":"ws-own"}`, nil, []middleware.ScopeTarget{ownBaseTarget}},
		{"table create base outside workspace", l.TableCreateBody(), jsonContent, `{"base_id":"base-victim","workspace_id":"ws-own"}`, nil, nil},
		{"table create unknown base", l.TableCreateBody(), jsonContent, `{"base_id":"base-unknown"}`, nil, nil},
		{"table create invalid json", l.TableCreateBody(), jsonContent, `nope`, nil, nil},
		{"membership", l.MembershipBody(), jsonContent,
			`{"membership":[{"workspace_id":"ws-own","bases":[{"base_id":"base-own"}]},{"workspace_id":"ws-victim"}]}`, nil,
			[]middleware.ScopeTarget{{WorkspaceID: ownWorkspaceID}, {WorkspaceID: victimWorkspace}}},
		{"membership base outside workspace", l.MembershipBody(), jsonContent, `{"membership":[{"workspace_id":"ws-own","bases":[{"base_id":"base-victim"}]}]}`, nil, nil},
		{"membership missing workspace", l.MembershipBody(), jsonContent, `{"membership":[{"bases":[]}]}`, nil, nil},
		{"membership empty", l.MembershipBody(), jsonContent, `{"membership":[]}`, nil, nil},
		{"membership invalid json", l.MembershipBody(), jsonContent, `{"membership":`, nil, nil},

		// Access rows
		{"access row in workspace", l.AccessMemberParam("id", constant.ScopeLevels.Workspace), "", "", idParam("am-ws"), []middleware.ScopeTarget{{WorkspaceID: victimWorkspace}}},
		{"access row in base maps to its workspace", l.AccessMemberParam("id", constant.ScopeLevels.Base), "", "", idParam("am-base"), []middleware.ScopeTarget{{WorkspaceID: victimWorkspace}}},
		{"access row in unknown base", l.AccessMemberParam("id", constant.ScopeLevels.Base), "", "", idParam("am-base-orphan"), nil},
		{"access row of other scope", l.AccessMemberParam("id", constant.ScopeLevels.Base), "", "", idParam("am-ws"), nil},
		{"system access row", l.AccessMemberParam("id", constant.ScopeLevels.Workspace), "", "", idParam("am-system"), nil},
		{"access row without scope id", l.AccessMemberParam("id", constant.ScopeLevels.Workspace), "", "", idParam("am-no-scope"), nil},
		{"access row unknown", l.AccessMemberParam("id", constant.ScopeLevels.Workspace), "", "", idParam("am-unknown"), nil},
		{"access row of unsupported scope", l.AccessMemberParam("id", constant.ScopeLevels.System), "", "", idParam("am-system"), nil},
		{"system route never resolves", l.AccessMemberParam("id", constant.ScopeLevels.System), "", "", idParam("am-system-scoped"), nil},
		{"access row without lookup", empty.AccessMemberParam("id", constant.ScopeLevels.Workspace), "", "", idParam("am-ws"), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := testContext(http.MethodPost, tt.contentType, tt.body, tt.params)

			got, err := tt.resolve(c, guardSchema)

			if tt.want == nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// JSON resolvers read the body before the handler does; the handler must still see all of it.
func TestScopeLookups_JSONBodyIsRestoredForHandler(t *testing.T) {
	body := `{"model_id":"model-own","rows":[{"a":1}]}`
	c := testContext(http.MethodPost, jsonContent, body, nil)

	_, err := fixtureLookups().ModelBody()(c, guardSchema)
	assert.NoError(t, err)

	rest, _ := io.ReadAll(c.Request.Body)
	assert.Equal(t, body, string(rest))
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

// A body that cannot be read is rejected rather than treated as empty.
func TestScopeLookups_UnreadableBodyIsRejected(t *testing.T) {
	c := testContext(http.MethodPost, jsonContent, "", nil)
	c.Request.Body = io.NopCloser(failingReader{})

	_, err := fixtureLookups().ModelBody()(c, guardSchema)

	assert.Error(t, err)
}

func serveObjectGuard(t *testing.T, guard gin.HandlerFunc, setUser bool) (int, bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if setUser {
			c.Set("user_id", guardUserID)
			c.Set("schema", guardSchema)
		}
		c.Next()
	})
	reached := false
	r.GET("/", guard, func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	return w.Code, reached
}

func fixedTargets(targets ...middleware.ScopeTarget) middleware.TargetResolver {
	return func(*gin.Context, string) ([]middleware.ScopeTarget, error) { return targets, nil }
}

func grantOn(svc *MockAccessMemberService, scopeID string, granted bool) {
	svc.On("CheckUserPermission", mock.Anything, guardSchema, guardUserID, mock.Anything,
		mock.MatchedBy(func(s *string) bool { return s != nil && *s == scopeID }),
		constant.ResourceCodes.Table, constant.ActionCodes.Update).Return(granted, nil)
}

func TestObjectPermissionGuard(t *testing.T) {
	wsMember := dto.AccessMemberDTO{ScopeType: constant.ScopeLevels.Workspace, ScopeID: strPtr(ownWorkspaceID)}
	baseMember := dto.AccessMemberDTO{ScopeType: constant.ScopeLevels.Base, ScopeID: strPtr(ownBaseID), WorkspaceID: strPtr(ownWorkspaceID)}

	t.Run("no authenticated user is denied", func(t *testing.T) {
		guard := middleware.NewObjectPermissionGuard("", "", new(MockAccessMemberService), fixedTargets(ownBaseTarget))
		_, reached := serveObjectGuard(t, guard, false)
		assert.False(t, reached)
	})

	t.Run("resolver error is denied", func(t *testing.T) {
		failing := func(*gin.Context, string) ([]middleware.ScopeTarget, error) { return nil, errors.New("bad request") }
		guard := middleware.NewObjectPermissionGuard("", "", new(MockAccessMemberService), failing)
		_, reached := serveObjectGuard(t, guard, true)
		assert.False(t, reached)
	})

	t.Run("no targets is denied", func(t *testing.T) {
		guard := middleware.NewObjectPermissionGuard("", "", new(MockAccessMemberService), fixedTargets())
		_, reached := serveObjectGuard(t, guard, true)
		assert.False(t, reached)
	})

	t.Run("membership lookup error is denied", func(t *testing.T) {
		svc := membersMock(nil, errors.New("db down"))
		guard := middleware.NewObjectPermissionGuard("", "", svc, fixedTargets(ownBaseTarget))
		_, reached := serveObjectGuard(t, guard, true)
		assert.False(t, reached)
	})

	t.Run("membership-only check passes for a covering membership", func(t *testing.T) {
		svc := membersMock([]dto.AccessMemberDTO{baseMember}, nil)
		guard := middleware.NewObjectPermissionGuard("", "", svc, fixedTargets(ownBaseTarget))
		code, reached := serveObjectGuard(t, guard, true)
		assert.True(t, reached)
		assert.Equal(t, http.StatusOK, code)
		svc.AssertNotCalled(t, "CheckUserPermission")
	})

	t.Run("base member does not cover a workspace-level target", func(t *testing.T) {
		svc := membersMock([]dto.AccessMemberDTO{baseMember}, nil)
		guard := middleware.NewObjectPermissionGuard("", "", svc, fixedTargets(middleware.ScopeTarget{WorkspaceID: ownWorkspaceID}))
		_, reached := serveObjectGuard(t, guard, true)
		assert.False(t, reached)
	})

	t.Run("permission denied on the covering membership", func(t *testing.T) {
		svc := membersMock([]dto.AccessMemberDTO{wsMember}, nil)
		grantOn(svc, ownWorkspaceID, false)
		guard := middleware.NewObjectPermissionGuard(constant.ResourceCodes.Table, constant.ActionCodes.Update, svc, fixedTargets(ownBaseTarget))
		_, reached := serveObjectGuard(t, guard, true)
		assert.False(t, reached)
	})

	t.Run("permission check error is denied", func(t *testing.T) {
		svc := membersMock([]dto.AccessMemberDTO{wsMember}, nil)
		svc.On("CheckUserPermission", mock.Anything, guardSchema, guardUserID, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(false, errors.New("db down"))
		guard := middleware.NewObjectPermissionGuard(constant.ResourceCodes.Table, constant.ActionCodes.Update, svc, fixedTargets(ownBaseTarget))
		_, reached := serveObjectGuard(t, guard, true)
		assert.False(t, reached)
	})

	t.Run("every target must pass", func(t *testing.T) {
		svc := membersMock([]dto.AccessMemberDTO{wsMember}, nil)
		grantOn(svc, ownWorkspaceID, true)
		guard := middleware.NewObjectPermissionGuard(constant.ResourceCodes.Table, constant.ActionCodes.Update, svc, fixedTargets(ownBaseTarget, victimBaseTarget))
		_, reached := serveObjectGuard(t, guard, true)
		assert.False(t, reached)
	})

	t.Run("duplicate targets are checked once", func(t *testing.T) {
		svc := membersMock([]dto.AccessMemberDTO{wsMember}, nil)
		grantOn(svc, ownWorkspaceID, true)
		guard := middleware.NewObjectPermissionGuard(constant.ResourceCodes.Table, constant.ActionCodes.Update, svc, fixedTargets(ownBaseTarget, ownBaseTarget))
		_, reached := serveObjectGuard(t, guard, true)
		assert.True(t, reached)
		svc.AssertNumberOfCalls(t, "CheckUserPermission", 1)
	})
}

func TestModelAccessFilter(t *testing.T) {
	serveFilter := func(t *testing.T, svc *MockAccessMemberService, l middleware.ScopeLookups, setUser bool, modelIDs ...string) (int, []bool) {
		t.Helper()
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) {
			if setUser {
				c.Set("user_id", guardUserID)
				c.Set("schema", guardSchema)
			}
			c.Next()
		})
		var results []bool
		r.GET("/", middleware.NewModelAccessFilter(svc, l), func(c *gin.Context) {
			val, _ := c.Get(middleware.ModelAccessFilterKey)
			canAccess := val.(func(string) bool)
			for _, id := range modelIDs {
				results = append(results, canAccess(id))
			}
			c.Status(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		return w.Code, results
	}

	t.Run("no authenticated user is rejected", func(t *testing.T) {
		code, results := serveFilter(t, new(MockAccessMemberService), fixtureLookups(), false)
		assert.NotEqual(t, http.StatusOK, code)
		assert.Nil(t, results)
	})

	t.Run("keeps only tables in the caller's scope", func(t *testing.T) {
		svc := membersMock([]dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.Base, ScopeID: strPtr(ownBaseID)}}, nil)
		code, results := serveFilter(t, svc, fixtureLookups(), true, ownModelID, victimModelID, "model-unknown")
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, []bool{true, false, false}, results)
	})

	t.Run("membership lookup error hides everything", func(t *testing.T) {
		svc := membersMock(nil, errors.New("db down"))
		_, results := serveFilter(t, svc, fixtureLookups(), true, ownModelID)
		assert.Equal(t, []bool{false}, results)
	})

	t.Run("each table is resolved once per request", func(t *testing.T) {
		calls := 0
		l := fixtureLookups()
		resolve := l.ModelBase
		l.ModelBase = func(ctx context.Context, schema, id string) (string, error) {
			calls++
			return resolve(ctx, schema, id)
		}
		svc := membersMock([]dto.AccessMemberDTO{{ScopeType: constant.ScopeLevels.System}}, nil)
		_, results := serveFilter(t, svc, l, true, ownModelID, ownModelID, ownModelID)
		assert.Equal(t, []bool{true, true, true}, results)
		assert.Equal(t, 1, calls)
	})
}
