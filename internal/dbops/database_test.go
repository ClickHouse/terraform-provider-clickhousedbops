package dbops

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/stretchr/testify/require"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/chcontainer"
)

func TestDatabaseSameAs(t *testing.T) {
	const id = "45c84395-7b44-41d3-ade2-229b1856cf87"
	tests := []struct {
		name        string
		left, right Database
		want        bool
	}{
		{name: "same UUID", left: Database{UUID: id, Name: "a"}, right: Database{UUID: strings.ToUpper(id), Name: "b"}, want: true},
		{name: "different UUIDs", left: Database{UUID: id, Name: "a"}, right: Database{UUID: uuid.NewString(), Name: "a"}, want: false},
		{name: "UUID and no UUID", left: Database{UUID: id, Name: "a"}, right: Database{UUID: uuid.Nil.String(), Name: "a"}, want: false},
		{name: "no UUID and same name", left: Database{UUID: uuid.Nil.String(), Name: "a"}, right: Database{UUID: uuid.Nil.String(), Name: "a"}, want: true},
		{name: "no UUID and different names", left: Database{UUID: uuid.Nil.String(), Name: "a"}, right: Database{UUID: uuid.Nil.String(), Name: "b"}, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.left.SameAs(tc.right))
			require.Equal(t, tc.want, tc.right.SameAs(tc.left))
		})
	}
}

func TestCreateDatabase(t *testing.T) {
	server := chcontainer.Start(t, "")

	for _, protocol := range protocols {
		raw := server.Client(t, protocol)
		client := newTestClient(t, raw)

		t.Run(protocol+"/binds engine parameters server-side", func(t *testing.T) {
			name := acctest.RandomWithPrefix("tfacc")
			execOnCleanup(t, raw, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name))

			created, err := client.CreateDatabase(context.Background(), Database{
				Name:             name,
				Engine:           "S3",
				EngineArguments:  []string{"{url:String}", "{key_id:String}", "{key_value:String}"},
				EngineParameters: map[string]string{"url": "http://localhost:1/bucket/", "key_id": "it's", "key_value": "{key_id:String}"},
			}, nil)

			require.NoError(t, err)
			require.Equal(t, &Database{UUID: uuid.Nil.String(), Name: name, Engine: "S3"}, created)
		})
	}
}

func TestGetDatabase(t *testing.T) {
	server := chcontainer.Start(t, "")

	for _, protocol := range protocols {
		raw := server.Client(t, protocol)
		client := newTestClient(t, raw)

		t.Run(protocol+"/resolves a database sharing the zero UUID", func(t *testing.T) {
			name := acctest.RandomWithPrefix("tfacc")
			execOnCleanup(t, raw, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name))
			mustExec(t, raw, fmt.Sprintf("CREATE DATABASE `%s` ENGINE = Memory COMMENT 'memory'", name))

			found, err := client.GetDatabase(context.Background(), &Database{Name: name}, nil)
			require.NoError(t, err)
			require.Equal(t, &Database{UUID: uuid.Nil.String(), Name: name, Comment: "memory", Engine: "Memory"}, found)

			_, err = client.GetDatabase(context.Background(), &Database{UUID: uuid.Nil.String()}, nil)
			require.ErrorContains(t, err, "can only be found by name")
		})

		t.Run(protocol+"/resolves a database with its own UUID by that UUID", func(t *testing.T) {
			name := acctest.RandomWithPrefix("tfacc")
			execOnCleanup(t, raw, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name))
			created, err := client.CreateDatabase(context.Background(), Database{Name: name}, nil)
			require.NoError(t, err)

			found, err := client.GetDatabase(context.Background(), &Database{UUID: strings.ToUpper(created.UUID), Name: "other"}, nil)
			require.NoError(t, err)
			require.Equal(t, created, found)
		})

		t.Run(protocol+"/returns nil for a missing database", func(t *testing.T) {
			found, err := client.GetDatabase(context.Background(), &Database{Name: acctest.RandomWithPrefix("tfacc")}, nil)
			require.NoError(t, err)
			require.Nil(t, found)
		})
	}
}

func TestDeleteDatabase(t *testing.T) {
	server := chcontainer.Start(t, "")

	for _, protocol := range protocols {
		raw := server.Client(t, protocol)
		client := newTestClient(t, raw)

		t.Run(protocol+"/drops only the database with the given UUID", func(t *testing.T) {
			name := acctest.RandomWithPrefix("tfacc")
			execOnCleanup(t, raw, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name))
			created, err := client.CreateDatabase(context.Background(), Database{Name: name}, nil)
			require.NoError(t, err)

			require.NoError(t, client.DeleteDatabase(context.Background(), Database{Name: name, UUID: uuid.Nil.String()}, nil))
			found, err := client.GetDatabase(context.Background(), &Database{Name: name}, nil)
			require.NoError(t, err)
			require.NotNil(t, found)

			require.NoError(t, client.DeleteDatabase(context.Background(), Database{Name: name, UUID: strings.ToUpper(created.UUID)}, nil))
			found, err = client.GetDatabase(context.Background(), &Database{Name: name}, nil)
			require.NoError(t, err)
			require.Nil(t, found)
		})
	}
}
