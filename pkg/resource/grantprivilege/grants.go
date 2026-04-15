package grantprivilege

import (
	"bufio"
	_ "embed"
	"log"
	"strings"
	"sync"
)

//go:generate curl -so grants.tsv https://raw.githubusercontent.com/ClickHouse/ClickHouse/master/tests/queries/0_stateless/01271_show_privileges.reference
//go:embed grants.tsv
var grants string

// parsedGrants caches the result of parsing the embedded grants.tsv exactly once.
// The TSV is static (embedded at build time), so the result never changes.
var parsedGrants = sync.OnceValue(parseGrants)

// parseGrants reads the grants.tsv file and turns it into a data structure to get information about all available permissions users can grant.
// The .tsv file comes from clickhouse core code and should be updated every time there is a change in permissions upstream.
// information returned by this function is used for validation of user inputs.
func parseGrants() availableGrants {
	return ParseGrantsTSV(grants)
}

// ParseGrantsTSV builds the full privilege hierarchy from TSV data.
// The TSV format comes from ClickHouse's 01271_show_privileges.reference.
func ParseGrantsTSV(data string) availableGrants {
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

	ret := availableGrants{
		Aliases: aliases,
		Groups:  groups,
		Scopes:  scopes,
	}

	return ret
}

// AllDescendants returns the privilege itself plus all its descendants
// (children, grandchildren, etc.) from the hierarchy.
// For a leaf privilege with no children it returns a single-element slice.
func AllDescendants(groups map[string][]string, privilege string) []string {
	result := []string{privilege}
	for _, child := range groups[privilege] {
		result = append(result, AllDescendants(groups, child)...)
	}
	return result
}
