package querybuilder

import (
	"testing"
)

func Test_AndWhere_String(t *testing.T) {
	tests := []struct {
		name  string
		where Where
		want  string
	}{
		{
			name:  "2 clauses",
			where: AndWhere(WhereMock("clause1"), WhereMock("clause2")),
			want:  "(clause1 AND clause2)",
		},
		{
			name:  "1 clause",
			where: AndWhere(WhereMock("clause1")),
			want:  "(clause1)",
		},
		{
			name:  "3 clauses",
			where: AndWhere(WhereMock("clause1"), WhereMock("clause2"), WhereMock("clause3")),
			want:  "(clause1 AND clause2 AND clause3)",
		},
		{
			name:  "nested",
			where: AndWhere(AndWhere(WhereMock("clause1"), WhereMock("clause2")), WhereMock("clause3")),
			want:  "((clause1 AND clause2) AND clause3)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.where.Clause(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

type whereMock struct {
	Value string
}

func WhereMock(str string) Where {
	return &whereMock{
		Value: str,
	}
}

func (w whereMock) Clause() string {
	return w.Value
}
