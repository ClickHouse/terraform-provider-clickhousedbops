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

// ParseGrantsTSV builds the privilege catalog from ClickHouse's
// 01271_show_privileges.reference TSV format.
func ParseGrantsTSV(data string) Catalog {
	aliases := make(map[string]string)
	groups := make(map[string][]string)
	scopes := make(map[string]string)

	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		splitted := strings.Split(line, "\t")

		clean := strings.ReplaceAll(strings.Trim(splitted[1], "[]"), "'", "")
		if clean != "" {
			for a := range strings.SplitSeq(clean, ",") {
				if a != splitted[0] {
					aliases[a] = splitted[0]
				}
			}
		}

		if splitted[3] != "\\N" {
			if groups[splitted[3]] == nil {
				groups[splitted[3]] = make([]string, 0)
			}
			groups[splitted[3]] = append(groups[splitted[3]], splitted[0])
		}

		if splitted[2] != "\\N" {
			scopes[splitted[0]] = splitted[2]
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return Catalog{Aliases: aliases, Groups: groups, Scopes: scopes}
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
