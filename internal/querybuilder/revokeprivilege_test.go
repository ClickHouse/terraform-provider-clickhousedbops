package querybuilder

import (
	"testing"
)

func Test_revokePrivilegeQueryBuilder(t *testing.T) {
	tests := []struct {
		name    string
		builder RevokePrivilegeQueryBuilder
		want    string
		wantErr bool
	}{
		{
			name:    "Select on all",
			builder: RevokePrivilege("SELECT", "user1"),
			want:    "REVOKE SELECT ON *.* FROM `user1`;",
			wantErr: false,
		},
		{
			name:    "Select on database",
			builder: RevokePrivilege("SELECT", "user1").WithDatabase(strptr("db1")),
			want:    "REVOKE SELECT ON `db1`.* FROM `user1`;",
			wantErr: false,
		},
		{
			name:    "Select on table",
			builder: RevokePrivilege("SELECT", "user1").WithDatabase(strptr("db1")).WithTable(strptr("tbl1")),
			want:    "REVOKE SELECT ON `db1`.`tbl1` FROM `user1`;",
			wantErr: false,
		},
		{
			name:    "Select on single column",
			builder: RevokePrivilege("SELECT", "user1").WithDatabase(strptr("db1")).WithTable(strptr("tbl1")).WithColumn(strptr("test")),
			want:    "REVOKE SELECT(`test`) ON `db1`.`tbl1` FROM `user1`;",
			wantErr: false,
		},
		{
			name:    "Missing access type",
			builder: RevokePrivilege("", "user1"),
			want:    "",
			wantErr: true,
		},
		{
			name:    "Missing from",
			builder: RevokePrivilege("SELECT", ""),
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.builder.Build()
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

func strptr(str string) *string {
	return &str
}
