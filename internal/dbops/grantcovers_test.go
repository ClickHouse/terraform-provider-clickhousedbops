package dbops

import "testing"

func ptr(s string) *string { return &s }

func TestGrantCovers(t *testing.T) {
	tests := []struct {
		name     string
		broader  GrantPrivilege
		narrower GrantPrivilege
		want     bool
	}{
		{
			name:     "database grant covers table grant in that database",
			broader:  GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("logs_production")},
			narrower: GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("logs_production"), TableName: ptr("logs_raw_v2")},
			want:     true,
		},
		{
			name:     "global grant covers a database grant",
			broader:  GrantPrivilege{AccessType: "SELECT"},
			narrower: GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("logs_production")},
			want:     true,
		},
		{
			name:     "same table grant covers itself",
			broader:  GrantPrivilege{AccessType: "INSERT", DatabaseName: ptr("db"), TableName: ptr("t")},
			narrower: GrantPrivilege{AccessType: "INSERT", DatabaseName: ptr("db"), TableName: ptr("t")},
			want:     true,
		},
		{
			name:     "different database does not cover",
			broader:  GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("other")},
			narrower: GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("logs_production"), TableName: ptr("logs_raw_v2")},
			want:     false,
		},
		{
			name:     "table grant does not cover the whole database",
			broader:  GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("db"), TableName: ptr("t")},
			narrower: GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("db")},
			want:     false,
		},
		{
			name:     "different access type does not cover",
			broader:  GrantPrivilege{AccessType: "INSERT", DatabaseName: ptr("db")},
			narrower: GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("db"), TableName: ptr("t")},
			want:     false,
		},
		{
			name:     "partial revoke never covers",
			broader:  GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("db"), IsPartialRevoke: true},
			narrower: GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("db"), TableName: ptr("t")},
			want:     false,
		},
		{
			name:     "covering grant without grant option does not cover a grant that needs it",
			broader:  GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("db"), GrantOption: false},
			narrower: GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("db"), TableName: ptr("t"), GrantOption: true},
			want:     false,
		},
		{
			name:     "covering grant with grant option covers a grant that needs it",
			broader:  GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("db"), GrantOption: true},
			narrower: GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("db"), TableName: ptr("t"), GrantOption: true},
			want:     true,
		},
		{
			name:     "column grant is covered by its table grant",
			broader:  GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("db"), TableName: ptr("t")},
			narrower: GrantPrivilege{AccessType: "SELECT", DatabaseName: ptr("db"), TableName: ptr("t"), ColumnName: ptr("c")},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := grantCovers(tt.broader, tt.narrower); got != tt.want {
				t.Errorf("grantCovers() = %v, want %v", got, tt.want)
			}
		})
	}
}
