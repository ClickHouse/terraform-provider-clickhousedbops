package grants

import "slices"

// ScopeAttributes describes which target attributes a privilege's scope allows.
type ScopeAttributes struct {
	Database     bool
	Table        bool
	Column       bool
	AccessObject bool
	SourceFilter bool
}

var attributesByScope = map[string]ScopeAttributes{
	"GLOBAL":           {},
	"DATABASE":         {Database: true},
	"TABLE":            {Database: true, Table: true},
	"VIEW":             {Database: true, Table: true},
	"DICTIONARY":       {Database: true, Table: true},
	"COLUMN":           {Database: true, Table: true, Column: true},
	"USER_NAME":        {AccessObject: true},
	"DEFINER":          {AccessObject: true},
	"SOURCE":           {AccessObject: true, SourceFilter: true},
	"NAMED_COLLECTION": {AccessObject: true},
	"TABLE_ENGINE":     {AccessObject: true},
}

// ScopeAttributesFor returns the attributes supported by every scoped member of
// the privilege (itself and its descendants), the union over all of them, and
// whether any of them has a supported scope at all.
func ScopeAttributesFor(privilege string) (ScopeAttributes, ScopeAttributes, bool) {
	cat := Parsed()
	attrs := ScopeAttributes{Database: true, Table: true, Column: true, AccessObject: true, SourceFilter: true}

	allAttrs := ScopeAttributes{}
	supported := false
	for _, p := range AllDescendants(cat.Groups, privilege) {
		scope, scoped := cat.Scopes[p]
		if !scoped {
			continue
		}
		// An unsupported scope yields no attributes, so its members narrow every target.
		a, ok := attributesByScope[scope]
		attrs.Database = attrs.Database && a.Database
		attrs.Table = attrs.Table && a.Table
		attrs.Column = attrs.Column && a.Column
		attrs.AccessObject = attrs.AccessObject && a.AccessObject
		attrs.SourceFilter = attrs.SourceFilter && a.SourceFilter
		if !ok {
			continue
		}
		supported = true
		allAttrs.Database = allAttrs.Database || a.Database
		allAttrs.Table = allAttrs.Table || a.Table
		allAttrs.Column = allAttrs.Column || a.Column
		allAttrs.AccessObject = allAttrs.AccessObject || a.AccessObject
		allAttrs.SourceFilter = allAttrs.SourceFilter || a.SourceFilter
	}

	if !supported {
		attrs = ScopeAttributes{}
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
