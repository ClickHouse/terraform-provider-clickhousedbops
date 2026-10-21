package dbops

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/stretchr/testify/require"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/grants"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/chcontainer"
)

func TestGetAllGrantsForGrantee(t *testing.T) {
	tests := map[string][]struct {
		name  string
		grant string
		want  GrantPrivilege
	}{
		"latest": {
			{"restores a database wildcard", "SELECT ON db*.*", GrantPrivilege{AccessType: "SELECT", DatabaseName: new("db*")}},
			{"restores a table wildcard", "SELECT ON db.t*", GrantPrivilege{AccessType: "SELECT", DatabaseName: new("db"), TableName: new("t*")}},
			{"restores an access object wildcard", "NAMED COLLECTION ON nc*", GrantPrivilege{AccessType: "NAMED COLLECTION", AccessObject: new("nc*")}},
			{"keeps an exact table", "SELECT ON db.t", GrantPrivilege{AccessType: "SELECT", DatabaseName: new("db"), TableName: new("t")}},
		},
		"26.6": {
			{"keeps a global grant before 26.7", "SELECT ON *.*", GrantPrivilege{AccessType: "SELECT"}},
			{"treats a database as a wildcard before 26.7", "SELECT ON db.*", GrantPrivilege{AccessType: "SELECT", DatabaseName: new("db*")}},
			{"treats a table as a wildcard before 26.7", "SELECT ON db.t", GrantPrivilege{AccessType: "SELECT", DatabaseName: new("db"), TableName: new("t*")}},
			{"treats an access object as a wildcard before 26.7", "NAMED COLLECTION ON nc", GrantPrivilege{AccessType: "NAMED COLLECTION", AccessObject: new("nc*")}},
		},
	}

	for version, tcs := range tests {
		t.Run(version, func(t *testing.T) {
			container := chcontainer.Start(t, version)

			for _, protocol := range protocols {
				raw := container.Client(t, protocol)
				client := newTestClient(t, raw)
				for _, tc := range tcs {
					t.Run(protocol+"/"+tc.name, func(t *testing.T) {
						user := acctest.RandomWithPrefix("tfacc")
						execOnCleanup(t, raw, fmt.Sprintf("DROP USER IF EXISTS `%s`", user))
						mustExec(t, raw, fmt.Sprintf("CREATE USER `%s` IDENTIFIED WITH no_password", user))
						mustExec(t, raw, fmt.Sprintf("GRANT %s TO `%s`", tc.grant, user))

						got, err := client.GetAllGrantsForGrantee(context.Background(), &user, nil, nil)
						require.NoError(t, err)

						want := tc.want
						want.GranteeUserName = &user
						require.Equal(t, []GrantPrivilege{want}, got)
					})
				}
			}
		})
	}
}

func TestGrantPrivilege(t *testing.T) {
	table := GrantPrivilege{AccessType: "SELECT", DatabaseName: new("db"), TableName: new("t")}
	column := GrantPrivilege{AccessType: "SELECT", DatabaseName: new("db"), TableName: new("t"), ColumnName: new("c")}
	tests := map[string][]struct {
		name        string
		existing    string
		grant       GrantPrivilege
		misses      int
		wantOverlap bool
	}{
		"latest": {
			{name: "waits for a lagging row of the grant", grant: table, misses: 1},
			{name: "finds a column grant", grant: column},
			{name: "finds a privilege stored as its members", grant: GrantPrivilege{AccessType: "CREATE", DatabaseName: new("db"), TableName: new("t"), ExpandedAccessTypes: grants.AllDescendants(grants.Parsed().Groups, "CREATE")}},
			{name: "finds a legacy source grant stored as READ and WRITE", grant: GrantPrivilege{AccessType: "S3", ExpandedAccessTypes: grants.AllDescendants(grants.Parsed().Groups, "S3")}},
			{name: "waits for a lagging broader grant", existing: "SELECT ON db.*", grant: table, misses: 2, wantOverlap: true},
			{name: "reports a table grant that absorbed a column grant", existing: "SELECT ON db.t", grant: column, wantOverlap: true},
			{name: "tells the grant from a same-named wildcard", existing: "SELECT ON db.t*", grant: table, wantOverlap: true},
		},
		"26.6": {
			{name: "takes a same-named wildcard for the grant before 26.7", existing: "SELECT ON db.t*", grant: table},
		},
	}

	for version, tcs := range tests {
		t.Run(version, func(t *testing.T) {
			container := chcontainer.Start(t, version)

			for _, protocol := range protocols {
				raw := container.Client(t, protocol)
				lagging := &laggingClient{ClickhouseClient: raw, match: "FROM `system`.`grants`"}
				client := newTestClient(t, lagging)
				for _, tc := range tcs {
					t.Run(protocol+"/"+tc.name, func(t *testing.T) {
						user := acctest.RandomWithPrefix("tfacc")
						execOnCleanup(t, raw, fmt.Sprintf("DROP USER IF EXISTS `%s`", user))
						mustExec(t, raw, fmt.Sprintf("CREATE USER `%s` IDENTIFIED WITH no_password", user))
						if tc.existing != "" {
							mustExec(t, raw, fmt.Sprintf("GRANT %s TO `%s`", tc.existing, user))
						}
						lagging.misses = tc.misses

						grant := tc.grant
						grant.GranteeUserName = &user
						got, err := client.GrantPrivilege(context.Background(), grant, nil)

						if tc.wantOverlap {
							var overlap *OverlapError
							require.ErrorAs(t, err, &overlap)
							require.NotEmpty(t, overlap.Covering)
						} else {
							require.NoError(t, err)
							require.NotNil(t, got)
						}
						require.Zero(t, lagging.misses, "lagging reads left")
					})
				}
			}
		})
	}
}
