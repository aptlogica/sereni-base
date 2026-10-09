package table_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// relationFakeDB is a tiny database/sql driver for the transactional link code paths.
// Each *sql.DB gets its own hooks (looked up by DSN), so tests don't share state.
//
// Default behaviour: every locked row exists, no row holds an ID, no cycle, every write succeeds.
type relationFakeDB struct {
	mu sync.Mutex

	missingRows map[int64]bool     // rows reported missing by lockRow
	holders     map[string][]int64 // column name -> rows returned by rowsHoldingID
	cycle       bool               // result of the recursive cycle queries
	execErr     error              // returned by every Exec when set
	beginErr    error

	queries   []string
	execs     []fakeExec
	commits   int
	rollbacks int
}

type fakeExec struct {
	Query string
	Args  []any
}

func (f *relationFakeDB) query(q string, args []any) ([]string, [][]driver.Value, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queries = append(f.queries, q)

	switch {
	case strings.Contains(q, "WITH RECURSIVE"):
		return []string{"exists"}, [][]driver.Value{{f.cycle}}, nil
	case strings.Contains(q, "WHERE id = $1 FOR UPDATE"):
		id, _ := args[0].(int64)
		if f.missingRows[id] {
			return []string{"id"}, nil, nil
		}
		return []string{"id"}, [][]driver.Value{{id}}, nil
	case strings.Contains(q, "FOR UPDATE"):
		var rows [][]driver.Value
		for column, ids := range f.holders {
			if strings.Contains(q, fmt.Sprintf(`"%s"`, column)) {
				for _, id := range ids {
					rows = append(rows, []driver.Value{id})
				}
			}
		}
		return []string{"id"}, rows, nil
	}
	return nil, nil, fmt.Errorf("unexpected query: %s", q)
}

func (f *relationFakeDB) exec(q string, args []any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.execs = append(f.execs, fakeExec{Query: q, Args: args})
	return f.execErr
}

// execsOn returns the UPDATE statements that touched the given column.
func (f *relationFakeDB) execsOn(column string) []fakeExec {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []fakeExec
	for _, e := range f.execs {
		if strings.Contains(e.Query, fmt.Sprintf(`"%s"`, column)) {
			out = append(out, e)
		}
	}
	return out
}

var relationFakeRegistry sync.Map // dsn -> *relationFakeDB

func newRelationFakeDB() (*sql.DB, *relationFakeDB) {
	dsn := uuid.NewString()
	state := &relationFakeDB{missingRows: map[int64]bool{}, holders: map[string][]int64{}}
	relationFakeRegistry.Store(dsn, state)
	db, err := sql.Open("relation_fake", dsn)
	if err != nil {
		panic(err)
	}
	return db, state
}

type relationFakeDriver struct{}

func (relationFakeDriver) Open(dsn string) (driver.Conn, error) {
	state, ok := relationFakeRegistry.Load(dsn)
	if !ok {
		return nil, errors.New("unknown fake dsn")
	}
	return &relationFakeConn{state: state.(*relationFakeDB)}, nil
}

type relationFakeConn struct{ state *relationFakeDB }

func (c *relationFakeConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *relationFakeConn) Close() error                        { return nil }
func (c *relationFakeConn) Begin() (driver.Tx, error) {
	if c.state.beginErr != nil {
		return nil, c.state.beginErr
	}
	return &relationFakeTx{state: c.state}, nil
}

func (c *relationFakeConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	cols, data, err := c.state.query(query, namedToAny(args))
	if err != nil {
		return nil, err
	}
	return &relationFakeRows{cols: cols, data: data, idx: -1}, nil
}

func (c *relationFakeConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if err := c.state.exec(query, namedToAny(args)); err != nil {
		return nil, err
	}
	return driver.RowsAffected(1), nil
}

type relationFakeTx struct{ state *relationFakeDB }

func (t *relationFakeTx) Commit() error {
	t.state.mu.Lock()
	defer t.state.mu.Unlock()
	t.state.commits++
	return nil
}

func (t *relationFakeTx) Rollback() error {
	t.state.mu.Lock()
	defer t.state.mu.Unlock()
	t.state.rollbacks++
	return nil
}

type relationFakeRows struct {
	cols []string
	data [][]driver.Value
	idx  int
}

func (r *relationFakeRows) Columns() []string { return r.cols }
func (r *relationFakeRows) Close() error      { return nil }
func (r *relationFakeRows) Next(dest []driver.Value) error {
	r.idx++
	if r.idx >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.idx])
	return nil
}

func namedToAny(args []driver.NamedValue) []any {
	out := make([]any, len(args))
	for i, a := range args {
		out[i] = a.Value
	}
	return out
}

func init() {
	sql.Register("relation_fake", relationFakeDriver{})
}
