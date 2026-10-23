package grants

import "testing"

func TestScopeAttributesFor(t *testing.T) {
	tests := []struct {
		privilege string
		want      ScopeAttributes
		wantAll   ScopeAttributes
		supported bool
	}{
		// Leaves: own scope equals the descendant union.
		{"SELECT", ScopeAttributes{Database: true, Table: true, Column: true}, ScopeAttributes{Database: true, Table: true, Column: true}, true},
		{"CREATE DATABASE", ScopeAttributes{Database: true}, ScopeAttributes{Database: true}, true},
		{"CREATE USER", ScopeAttributes{AccessObject: true}, ScopeAttributes{AccessObject: true}, true},
		{"CREATE", ScopeAttributes{}, ScopeAttributes{Database: true, Table: true}, true},
		{"ACCESS MANAGEMENT", ScopeAttributes{}, ScopeAttributes{Database: true, Table: true, AccessObject: true}, true},
		{"ALL", ScopeAttributes{}, ScopeAttributes{Database: true, Table: true, Column: true, AccessObject: true, SourceFilter: true}, true},
		{"ALTER VIEW", ScopeAttributes{Database: true, Table: true}, ScopeAttributes{Database: true, Table: true}, true},
		{"ALTER TABLE", ScopeAttributes{Database: true, Table: true}, ScopeAttributes{Database: true, Table: true, Column: true}, true},
		{"ALTER COLUMN", ScopeAttributes{Database: true, Table: true, Column: true}, ScopeAttributes{Database: true, Table: true, Column: true}, true},
		{"SHOW", ScopeAttributes{Database: true}, ScopeAttributes{Database: true, Table: true, Column: true}, true},
		{"ALTER DATABASE", ScopeAttributes{Database: true}, ScopeAttributes{Database: true}, true},
		{"ALTER", ScopeAttributes{}, ScopeAttributes{Database: true, Table: true, Column: true}, true},
		{"TABLE ENGINE", ScopeAttributes{AccessObject: true}, ScopeAttributes{AccessObject: true}, true},
		{"READ", ScopeAttributes{AccessObject: true, SourceFilter: true}, ScopeAttributes{AccessObject: true, SourceFilter: true}, true},
		{"NAMED COLLECTION", ScopeAttributes{AccessObject: true}, ScopeAttributes{AccessObject: true}, true},
		{"NAMED COLLECTION ADMIN", ScopeAttributes{AccessObject: true}, ScopeAttributes{AccessObject: true}, true},
		{"CREATE NAMED COLLECTION", ScopeAttributes{AccessObject: true}, ScopeAttributes{AccessObject: true}, true},
		{"UNKNOWN", ScopeAttributes{}, ScopeAttributes{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.privilege, func(t *testing.T) {
			got, gotAll, ok := ScopeAttributesFor(tt.privilege)
			if got != tt.want || gotAll != tt.wantAll || ok != tt.supported {
				t.Errorf("ScopeAttributesFor(%q) = %+v, %+v, %v; want %+v, %+v, %v", tt.privilege, got, gotAll, ok, tt.want, tt.wantAll, tt.supported)
			}
		})
	}
}
