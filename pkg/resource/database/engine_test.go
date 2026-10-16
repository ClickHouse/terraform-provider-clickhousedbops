package database

import (
	"reflect"
	"testing"
)

func TestParseEngineFull(t *testing.T) {
	tests := []struct {
		name       string
		engineFull string
		want       engineClause
		wantOK     bool
	}{
		{
			name:       "No arguments",
			engineFull: "Atomic",
			want:       engineClause{name: "Atomic"},
			wantOK:     true,
		},
		{
			name:       "Empty argument list",
			engineFull: "Memory()",
			want:       engineClause{name: "Memory"},
			wantOK:     true,
		},
		{
			name:       "Replicated",
			engineFull: "Replicated('/clickhouse/databases/{database}', '{shard}', '{replica}')",
			want: engineClause{
				name:      "Replicated",
				arguments: []string{"'/clickhouse/databases/{database}'", "'{shard}'", "'{replica}'"},
			},
			wantOK: true,
		},
		{
			name:       "Arguments with nested expressions, quotes, and escapes",
			engineFull: `Engine(f('a,b', [1, 2]), 'it\'s, (here)', "ident,", ` + "`x)`" + `)`,
			want: engineClause{
				name:      "Engine",
				arguments: []string{"f('a,b', [1, 2])", `'it\'s, (here)'`, `"ident,"`, "`x)`"},
			},
			wantOK: true,
		},
		{
			name:       "Arguments and settings",
			engineFull: "DataLakeCatalog('https://catalog.example.test', '[HIDDEN]') SETTINGS catalog_type = 'rest', warehouse = 'a = b, c'",
			want: engineClause{
				name:      "DataLakeCatalog",
				arguments: []string{"'https://catalog.example.test'", "'[HIDDEN]'"},
				settings:  map[string]string{"catalog_type": "'rest'", "warehouse": "'a = b, c'"},
			},
			wantOK: true,
		},
		{
			name:       "Settings without arguments",
			engineFull: "Replicated SETTINGS logs_to_keep = 500",
			want: engineClause{
				name:     "Replicated",
				settings: map[string]string{"logs_to_keep": "500"},
			},
			wantOK: true,
		},
		{name: "Empty", engineFull: ""},
		{name: "Unbalanced arguments", engineFull: "Replicated('a', 'b'"},
		{name: "Unterminated quote", engineFull: "Replicated('a)"},
		{name: "Unknown clause", engineFull: "Replicated('a') COMMENT 'b'"},
		{name: "Trailing text after arguments", engineFull: "Replicated('a')x"},
		{name: "Malformed setting", engineFull: "Replicated SETTINGS logs_to_keep"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseEngineFull(tt.engineFull)
			if ok != tt.wantOK {
				t.Fatalf("parseEngineFull() ok = %v, want %v (clause %#v)", ok, tt.wantOK, got)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseEngineFull() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestEngineExpressionMatches(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		actual     string
		want       bool
	}{
		{name: "Identical", configured: "'{shard}'", actual: "'{shard}'", want: true},
		{name: "Surrounding whitespace", configured: " 'a' ", actual: "'a'", want: true},
		{name: "Different value", configured: "'a'", actual: "'b'"},
		{name: "Different formatting", configured: `"a"`, actual: "'a'"},
		{name: "Placeholder matches string literal", configured: "{secret:String}", actual: `'s3\'cret'`, want: true},
		{name: "Placeholder inside expression", configured: "f({secret:String}, 1)", actual: "f('value', 1)", want: true},
		{name: "Placeholder does not match number", configured: "{secret:String}", actual: "1"},
		{name: "Placeholder does not match more than a literal", configured: "{secret:String}", actual: "'a', 'b'"},
		{name: "Hidden value matches string literal", configured: "'password'", actual: "'[HIDDEN]'", want: true},
		{name: "Hidden value matches placeholder", configured: "{secret:String}", actual: "'[HIDDEN]'", want: true},
		{name: "Hidden value does not match number", configured: "1", actual: "'[HIDDEN]'"},
		{name: "Regexp metacharacters are literal", configured: "'a.c'", actual: "'abc'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := engineExpressionMatches(tt.configured, tt.actual); got != tt.want {
				t.Errorf("engineExpressionMatches(%q, %q) = %v, want %v", tt.configured, tt.actual, got, tt.want)
			}
		})
	}
}

func TestEngineClauseMatches(t *testing.T) {
	clause := engineClause{
		name:      "DataLakeCatalog",
		arguments: []string{"'https://catalog.example.test'"},
		settings:  map[string]string{"catalog_type": "'rest'", "catalog_credential": "'[HIDDEN]'"},
	}

	if !clause.matchesArguments([]string{"'https://catalog.example.test'"}) {
		t.Error("matchesArguments() = false for identical arguments")
	}
	if clause.matchesArguments([]string{"'https://catalog.example.test'", "'extra'"}) {
		t.Error("matchesArguments() = true for an extra argument")
	}
	if clause.matchesArguments(nil) {
		t.Error("matchesArguments() = true for missing arguments")
	}
	if !clause.matchesSettings(map[string]string{"catalog_type": "'rest'", "catalog_credential": "{credential:String}"}) { //nolint:gosec // Contains a placeholder, not a credential.
		t.Error("matchesSettings() = false for matching settings")
	}
	if clause.matchesSettings(map[string]string{"catalog_type": "'rest'"}) {
		t.Error("matchesSettings() = true for a missing setting")
	}
	if clause.matchesSettings(map[string]string{"catalog_type": "'rest'", "warehouse": "'main'"}) {
		t.Error("matchesSettings() = true for a different setting")
	}
}
