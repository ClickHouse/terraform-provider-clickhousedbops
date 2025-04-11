package querybuilder

import (
	"testing"
)

func Test_grantQueryBuilder_Build(t *testing.T) {
	tests := []struct {
		name        string
		roleName    string
		to          string
		adminOption bool
		want        string
		wantErr     bool
	}{
		{
			name:     "Simple grant role",
			roleName: "test",
			to:       "user",
			want:     "GRANT `test` TO `user`;",
			wantErr:  false,
		},
		{
			name:     "Grant role with funky name",
			roleName: "te`st",
			to:       "user",
			want:     "GRANT `te\\`st` TO `user`;",
			wantErr:  false,
		},
		{
			name:        "Grant role with admin option",
			roleName:    "test",
			to:          "user",
			adminOption: true,
			want:        "GRANT `test` TO `user` WITH ADMIN OPTION;",
			wantErr:     false,
		},
		{
			name:     "Empty role name",
			roleName: "",
			to:       "user",
			want:     "",
			wantErr:  true,
		},
		{
			name:     "Empty grantee",
			roleName: "test",
			to:       "",
			want:     "",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &grantQueryBuilder{
				roleName:    tt.roleName,
				to:          tt.to,
				adminOption: tt.adminOption,
			}
			got, err := q.Build()
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Build() got = %v, want %v", got, tt.want)
			}
		})
	}
}
