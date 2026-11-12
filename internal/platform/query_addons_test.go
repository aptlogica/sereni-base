package platform

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- search ----

func TestSearchSQL(t *testing.T) {
	// Only text and select fields are searched (not uuid, numbers, links...); % and _ are escaped
	sql, args := whereSQL(t, url.Values{"search": {"  50%_off  "}})
	assert.Equal(t, `((("title")::text ILIKE $1) OR (("status")::text ILIKE $2))`, sql)
	assert.Equal(t, []interface{}{`%50\%\_off%`, `%50\%\_off%`}, args)

	// Combined with filters using AND
	sql, _ = whereSQL(t, url.Values{"search": {"acme"}, "filter[Paid]": {"true"}})
	assert.Equal(t, `("paid" = $1) AND ((("title")::text ILIKE $2) OR (("status")::text ILIKE $3))`, sql)
}

func TestSearchEmptyAndLimits(t *testing.T) {
	for _, s := range []string{"", "   "} {
		opts, err := parseListOptions(url.Values{"search": {s}}, whereFields(), testLinks)
		require.NoError(t, err)
		assert.Empty(t, opts.filters, "an empty search filters nothing")
	}

	_, err := parseListOptions(url.Values{"search": {strings.Repeat("é", maxSearchLength+1)}}, whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most 200 characters")
	_, err = parseListOptions(url.Values{"search": {strings.Repeat("é", maxSearchLength)}}, whereFields(), testLinks)
	assert.NoError(t, err)

	numbersOnly := []*field{{ID: "00000000-0000-0000-0000-000000000003", Name: "Amount", ColumnName: "amount", Type: typeCurrency}}
	_, err = parseListOptions(url.Values{"search": {"x"}}, numbersOnly, testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no text fields to search")
}

func TestSearchIsPartOfTheCursorHash(t *testing.T) {
	hash := func(q url.Values) string {
		opts, err := parseListOptions(q, whereFields(), testLinks)
		require.NoError(t, err)
		return opts.hash
	}
	none, acme := hash(url.Values{}), hash(url.Values{"search": {"acme"}})
	assert.NotEqual(t, none, acme)
	assert.NotEqual(t, acme, hash(url.Values{"search": {"other"}}))
	assert.Equal(t, acme, hash(url.Values{"search": {" acme "}}), "spaces around the text don't matter")

	// A cursor of one search can't be used with another
	q := url.Values{"search": {"other"}, "cursor": {encodeCursor(50, acme)}}
	_, err := parseListOptions(q, whereFields(), testLinks)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "different filter or sort")
}

// ---- expand ----

func TestSplitExpandPath(t *testing.T) {
	fields := []*field{
		{ID: "00000000-0000-0000-0000-000000000001", Name: "Customer", Type: typeLinks},
		{ID: "00000000-0000-0000-0000-000000000002", Name: "v1.2", Type: typeLinks},
	}
	f, rest, err := splitExpandPath(fields, "Customer")
	require.NoError(t, err)
	assert.Equal(t, "Customer", f.Name)
	assert.Equal(t, "", rest)

	f, rest, err = splitExpandPath(fields, "Customer.Company.Owner")
	require.NoError(t, err)
	assert.Equal(t, "Customer", f.Name)
	assert.Equal(t, "Company.Owner", rest)

	// A name with a dot is taken whole first
	f, rest, err = splitExpandPath(fields, "v1.2")
	require.NoError(t, err)
	assert.Equal(t, "v1.2", f.Name)
	assert.Equal(t, "", rest)
	f, rest, err = splitExpandPath(fields, "v1.2.Company")
	require.NoError(t, err)
	assert.Equal(t, "v1.2", f.Name)
	assert.Equal(t, "Company", rest)

	// By ID
	f, _, err = splitExpandPath(fields, fields[0].PublicID()+".Company")
	require.NoError(t, err)
	assert.Equal(t, "Customer", f.Name)

	_, _, err = splitExpandPath(fields, "Custmer.Company")
	require.Error(t, err)
	assert.Contains(t, fieldLookupError(err, "expand", fields, "Custmer").Error(), `No field named "Custmer.Company".`)
}

func TestWithExpanded(t *testing.T) {
	name, customer, owner := &field{Name: "Name"}, &field{Name: "Customer"}, &field{Name: "Owner"}
	all := []*field{name, customer, owner}
	assert.Equal(t, []*field{name}, withExpanded([]*field{name}, all, nil))

	selected := []*field{name}
	got := withExpanded(selected, all, expandSet{owner: {}, customer: {}})
	assert.Equal(t, []*field{name, customer, owner}, got, "expanded fields are added in table order")
	assert.Len(t, selected, 1, "the select list itself is not changed")
	assert.Equal(t, all, withExpanded(all, all, expandSet{customer: {}}))
}

// ---- timeout ----

func TestTimeoutError(t *testing.T) {
	expired, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-expired.Done()

	dbErr := errors.New("pq: canceling statement due to user request")
	err := timeoutError(expired, dbErr)
	var apiErr *Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, CodeQueryTimeout, apiErr.Code)
	assert.Equal(t, 503, apiErr.Code.Status())
	assert.Contains(t, apiErr.Message, "took longer than 10s")

	// Errors the API already explains, errors with time left, and no error stay as they are
	bad := badRequest("expand: nope")
	assert.Same(t, bad, timeoutError(expired, bad))
	assert.Same(t, dbErr, timeoutError(context.Background(), dbErr))
	assert.NoError(t, timeoutError(expired, nil))
}
