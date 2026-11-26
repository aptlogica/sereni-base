package platform

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMetas() []*columnMeta {
	return []*columnMeta{
		{ID: "11111111-1111-1111-1111-111111111111", Title: "Email", ColumnName: "email_1", UIDT: "email", DT: "TEXT"},
		{ID: "22222222-2222-2222-2222-222222222222", Title: "Age", ColumnName: "age_2", UIDT: "number", DT: "INTEGER"},
		{ID: "33333333-3333-3333-3333-333333333333", Title: "Odd Name", ColumnName: "Odd-Name", UIDT: "text", DT: "TEXT"},
		{ID: "44444444-4444-4444-4444-444444444444", Title: "Orders", ColumnName: "", UIDT: "links"},
	}
}

func jsonWhere(t *testing.T, s string) map[string]interface{} {
	t.Helper()
	dec := json.NewDecoder(stringsReader(s))
	dec.UseNumber()
	var m map[string]interface{}
	require.NoError(t, dec.Decode(&m))
	return m
}

func TestCompileConditionGolden(t *testing.T) {
	cases := []struct{ name, where, sql string }{
		{"contains", `{"all":[{"column":"Email","op":"contains","value":"@"}]}`, `(email_1 LIKE '%@%')`},
		{"range", `{"all":[{"column":"Age","op":"gte","value":18},{"column":"Age","op":"lte","value":100}]}`, `(age_2 >= 18 AND age_2 <= 100)`},
		{"any and none", `{"any":[{"column":"Age","op":"is_null"},{"none":[{"column":"Age","op":"lt","value":0}]}]}`, `(age_2 IS NULL OR NOT (age_2 < 0))`},
		{"in", `{"all":[{"column":"Age","op":"in","value":[1,2,3]}]}`, `(age_2 IN (1, 2, 3))`},
		{"quote escaped", `{"all":[{"column":"Email","op":"eq","value":"o'brien"}]}`, `(email_1 = 'o''brien')`},
		{"like wildcards escaped", `{"all":[{"column":"Email","op":"starts_with","value":"50%_"}]}`, `(email_1 LIKE '50\%\_%')`},
		{"quoted identifier", `{"all":[{"column":"Odd Name","op":"not_empty"}]}`, `((` + `"Odd-Name"` + ` IS NOT NULL AND btrim("Odd-Name") <> ''))`},
		{"by id", `{"all":[{"column":"fld_22222222222222222222222222222222","op":"eq","value":true}]}`, `(age_2 = TRUE)`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sql, _, err := compileCondition(jsonWhere(t, tc.where), testMetas())
			require.NoError(t, err)
			assert.Equal(t, tc.sql, sql)
		})
	}
}

func TestCompileConditionUsedColumns(t *testing.T) {
	_, used, err := compileCondition(jsonWhere(t, `{"all":[{"column":"Age","op":"gt","value":1},{"column":"Age","op":"lt","value":9},{"column":"Email","op":"is_null"}]}`), testMetas())
	require.NoError(t, err)
	require.Len(t, used, 2)
	assert.Equal(t, "Age", used[0].Title)
}

func TestCompileConditionRejects(t *testing.T) {
	bad := map[string]string{
		"unknown column":   `{"all":[{"column":"Nope","op":"eq","value":1}]}`,
		"unknown op":       `{"all":[{"column":"Age","op":"drop","value":1}]}`,
		"link column":      `{"all":[{"column":"Orders","op":"is_null"}]}`,
		"text op on int":   `{"all":[{"column":"Age","op":"contains","value":"1"}]}`,
		"empty group":      `{"all":[]}`,
		"two group keys":   `{"all":[{"column":"Age","op":"is_null"}],"any":[{"column":"Age","op":"is_null"}]}`,
		"value with null":  `{"all":[{"column":"Age","op":"is_null","value":1}]}`,
		"object as value":  `{"all":[{"column":"Age","op":"eq","value":{"a":1}}]}`,
		"unknown key":      `{"all":[{"column":"Age","op":"eq","value":1,"sql":"1=1"}]}`,
		"missing value":    `{"all":[{"column":"Age","op":"eq"}]}`,
		"sql in value ok?": ``,
	}
	delete(bad, "sql in value ok?")
	for name, where := range bad {
		t.Run(name, func(t *testing.T) {
			_, _, err := compileCondition(jsonWhere(t, where), testMetas())
			require.Error(t, err)
			assert.Equal(t, CodeValidationFailed, asAPIError(t, err).Code)
		})
	}
}

func TestCompileConditionLimits(t *testing.T) {
	deep := `{"all":[{"column":"Age","op":"is_null"}]}`
	for i := 0; i < maxConditionDepth; i++ {
		deep = `{"all":[` + deep + `]}`
	}
	_, _, err := compileCondition(jsonWhere(t, deep), testMetas())
	require.Error(t, err)
}

func TestSettingsParsing(t *testing.T) {
	assert.False(t, parseBaseSettings(nil).enabled(featureBaseManagement))
	assert.False(t, parseBaseSettings([]byte(`not json`)).enabled(featureBaseManagement))
	assert.False(t, parseBaseSettings([]byte(`{}`)).enabled(featureBaseManagement))
	s := parseBaseSettings([]byte(`{"base_management":{"enabled":true,"enabled_by":"u1"},"future":{"x":1}}`))
	assert.True(t, s.enabled(featureBaseManagement))
	assert.Equal(t, "u1", s.BaseManagement.EnabledBy)
}

func TestUidtLookup(t *testing.T) {
	u, ok := appUIDT("longtext")
	assert.True(t, ok)
	assert.Equal(t, "longText", u)
	_, ok = appUIDT("nonsense")
	assert.False(t, ok)
}

func TestParseColumnInput(t *testing.T) {
	in, details, err := parseColumnInput(map[string]interface{}{"title": "Email", "uidt": "email",
		"constraints": map[string]interface{}{"unique": true, "bogus": 1}}, "body", true)
	require.NoError(t, err)
	assert.Equal(t, "email", in.UIDT)
	require.Len(t, details, 1)
	assert.Contains(t, details[0].Field, "bogus")

	_, details, _ = parseColumnInput(map[string]interface{}{}, "body", true)
	assert.Len(t, details, 2) // title and uidt required

	for _, uidt := range []string{"formula", "attachment", "user"} {
		_, _, err = parseColumnInput(map[string]interface{}{"title": "X", "uidt": uidt}, "body", true)
		assert.Equal(t, CodeTypeNotSupported, asAPIError(t, err).Code, uidt)
	}
	_, _, err = parseColumnInput(map[string]interface{}{"title": "X", "uidt": "text", "system": true}, "body", true)
	assert.Equal(t, CodeBadRequest, asAPIError(t, err).Code)
}

func TestSchemaLocks(t *testing.T) {
	var l schemaLocks
	release, ok := l.acquire(t.Context(), "b", 50_000_000)
	require.True(t, ok)
	_, ok = l.acquire(t.Context(), "b", 20_000_000)
	assert.False(t, ok, "second writer on the same base waits and gives up")
	_, ok = l.acquire(t.Context(), "other", 20_000_000)
	assert.True(t, ok, "other bases are independent")
	release()
	_, ok = l.acquire(t.Context(), "b", 20_000_000)
	assert.True(t, ok)
}

func TestErrorCodeStatuses(t *testing.T) {
	assert.Equal(t, 409, CodeInUse.Status())
	assert.Equal(t, 403, CodeFeatureDisabled.Status())
	assert.Equal(t, 403, CodeSystemColumn.Status())
	assert.Equal(t, 422, CodeTypeNotSupported.Status())
	assert.Equal(t, 409, CodeConversionNotAllowed.Status())
}

func stringsReader(s string) *strings.Reader { return strings.NewReader(s) }
