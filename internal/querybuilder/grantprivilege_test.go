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
			name:    "Select on wildcard database",
			builder: GrantPrivilege("SELECT", "user1").WithDatabase(strptr("prefix_*")),
			want:    "GRANT SELECT ON prefix_*.* TO `user1`;",
			wantErr: false,
		},
		{
			name:    "Select on table",
			builder: GrantPrivilege("SELECT", "user1").WithDatabase(strptr("db1")).WithTable(strptr("tbl1")),
			want:    "GRANT SELECT ON `db1`.`tbl1` TO `user1`;",
			wantErr: false,
		},
		{
			name:    "Select on wildcard table",
			builder: GrantPrivilege("SELECT", "user1").WithDatabase(strptr("db1")).WithTable(strptr("tbl_*")),
			want:    "GRANT SELECT ON `db1`.tbl_* TO `user1`;",
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
			name:    "Current grants ALL on all",
			builder: GrantPrivilege("ALL", "user1").WithCurrentGrants(true),
			want:    "GRANT CURRENT GRANTS(ALL ON *.*) TO `user1`;",
			wantErr: false,
		},
		{
			name:    "Current grants SELECT on all",
			builder: GrantPrivilege("SELECT", "user1").WithCurrentGrants(true),
			want:    "GRANT CURRENT GRANTS(SELECT ON *.*) TO `user1`;",
			wantErr: false,
		},
		{
			name:    "Current grants on database with grant option",
			builder: GrantPrivilege("SELECT", "role1").WithCurrentGrants(true).WithDatabase(strptr("db1")).WithGrantOption(true),
			want:    "GRANT CURRENT GRANTS(SELECT ON `db1`.*) TO `role1` WITH GRANT OPTION;",
			wantErr: false,
		},
		{
			name:    "Current grants false is unchanged",
			builder: GrantPrivilege("SELECT", "user1").WithCurrentGrants(false),
			want:    "GRANT SELECT ON *.* TO `user1`;",
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
