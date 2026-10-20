package dbops

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/stretchr/testify/require"

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
