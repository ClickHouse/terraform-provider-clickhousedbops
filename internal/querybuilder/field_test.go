package querybuilder

import (
	"testing"
)

func Test_field_SQLDef(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		toString  bool
		want      string
	}{
		{
			name:      "Simple field",
			fieldName: "field1",
			toString:  false,
			want:      "`field1`",
		},
		{
			name:      "Field name with backtick",
			fieldName: "fie`ld1",
			toString:  false,
			want:      "`fie\\`ld1`",
		},
		{
			name:      "Simple field with toString",
			fieldName: "field1",
			toString:  true,
			want:      "toString(`field1`) AS `field1`",
		},
		{
			name:      "Field name with backtick and toString",
			fieldName: "fie`ld1",
			toString:  true,
			want:      "toString(`fie\\`ld1`) AS `fie\\`ld1`",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &field{
				name:     tt.fieldName,
				toString: tt.toString,
			}
			if got := f.SQLDef(); got != tt.want {
				t.Errorf("SQLDef() = %v, want %v", got, tt.want)
			}
		})
	}
}
