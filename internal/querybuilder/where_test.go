package querybuilder

import (
	"testing"
)

type optionMock struct {
	Value string
}

func OptionMock(str string) Option {
	return &optionMock{
		Value: str,
	}
}

func (w *optionMock) String() string {
	return w.Value
}

func Test_Where_String(t *testing.T) {
	tests := []struct {
		name  string
		where Option
		want  string
	}{
		{
			name:  "String",
			where: Where("name", "mark"),
			want:  "WHERE `name` = 'mark'",
		},
		{
			name:  "Numeric",
			where: Where("age", 3),
			want:  "WHERE `age` = 3",
		},
		{
			name:  "String with backtick in name",
			where: Where("te`st", "value"),
			want:  "WHERE `te\\`st` = 'value'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.where.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
