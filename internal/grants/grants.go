// Package grants exposes the ClickHouse privilege catalog (parsed from the
// embedded grants.tsv) and the coverage/scope helpers shared by the
// grant_privilege resource and the dbops client.
package grants

import (
	"bufio"
	_ "embed"
	"log"
	"strings"
	"sync"
)

//go:generate curl -so grants.tsv https://raw.githubusercontent.com/ClickHouse/ClickHouse/master/tests/queries/0_stateless/01271_show_privileges.reference
//go:embed grants.tsv
var grantsTSV string

// Catalog is the parsed privilege catalog: alias->canonical, the group
// hierarchy, and per-privilege scope categories.
type Catalog struct {
	Aliases map[string]string
	Groups  map[string][]string
	Scopes  map[string]string
}

var parsed = sync.OnceValue(func() Catalog { return ParseGrantsTSV(grantsTSV) })

// Parsed returns the catalog parsed from the embedded grants.tsv, cached once.
func Parsed() Catalog { return parsed() }

// Privilege is a privilege as SHOW PRIVILEGES lists it; nil Scope or ParentGroup means none.
type Privilege struct {
	Name        string
	Aliases     []string
	Scope       *string
	ParentGroup *string
}

// NewCatalog builds the catalog from a list of privileges.
func NewCatalog(privileges []Privilege) Catalog {
	catalog := Catalog{Aliases: make(map[string]string), Groups: make(map[string][]string), Scopes: make(map[string]string)}
	for _, p := range privileges {
		for _, alias := range p.Aliases {
			if alias != p.Name {
				catalog.Aliases[alias] = p.Name
			}
		}

		if p.ParentGroup != nil {
			catalog.Groups[*p.ParentGroup] = append(catalog.Groups[*p.ParentGroup], p.Name)
		}

		if p.Scope != nil {
			catalog.Scopes[p.Name] = *p.Scope
		}
	}
	return catalog
}

// ParseGrantsTSV builds the privilege catalog from ClickHouse's
// 01271_show_privileges.reference TSV format.
func ParseGrantsTSV(data string) Catalog {
	nullable := func(field string) *string {
		if field == `\N` {
			return nil
		}
		return &field
	}

	privileges := make([]Privilege, 0)
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		splitted := strings.Split(scanner.Text(), "\t")

		var aliases []string
		if clean := strings.ReplaceAll(strings.Trim(splitted[1], "[]"), "'", ""); clean != "" {
			aliases = strings.Split(clean, ",")
		}

		privileges = append(privileges, Privilege{
			Name:        splitted[0],
			Aliases:     aliases,
			Scope:       nullable(splitted[2]),
			ParentGroup: nullable(splitted[3]),
		})
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return NewCatalog(privileges)
}

// AllDescendants returns the privilege plus all its descendants (children,
// grandchildren, ...) from the group hierarchy. A leaf yields a single element.
func AllDescendants(groups map[string][]string, privilege string) []string {
	result := []string{privilege}
	for _, child := range groups[privilege] {
		result = append(result, AllDescendants(groups, child)...)
	}
	return result
}
