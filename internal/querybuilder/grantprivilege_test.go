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
			builder: GrantPrivilege("SELECT", "user1").WithDatabase(new("db1")),
			want:    "GRANT SELECT ON `db1`.* TO `user1`;",
			wantErr: false,
		},
		{
			name:    "Select on wildcard database",
			builder: GrantPrivilege("SELECT", "user1").WithDatabase(new("prefix_*")),
			want:    "GRANT SELECT ON prefix_*.* TO `user1`;",
			wantErr: false,
		},
		{
			name:    "Select on table",
			builder: GrantPrivilege("SELECT", "user1").WithDatabase(new("db1")).WithTable(new("tbl1")),
			want:    "GRANT SELECT ON `db1`.`tbl1` TO `user1`;",
			wantErr: false,
		},
		{
			name:    "Select on wildcard table",
			builder: GrantPrivilege("SELECT", "user1").WithDatabase(new("db1")).WithTable(new("tbl_*")),
			want:    "GRANT SELECT ON `db1`.tbl_* TO `user1`;",
			wantErr: false,
		},
		{
			name:    "Select on single column",
			builder: GrantPrivilege("SELECT", "user1").WithDatabase(new("db1")).WithTable(new("tbl1")).WithColumn(new("test")),
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
			builder: GrantPrivilege("SELECT", "role1").WithCurrentGrants(true).WithDatabase(new("db1")).WithGrantOption(true),
			want:    "GRANT CURRENT GRANTS(SELECT ON `db1`.*) TO `role1` WITH GRANT OPTION;",
			wantErr: false,
		},
		{
			name:    "Current grants on single column",
			builder: GrantPrivilege("SELECT", "user1").WithCurrentGrants(true).WithDatabase(new("db1")).WithTable(new("tbl1")).WithColumn(new("test")),
			want:    "GRANT CURRENT GRANTS(SELECT(`test`) ON `db1`.`tbl1`) TO `user1`;",
			wantErr: false,
		},
		{
			name:    "Current grants false is unchanged",
			builder: GrantPrivilege("SELECT", "user1").WithCurrentGrants(false),
			want:    "GRANT SELECT ON *.* TO `user1`;",
			wantErr: false,
		},
		{
			name:    "Access management CREATE USER on all",
			builder: GrantPrivilege("CREATE USER", "admin"),
			want:    "GRANT CREATE USER ON *.* TO `admin`;",
			wantErr: false,
		},
		{
			name:    "Access management ALTER ROLE with grant option",
			builder: GrantPrivilege("ALTER ROLE", "admin").WithGrantOption(true),
			want:    "GRANT ALTER ROLE ON *.* TO `admin` WITH GRANT OPTION;",
			wantErr: false,
		},
		{
			name:    "Access object on named user",
			builder: GrantPrivilege("CREATE USER", "admin").WithAccessObject(new("bob")),
			want:    "GRANT CREATE USER ON `bob` TO `admin`;",
			wantErr: false,
		},
		{
			name:    "Access object prefix pattern",
			builder: GrantPrivilege("CREATE USER", "admin").WithAccessObject(new("team_*")),
			want:    "GRANT CREATE USER ON team_* TO `admin`;",
			wantErr: false,
		},
		{
			name:    "Table engine target",
			builder: GrantPrivilege("TABLE ENGINE", "builder").WithAccessObject(new("Distributed")).WithParameterizedTarget(true),
			want:    "GRANT TABLE ENGINE ON `Distributed` TO `builder`;",
			wantErr: false,
		},
		{
			name:    "All table engines",
			builder: GrantPrivilege("TABLE ENGINE", "builder").WithParameterizedTarget(true),
			want:    "GRANT TABLE ENGINE ON * TO `builder`;",
			wantErr: false,
		},
		{
			name:    "Named source target",
			builder: GrantPrivilege("READ", "reader").WithAccessObject(new("S3")).WithParameterizedTarget(true),
			want:    "GRANT READ ON `S3` TO `reader`;",
			wantErr: false,
		},
		{
			name: "Filtered source target",
			builder: GrantPrivilege("READ", "reader").
				WithAccessObject(new("URL")).
				WithAccessObjectFilter(new(`https://example\.com/files/.*`)).
				WithParameterizedTarget(true),
			want:    "GRANT READ ON `URL`('https://example\\\\.com/files/.*') TO `reader`;",
			wantErr: false,
		},
		{
			name: "Filtered source target escapes SQL string",
			builder: GrantPrivilege("READ", "reader").
				WithAccessObject(new("URL")).
				WithAccessObjectFilter(new(`https://example.com/o'hare\\.*`)).
				WithParameterizedTarget(true),
			want:    "GRANT READ ON `URL`('https://example.com/o\\'hare\\\\\\\\.*') TO `reader`;",
			wantErr: false,
		},
		{
			name: "Filter requires an access object",
			builder: GrantPrivilege("READ", "reader").
				WithAccessObjectFilter(new(".*")).
				WithParameterizedTarget(true),
			wantErr: true,
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
