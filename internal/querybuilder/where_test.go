package querybuilder

import (
	"testing"
)

func Test_SimpleWhere_Clause(t *testing.T) {
	tests := []struct {
		name  string
		where Where
		want  string
	}{
		{
			name:  "String",
			where: SimpleWhere("name", "mark"),
			want:  "WHERE `name` = 'mark'",
		},
		{
			name:  "Numeric",
			where: SimpleWhere("age", 3),
			want:  "WHERE `age` = 3",
		},
		{
			name:  "String with backtick in name",
			where: SimpleWhere("te`st", "value"),
			want:  "WHERE `te\\`st` = 'value'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.where.Clause(); got != tt.want {
				t.Errorf("Clause() = %v, want %v", got, tt.want)
			}
		})
	}
}
