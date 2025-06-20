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
			where: WhereEquals("name", "mark"),
			want:  "`name` = 'mark'",
		},
		{
			name:  "Numeric",
			where: WhereEquals("age", 3),
			want:  "`age` = 3",
		},
		{
			name:  "String with backtick in name",
			where: WhereEquals("te`st", "value"),
			want:  "`te\\`st` = 'value'",
		},
		{
			name:  "String Differs",
			where: WhereDiffers("name", "mark"),
			want:  "`name` <> 'mark'",
		},
		{
			name:  "Numeric Differs",
			where: WhereDiffers("age", 3),
			want:  "`age` <> 3",
		},
		{
			name:  "String with backtick in name Differs",
			where: WhereDiffers("te`st", "value"),
			want:  "`te\\`st` <> 'value'",
		},
		{
			name:  "Null",
			where: IsNull("age"),
			want:  "`age` IS NULL",
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
