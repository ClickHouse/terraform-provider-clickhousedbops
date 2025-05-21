package querybuilder

import (
	"testing"
)

func Test_backtick(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "No backtick",
			s:    "test",
			want: "`test`",
		},
		{
			name: "One backtick",
			s:    "te`st",
			want: "`te\\`st`",
		},
		{
			name: "Multiple backticks",
			s:    "t`e`st",
			want: "`t\\`e\\`st`",
		},
		{
			name: "SQL injection attempt",
			s:    "te\\`st",
			want: "`te\\\\\\`st`",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := backtick(tt.s); got != tt.want {
				t.Errorf("backtick() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_quote(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "No quote",
			s:    "test",
			want: "'test'",
		},
		{
			name: "One quote",
			s:    "te'st",
			want: "'te\\'st'",
		},
		{
			name: "Multiple quotes",
			s:    "t'e'st",
			want: "'t\\'e\\'st'",
		},
		{
			name: "SQL injection attempt",
			s:    "te\\'st",
			want: "'te\\\\\\'st'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quote(tt.s); got != tt.want {
				t.Errorf("quote() = %v, want %v", got, tt.want)
			}
		})
	}
}
