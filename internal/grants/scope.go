package grants

import "slices"

// ScopeAttributes describes which target attributes a privilege's scope allows.
type ScopeAttributes struct {
	Database     bool
	Table        bool
	Column       bool
	AccessObject bool
}

var attributesByScope = map[string]ScopeAttributes{
	"GLOBAL":       {},
	"DATABASE":     {Database: true},
	"TABLE":        {Database: true, Table: true},
	"VIEW":         {Database: true, Table: true},
	"DICTIONARY":   {Database: true, Table: true},
	"COLUMN":       {Database: true, Table: true, Column: true},
	"USER_NAME":    {AccessObject: true},
	"DEFINER":      {AccessObject: true},
	"SOURCE":       {AccessObject: true},
	"TABLE_ENGINE": {},
}

// ScopeFor returns the ClickHouse scope family for a privilege.
func ScopeFor(privilege string) string {
	return Parsed().Scopes[privilege]
}

// ScopeAttributesFor returns the attributes supported by the privilege's own
// scope, the union over all its descendants, and whether the privilege (or any
// descendant) has a supported scope at all.
func ScopeAttributesFor(privilege string) (ScopeAttributes, ScopeAttributes, bool) {
	cat := Parsed()
	attrs := attributesByScope[cat.Scopes[privilege]]

	allAttrs := ScopeAttributes{}
	supported := false
	for _, p := range AllDescendants(cat.Groups, privilege) {
		a, ok := attributesByScope[cat.Scopes[p]]
		if !ok {
			continue
		}
		supported = true
		allAttrs.Database = allAttrs.Database || a.Database
		allAttrs.Table = allAttrs.Table || a.Table
		allAttrs.Column = allAttrs.Column || a.Column
		allAttrs.AccessObject = allAttrs.AccessObject || a.AccessObject
	}

	return attrs, allAttrs, supported
}

// SubsetOf reports whether every attribute set in s is also set in o.
func (s ScopeAttributes) SubsetOf(o ScopeAttributes) bool {
	return (!s.Database || o.Database) &&
		(!s.Table || o.Table) &&
		(!s.Column || o.Column) &&
		(!s.AccessObject || o.AccessObject)
}

// FoldedMembers returns the members of privilege actually granted at the requested
// (non-global) scope, and whether the restriction silently drops members of a
// different scope family (object hierarchy vs access object), as when a group such
// as ALL or ACCESS MANAGEMENT is restricted to a database or an access object.
func FoldedMembers(privilege string, requested ScopeAttributes) ([]string, bool) {
	if requested == (ScopeAttributes{}) {
		return nil, false
	}
	cat := Parsed()
	var granted []string
	var hasObject, hasAccessObject bool
	for _, p := range AllDescendants(cat.Groups, privilege) {
		a, ok := attributesByScope[cat.Scopes[p]]
		if !ok {
			continue
		}
		if a.Database || a.Table || a.Column {
			hasObject = true
		}
		if a.AccessObject {
			hasAccessObject = true
		}
		if requested.SubsetOf(a) {
			granted = append(granted, p)
		}
	}
	slices.Sort(granted)
	return granted, hasObject && hasAccessObject
}
