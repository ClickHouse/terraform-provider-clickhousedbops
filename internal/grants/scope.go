package grants

// ScopeAttributes describes which target attributes a privilege's scope allows.
type ScopeAttributes struct {
	Database bool
	Table    bool
	Column   bool
}

var attributesByScope = map[string]ScopeAttributes{
	"GLOBAL":       {},
	"DATABASE":     {Database: true},
	"TABLE":        {Database: true, Table: true},
	"VIEW":         {Database: true, Table: true},
	"DICTIONARY":   {Database: true, Table: true},
	"COLUMN":       {Database: true, Table: true, Column: true},
	"USER_NAME":    {},
	"DEFINER":      {},
	"SOURCE":       {},
	"TABLE_ENGINE": {},
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
	}

	return attrs, allAttrs, supported
}
