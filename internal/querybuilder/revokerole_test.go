package querybuilder

import (
	"testing"
)

func Test_revokeRoleQueryBuilder_Build(t *testing.T) {
	tests := []struct {
		name     string
		roleName string
		from     string
		want     string
		wantErr  bool
	}{
		{
			name:     "Simple revoke role",
			roleName: "test",
			from:     "user",
			want:     "REVOKE `test` FROM `user`;",
			wantErr:  false,
		},
		{
			name:     "REVOKE role with funky name",
			roleName: "te`st",
			from:     "user",
			want:     "REVOKE `te\\`st` FROM `user`;",
			wantErr:  false,
		},
		{
			name:     "Empty role name",
			roleName: "",
			from:     "user",
			want:     "",
			wantErr:  true,
		},
		{
			name:     "Empty from",
			roleName: "test",
			from:     "",
			want:     "",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &revokeRoleQueryBuilder{
				roleName: tt.roleName,
				from:     tt.from,
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
