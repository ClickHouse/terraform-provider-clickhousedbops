package grants

import "testing"

func TestCovers(t *testing.T) {
	tests := []struct {
		name     string
		broader  Grant
		narrower Grant
		want     bool
	}{
		// AccessType: identity and group coverage.
		{"same privilege", Grant{AccessType: "SELECT"}, Grant{AccessType: "SELECT"}, true},
		{"unrelated privileges", Grant{AccessType: "INSERT"}, Grant{AccessType: "SELECT"}, false},
		{"group covers a member", Grant{AccessType: "CREATE"}, Grant{AccessType: "CREATE TABLE"}, true},
		{"top-level group covers a deeper member", Grant{AccessType: "ALL"}, Grant{AccessType: "CREATE TABLE"}, true},
		{"leaf does not cover a group", Grant{AccessType: "CREATE TABLE"}, Grant{AccessType: "CREATE"}, false},

		// Database dimension.
		{"database: same", Grant{AccessType: "SELECT", Database: ptr("test")}, Grant{AccessType: "SELECT", Database: ptr("test")}, true},
		{"database: unrelated", Grant{AccessType: "SELECT", Database: ptr("prod")}, Grant{AccessType: "SELECT", Database: ptr("test")}, false},
		{"database: broader unrestricted covers specific", Grant{AccessType: "SELECT"}, Grant{AccessType: "SELECT", Database: ptr("test")}, true},
		{"database: specific does not cover all", Grant{AccessType: "SELECT", Database: ptr("test")}, Grant{AccessType: "SELECT"}, false},
		{"database: prefix wildcard covers", Grant{AccessType: "SELECT", Database: ptr("tes*")}, Grant{AccessType: "SELECT", Database: ptr("test*")}, true},
		{"database: implicit prefix wildcard covers", Grant{AccessType: "SELECT", Database: ptr("tes")}, Grant{AccessType: "SELECT", Database: ptr("test")}, true},

		// Table dimension.
		{"table: broader database covers a table in it", Grant{AccessType: "SELECT", Database: ptr("db")}, Grant{AccessType: "SELECT", Database: ptr("db"), Table: ptr("t")}, true},
		{"table: specific table does not cover the database", Grant{AccessType: "SELECT", Database: ptr("db"), Table: ptr("t")}, Grant{AccessType: "SELECT", Database: ptr("db")}, false},
		{"table: different table", Grant{AccessType: "SELECT", Database: ptr("db"), Table: ptr("t1")}, Grant{AccessType: "SELECT", Database: ptr("db"), Table: ptr("t2")}, false},

		// Column dimension.
		{"column: broader table covers a column in it", Grant{AccessType: "SELECT", Database: ptr("db"), Table: ptr("t")}, Grant{AccessType: "SELECT", Database: ptr("db"), Table: ptr("t"), Column: ptr("c")}, true},
		{"column: specific column does not cover all", Grant{AccessType: "SELECT", Database: ptr("db"), Table: ptr("t"), Column: ptr("c")}, Grant{AccessType: "SELECT", Database: ptr("db"), Table: ptr("t")}, false},
		{"column: different column", Grant{AccessType: "SELECT", Database: ptr("db"), Table: ptr("t"), Column: ptr("c1")}, Grant{AccessType: "SELECT", Database: ptr("db"), Table: ptr("t"), Column: ptr("c2")}, false},

		// Grant option.
		{"grant option needed but broader lacks it", Grant{AccessType: "SELECT"}, Grant{AccessType: "SELECT", GrantOption: true}, false},
		{"grant option needed and broader has it", Grant{AccessType: "SELECT", GrantOption: true}, Grant{AccessType: "SELECT", GrantOption: true}, true},
		{"grant option not needed", Grant{AccessType: "SELECT", GrantOption: true}, Grant{AccessType: "SELECT"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Covers(tt.broader, tt.narrower); got != tt.want {
				t.Errorf("Covers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func ptr(s string) *string { return &s }
