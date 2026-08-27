package clickhouseclient

import (
	"context"
	"reflect"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/pingcap/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConn implements driver.Conn for the methods used by nativeClient.Select.
// The embedded interface panics on any unexpected call.
type fakeConn struct {
	driver.Conn
	rows driver.Rows
	err  error
}

func (f *fakeConn) Query(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
	return f.rows, f.err
}

// fakeColumnType reports a string column so Select scans into *string.
type fakeColumnType struct {
	driver.ColumnType
	name string
}

func (f *fakeColumnType) Name() string           { return f.name }
func (f *fakeColumnType) ScanType() reflect.Type { return reflect.TypeOf("") }

// fakeRows simulates a clickhouse-go result stream. After `rows` are consumed,
// Next returns false and Err returns `err`, mimicking a mid-stream failure
// (connection reset, query aborted server-side) that clickhouse-go surfaces
// only through Err() after Next() returns false.
type fakeRows struct {
	driver.Rows
	columns []string
	rows    [][]string
	err     error

	pos    int
	closed bool
}

func (f *fakeRows) Next() bool {
	if f.pos < len(f.rows) {
		f.pos++
		return true
	}
	return false
}

func (f *fakeRows) Scan(dest ...any) error {
	for i, d := range dest {
		*(d.(*string)) = f.rows[f.pos-1][i]
	}
	return nil
}

func (f *fakeRows) ColumnTypes() []driver.ColumnType {
	out := make([]driver.ColumnType, 0, len(f.columns))
	for _, c := range f.columns {
		out = append(out, &fakeColumnType{name: c})
	}
	return out
}

func (f *fakeRows) Columns() []string { return f.columns }

func (f *fakeRows) Err() error {
	if f.pos >= len(f.rows) {
		return f.err
	}
	return nil
}

func (f *fakeRows) Close() error {
	f.closed = true
	return nil
}

func TestSelectReturnsErrorOnMidStreamFailure(t *testing.T) {
	// A result stream that dies before delivering all rows must surface an
	// error: reporting the partial (or empty) result as complete makes callers
	// treat an existing resource as absent, triggering conflicting re-creates
	// or silently skipped deletes.
	rows := &fakeRows{
		columns: []string{"name"},
		rows:    [][]string{{"row-before-failure"}},
		err:     errors.New("connection reset by peer"),
	}
	client := &nativeClient{connection: &fakeConn{rows: rows}}

	var seen []string
	err := client.Select(context.Background(), "SELECT name FROM system.users", func(r Row) error {
		name, err := r.GetString("name")
		require.NoError(t, err)
		seen = append(seen, name)
		return nil
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "connection reset by peer")
	assert.Equal(t, []string{"row-before-failure"}, seen, "rows delivered before the failure are processed")
	assert.True(t, rows.closed, "rows must be closed")
}

func TestSelectEmptyStreamWithErrorReturnsError(t *testing.T) {
	// The incident shape: the stream fails before the first row, so Next()
	// returns false immediately and the error is only visible via Err().
	// Select must not report a clean empty result.
	rows := &fakeRows{
		columns: []string{"name"},
		err:     errors.New("connection reset by peer"),
	}
	client := &nativeClient{connection: &fakeConn{rows: rows}}

	called := false
	err := client.Select(context.Background(), "SELECT name FROM system.users", func(Row) error {
		called = true
		return nil
	})

	require.Error(t, err)
	assert.ErrorContains(t, err, "connection reset by peer")
	assert.False(t, called, "callback must not run for a failed stream")
}

func TestSelectZeroRowsIsNotAnError(t *testing.T) {
	// A legitimately empty result set still means "not found".
	rows := &fakeRows{columns: []string{"name"}}
	client := &nativeClient{connection: &fakeConn{rows: rows}}

	called := false
	err := client.Select(context.Background(), "SELECT name FROM system.users", func(Row) error {
		called = true
		return nil
	})

	require.NoError(t, err)
	assert.False(t, called)
	assert.True(t, rows.closed, "rows must be closed")
}
