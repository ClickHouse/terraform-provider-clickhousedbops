package dbops

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type Database struct {
	UUID            string            `json:"uuid"`
	Name            string            `json:"name"`
	Comment         string            `json:"comment" ch:"comment"`
	Engine          string            `json:"engine"`
	EngineArguments []string          `json:"-"`
	EngineSettings  map[string]string `json:"-"`
	// EngineParameters bind the {name:Type} query parameters of EngineArguments and EngineSettings server-side.
	EngineParameters map[string]string `json:"-"`
}

func (d Database) HasUUID() bool {
	return d.parsedUUID() != uuid.Nil
}

func (d Database) SameAs(other Database) bool {
	if d.HasUUID() || other.HasUUID() {
		return d.parsedUUID() == other.parsedUUID()
	}
	return d.Name == other.Name
}

func (d Database) parsedUUID() uuid.UUID {
	parsed, err := uuid.Parse(d.UUID)
	if err != nil {
		return uuid.Nil
	}
	return parsed
}

func (i *impl) CreateDatabase(ctx context.Context, database Database, clusterName *string) (*Database, error) {
	builder := querybuilder.NewCreateDatabase(database.Name).WithCluster(clusterName)
	if database.Comment != "" {
		builder.WithComment(database.Comment)
	}
	if database.Engine != "" {
		builder.WithEngine(database.Engine, database.EngineArguments, database.EngineSettings)
	}
	sql, err := builder.Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql, database.EngineParameters)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return retryWithBackoff(ctx, "database", database.Name, func() (*Database, error) {
		return i.GetDatabase(ctx, &Database{Name: database.Name}, clusterName)
	}, i.readAfterWriteTimeoutArgs()...)
}

func (i *impl) GetDatabase(ctx context.Context, database *Database, clusterName *string) (*Database, error) {
	where := querybuilder.WhereEquals("name", database.Name)
	switch {
	case database.HasUUID():
		where = querybuilder.WhereEquals("uuid", database.parsedUUID().String())
	case database.Name == "":
		return nil, errors.New("a database without its own UUID can only be found by name")
	}

	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{
			querybuilder.NewField("uuid").ToString(),
			querybuilder.NewField("name"),
			querybuilder.NewField("comment"),
			querybuilder.NewField("engine"),
		},
		"system.databases",
	).WithClusterAllReplicas(clusterName).Where(where).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var found *Database

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		u, err := data.GetString("uuid")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'uuid' field")
		}
		n, err := data.GetString("name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'name' field")
		}
		c, err := data.GetString("comment")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'comment' field")
		}
		e, err := data.GetString("engine")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'engine' field")
		}
		candidate := &Database{UUID: u, Name: n, Comment: c, Engine: e}
		if found == nil {
			found = candidate
			return nil
		}
		if found.UUID != candidate.UUID || found.Name != candidate.Name || found.Comment != candidate.Comment || found.Engine != candidate.Engine {
			return errors.Errorf(
				"database %q has inconsistent metadata across cluster replicas (UUIDs %q/%q, names %q/%q, comments %q/%q, engines %q/%q)",
				found.Name, found.UUID, candidate.UUID, found.Name, candidate.Name, found.Comment, candidate.Comment, found.Engine, candidate.Engine,
			)
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return found, nil
}

func (i *impl) DeleteDatabase(ctx context.Context, database Database, clusterName *string) error {
	found, err := i.GetDatabase(ctx, &database, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error getting database name")
	}

	if found == nil || !found.SameAs(database) {
		// This is desired state.
		return nil
	}

	sql, err := querybuilder.NewDropDatabase(found.Name).WithCluster(clusterName).Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}

// CheckDatabaseReplicas errors when the database is missing on some replicas of the cluster or has different UUIDs on them.
func (i *impl) CheckDatabaseReplicas(ctx context.Context, name string, clusterName string) error {
	sql, err := querybuilder.
		NewSelect([]querybuilder.Field{
			querybuilder.NewRawField("hostName()", "host"),
			querybuilder.NewField("uuid").ToString(),
		}, "system.databases").
		WithClusterAllReplicas(&clusterName).
		Where(querybuilder.WhereEquals("name", name)).
		Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	hostsByUUID := map[string][]string{}
	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		host, err := data.GetString("host")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'host' field")
		}
		uuid, err := data.GetString("uuid")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'uuid' field")
		}
		hostsByUUID[uuid] = append(hostsByUUID[uuid], host)
		return nil
	})
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("cannot verify database %q on all replicas of cluster %q", name, clusterName))
	}

	if len(hostsByUUID) == 0 {
		return nil
	}

	replicas, err := i.countReplicas(ctx, clusterName)
	if err != nil {
		return err
	}

	for _, uuid := range slices.Sorted(maps.Keys(hostsByUUID)) {
		hosts := hostsByUUID[uuid]
		if uint64(len(hosts)) < replicas {
			slices.Sort(hosts)
			return errors.Errorf("database with UUID %q exists on %d of %d replicas of cluster %q: %s", uuid, len(hosts), replicas, clusterName, strings.Join(hosts, ", "))
		}
	}

	return nil
}

func (i *impl) countReplicas(ctx context.Context, clusterName string) (uint64, error) {
	sql, err := querybuilder.
		NewSelect([]querybuilder.Field{querybuilder.NewRawField("uniqExact(host_name, port)", "replicas")}, "system.clusters").
		Where(querybuilder.WhereEquals("cluster", clusterName)).
		Build()
	if err != nil {
		return 0, errors.WithMessage(err, "error building query")
	}

	var replicas uint64
	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		n, err := data.GetUInt64("replicas")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'replicas' field")
		}
		replicas = n
		return nil
	})
	if err != nil {
		return 0, errors.WithMessage(err, "error running query")
	}

	return replicas, nil
}
