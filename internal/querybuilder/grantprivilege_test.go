package querybuilder

import (
	"testing"
)

func Test_grantPrivilegeQueryBuilder(t *testing.T) {
	tests := []struct {
		name    string
		builder GrantPrivilegeQueryBuilder
		want    string
		wantErr bool
	}{
		{
			name:    "Select on all",
			builder: GrantPrivilege("SELECT", "user1"),
			want:    "GRANT SELECT ON *.* TO `user1`;",
			wantErr: false,
		},
		{
			name:    "Select on database",
			builder: GrantPrivilege("SELECT", "user1").WithDatabase(strptr("db1")),
			want:    "GRANT SELECT ON `db1`.* TO `user1`;",
			wantErr: false,
		},
		{
			name:    "Select on table",
			builder: GrantPrivilege("SELECT", "user1").WithDatabase(strptr("db1")).WithTable(strptr("tbl1")),
			want:    "GRANT SELECT ON `db1`.`tbl1` TO `user1`;",
			wantErr: false,
		},
		{
			name:    "Select on single column",
			builder: GrantPrivilege("SELECT", "user1").WithDatabase(strptr("db1")).WithTable(strptr("tbl1")).WithColumn(strptr("test")),
			want:    "GRANT SELECT(`test`) ON `db1`.`tbl1` TO `user1`;",
			wantErr: false,
		},
		{
			name:    "Grant option",
			builder: GrantPrivilege("SELECT", "user1").WithGrantOption(true),
			want:    "GRANT SELECT ON *.* TO `user1` WITH GRANT OPTION;",
			wantErr: false,
		},
		{
			name:    "Missing access type",
			builder: GrantPrivilege("", "user1"),
			want:    "",
			wantErr: true,
		},
		{
			name:    "Missing to",
			builder: GrantPrivilege("SELECT", ""),
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
