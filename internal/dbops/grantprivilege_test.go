package dbops

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/grants"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/chcontainer"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/tests"
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

func TestGrantPrivilege_sourceFilter(t *testing.T) {
	filtered := func(filter string) GrantPrivilege {
		return GrantPrivilege{AccessType: "READ", AccessObject: new("URL"), SourceFilter: new(filter)}
	}

	regexp, regexpStored := filtered(`https://example\.com/.*`), "URL(`https://example\\\\.com/.*`)"
	cases := map[string][]struct {
		name        string
		existing    string
		grant       GrantPrivilege
		wantStored  string
		wantOverlap bool
	}{
		"latest": {
			{name: "grants the filter", grant: regexp, wantStored: regexpStored},
			{name: "grants an identifier-like filter shown unquoted", grant: filtered("bucket_1"), wantStored: "URL(bucket_1)"},
			{name: "grants a filter shown escaped", grant: filtered("a`b\tc\n"), wantStored: "URL(`a\\`b\\tc\\n`)"},
			{name: "grants the filter beside another filter", existing: "READ ON URL('other')", grant: regexp, wantStored: regexpStored},
			{name: "grants the source beside a filter", existing: "READ ON URL('other')", grant: GrantPrivilege{AccessType: "READ", AccessObject: new("URL")}, wantStored: "URL"},
			{name: "reports a filter absorbed by an unfiltered grant", existing: "READ ON URL", grant: regexp, wantOverlap: true},
		},
		"26.6": {
			{name: "keeps a source exact before 26.7", grant: regexp, wantStored: regexpStored},
		},
	}

	for version, tcs := range cases {
		t.Run(version, func(t *testing.T) {
			container := chcontainer.StartWithConfig(t, version, tests.ConfigSingleReadWriteGrants)

			for _, protocol := range protocols {
				raw := container.Client(t, protocol)
				client := newTestClient(t, raw)
				user := acctest.RandomWithPrefix("tfacc")
				mustExec(t, raw, fmt.Sprintf("CREATE USER `%s` IDENTIFIED WITH no_password", user))
				execOnCleanup(t, raw, fmt.Sprintf("DROP USER IF EXISTS `%s`", user))

				for _, tc := range tcs {
					t.Run(protocol+"/"+tc.name, func(t *testing.T) {
						execOnCleanup(t, raw, fmt.Sprintf("REVOKE ALL FROM `%s`", user))

						if tc.existing != "" {
							mustExec(t, raw, fmt.Sprintf("GRANT %s TO `%s`", tc.existing, user))
						}

						grant := tc.grant
						grant.GranteeUserName = &user
						got, err := client.GrantPrivilege(context.Background(), grant, nil)

						if tc.wantOverlap {
							var overlap *OverlapError
							require.ErrorAs(t, err, &overlap)
							return
						}
						require.NoError(t, err)
						require.NotNil(t, got)

						rows, err := client.GetAllGrantsForGrantee(context.Background(), &user, nil, nil)
						require.NoError(t, err)
						stored := make([]string, 0, len(rows))
						for _, row := range rows {
							stored = append(stored, *row.AccessObject)
						}
						require.Contains(t, stored, tc.wantStored)
					})
				}
			}
		})
	}
}

func TestGetGrantPrivilege_cached(t *testing.T) {
	const grantsQuery = "FROM `system`.`grants`"
	table := GrantPrivilege{AccessType: "SELECT", DatabaseName: new("db"), TableName: new("t")}
	forUser := func(grant GrantPrivilege, user string) *GrantPrivilege {
		grant.GranteeUserName = &user
		return &grant
	}

	container := chcontainer.Start(t, "latest")
	for _, protocol := range protocols {
		raw := container.Client(t, protocol)
		newUser := func(t *testing.T, grant string) string {
			user := acctest.RandomWithPrefix("tfacc")
			execOnCleanup(t, raw, fmt.Sprintf("DROP USER IF EXISTS `%s`", user))
			mustExec(t, raw, fmt.Sprintf("CREATE USER `%s` IDENTIFIED WITH no_password", user))
			if grant != "" {
				mustExec(t, raw, fmt.Sprintf("GRANT %s TO `%s`", grant, user))
			}
			return user
		}

		t.Run(protocol+"/reads a grantee once", func(t *testing.T) {
			user := newUser(t, "SELECT ON db.t")
			counting := &countingClient{ClickhouseClient: raw, match: grantsQuery}
			client := newTestClient(t, counting)

			var wg sync.WaitGroup
			for range 8 {
				wg.Go(func() {
					got, err := client.GetGrantPrivilege(context.Background(), forUser(table, user), nil, true)
					assert.NoError(t, err)
					assert.NotNil(t, got)
				})
			}
			wg.Wait()

			got, err := client.GetGrantPrivilege(context.Background(), forUser(GrantPrivilege{AccessType: "INSERT", DatabaseName: new("db"), TableName: new("t")}, user), nil, true)
			require.NoError(t, err)
			require.Nil(t, got)
			require.EqualValues(t, 1, counting.selects.Load())
		})

		t.Run(protocol+"/reads every time without allowCached", func(t *testing.T) {
			user := newUser(t, "SELECT ON db.t")
			counting := &countingClient{ClickhouseClient: raw, match: grantsQuery}
			client := newTestClient(t, counting)

			for _, allowCached := range []bool{false, false, true} {
				got, err := client.GetGrantPrivilege(context.Background(), forUser(table, user), nil, allowCached)
				require.NoError(t, err)
				require.NotNil(t, got)
			}
			require.EqualValues(t, 3, counting.selects.Load())
		})

		t.Run(protocol+"/keeps a user and a role with the same name apart", func(t *testing.T) {
			user := newUser(t, "SELECT ON db.t")
			mustExec(t, raw, fmt.Sprintf("CREATE ROLE `%s`", user))
			execOnCleanup(t, raw, fmt.Sprintf("DROP ROLE IF EXISTS `%s`", user))
			counting := &countingClient{ClickhouseClient: raw, match: grantsQuery}
			client := newTestClient(t, counting)

			got, err := client.GetGrantPrivilege(context.Background(), forUser(table, user), nil, true)
			require.NoError(t, err)
			require.NotNil(t, got)

			role := table
			role.GranteeRoleName = &user
			got, err = client.GetGrantPrivilege(context.Background(), &role, nil, true)
			require.NoError(t, err)
			require.Nil(t, got)
			require.EqualValues(t, 2, counting.selects.Load())
		})

		t.Run(protocol+"/reads again once after a statement", func(t *testing.T) {
			user := newUser(t, "SELECT ON db.t")
			counting := &countingClient{ClickhouseClient: raw, match: grantsQuery}
			client := newTestClient(t, counting)

			got, err := client.GetGrantPrivilege(context.Background(), forUser(table, user), nil, true)
			require.NoError(t, err)
			require.NotNil(t, got)

			require.NoError(t, client.RevokeGrantPrivilege(context.Background(), *forUser(table, user), nil))

			for range 2 {
				got, err = client.GetGrantPrivilege(context.Background(), forUser(table, user), nil, true)
				require.NoError(t, err)
				require.Nil(t, got)
			}
			require.EqualValues(t, 2, counting.selects.Load())
		})

		t.Run(protocol+"/does not keep a failed read", func(t *testing.T) {
			user := newUser(t, "SELECT ON db.t")
			counting := &countingClient{ClickhouseClient: raw, match: grantsQuery}
			counting.failures.Store(1)
			client := newTestClient(t, counting)

			_, err := client.GetGrantPrivilege(context.Background(), forUser(table, user), nil, true)
			require.Error(t, err)

			got, err := client.GetGrantPrivilege(context.Background(), forUser(table, user), nil, true)
			require.NoError(t, err)
			require.NotNil(t, got)
			require.EqualValues(t, 2, counting.selects.Load())
		})

		t.Run(protocol+"/does not keep grants read during a statement", func(t *testing.T) {
			user := newUser(t, "SELECT ON db.t")
			hooked := &execHookClient{ClickhouseClient: raw}
			client := newTestClient(t, hooked)
			hooked.onExec = func() {
				_, err := client.GetGrantPrivilege(context.Background(), forUser(table, user), nil, true)
				assert.NoError(t, err)
			}

			require.NoError(t, client.RevokeGrantPrivilege(context.Background(), *forUser(table, user), nil))

			got, err := client.GetGrantPrivilege(context.Background(), forUser(table, user), nil, true)
			require.NoError(t, err)
			require.Nil(t, got)
		})
	}
}
