package grantprivilege

import (
	"bufio"
	_ "embed"
	"log"
	"strings"
)

//go:generate curl -so grants.tsv https://raw.githubusercontent.com/ClickHouse/ClickHouse/master/tests/queries/0_stateless/01271_show_privileges.reference
//go:embed grants.tsv
var grants string

// parseGrants reads the grants.tsv file and turns it into a data structure to get information about all available permissions users can grant.
// The .tsv file comes from clickhouse core code and should be updated every time there is a change in permissions upstream.
// information returned by this function is used for validation of user inputs.
func parseGrants() availableGrants {
	aliases := make(map[string]string)
	groups := make(map[string][]string)
	scopes := make(map[string]string)

	scanner := bufio.NewScanner(strings.NewReader(grants))
	for scanner.Scan() {
		line := scanner.Text()

		splitted := strings.Split(line, "\t")

		clean := strings.ReplaceAll(strings.Trim(splitted[1], "[]"), "'", "")
		if clean != "" {
			for _, a := range strings.Split(clean, ",") {
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
