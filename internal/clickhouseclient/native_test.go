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

// fakeConn implements the driver.Conn methods used by Select; the embedded interface panics on anything else.
type fakeConn struct {
	driver.Conn
	rows driver.Rows
}

func (f *fakeConn) Query(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
	return f.rows, nil
}

type fakeColumnType struct {
	driver.ColumnType
	name string
}

func (f *fakeColumnType) Name() string           { return f.name }
func (f *fakeColumnType) ScanType() reflect.Type { return reflect.TypeOf("") }

// fakeRows serves scripted rows, then Next() returns false with err surfaced only via Err(), like a mid-stream failure.
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
	assert.Equal(t, []string{"row-before-failure"}, seen)
	assert.True(t, rows.closed)
}

func TestSelectZeroRowsIsNotAnError(t *testing.T) {
	rows := &fakeRows{columns: []string{"name"}}
	client := &nativeClient{connection: &fakeConn{rows: rows}}

	called := false
	err := client.Select(context.Background(), "SELECT name FROM system.users", func(Row) error {
		called = true
		return nil
	})

	require.NoError(t, err)
	assert.False(t, called)
	assert.True(t, rows.closed)
}
