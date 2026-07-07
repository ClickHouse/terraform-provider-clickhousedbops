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
		{"ALL", ScopeAttributes{}, ScopeAttributes{Database: true, Table: true, Column: true, AccessObject: true}, true},
		{"TABLE ENGINE", ScopeAttributes{}, ScopeAttributes{}, true},
		{"READ", ScopeAttributes{AccessObject: true}, ScopeAttributes{AccessObject: true}, true},
		{"CREATE NAMED COLLECTION", ScopeAttributes{}, ScopeAttributes{}, false},
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
