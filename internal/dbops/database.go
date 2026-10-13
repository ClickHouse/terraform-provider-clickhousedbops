package dbops

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type Database struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Comment string `json:"comment" ch:"comment"`
}

func (i *impl) CreateDatabase(ctx context.Context, database Database, clusterName *string) (*Database, error) {
	builder := querybuilder.NewCreateDatabase(database.Name).WithCluster(clusterName)
	if database.Comment != "" {
		builder.WithComment(database.Comment)
	}
	sql, err := builder.Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return i.FindDatabaseByName(ctx, database.Name, clusterName)
}

func (i *impl) GetDatabase(ctx context.Context, uuid string, clusterName *string) (*Database, error) {
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{querybuilder.NewField("name"), querybuilder.NewField("comment")},
		"system.databases",
	).WithCluster(clusterName).Where(querybuilder.WhereEquals("uuid", uuid)).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var database *Database

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		n, err := data.GetString("name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'name' field")
		}
		c, err := data.GetString("comment")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'comment' field")
		}
		database = &Database{
			UUID:    uuid,
			Name:    n,
			Comment: c,
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	if database == nil {
		// Database not found
		return nil, nil
	}

	return database, nil
}

func (i *impl) DeleteDatabase(ctx context.Context, uuid string, clusterName *string) error {
	database, err := i.GetDatabase(ctx, uuid, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error getting database name")
	}

	if database == nil {
		// This is desired state.
		return nil
	}

	sql, err := querybuilder.NewDropDatabase(database.Name).WithCluster(clusterName).Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}

func (i *impl) FindDatabaseByName(ctx context.Context, name string, clusterName *string) (*Database, error) {
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{querybuilder.NewField("uuid").ToString()},
		"system.databases",
	).WithCluster(clusterName).Where(querybuilder.WhereEquals("name", name)).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var uuid string

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		uuid, err = data.GetString("uuid")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'uuid' field")
		}

		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	if uuid == "" {
		return nil, errors.New("database with such name not found")
	}

	return i.GetDatabase(ctx, uuid, clusterName)
}

// CheckDatabaseReplicas errors when the database UUID is present only on some replicas of the cluster.
func (i *impl) CheckDatabaseReplicas(ctx context.Context, uuid string, clusterName string) error {
	sql, err := querybuilder.
		NewSelect([]querybuilder.Field{querybuilder.NewRawField("hostName()", "host")}, "system.databases").
		WithClusterAllReplicas(&clusterName).
		Where(querybuilder.WhereEquals("uuid", uuid)).
		Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	var hosts []string
	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		host, err := data.GetString("host")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'host' field")
		}
		hosts = append(hosts, host)
		return nil
	})
	if err != nil {
		return errors.WithMessage(err, fmt.Sprintf("cannot verify database with UUID %q on all replicas of cluster %q", uuid, clusterName))
	}

	if len(hosts) == 0 {
		return nil
	}

	replicas, err := i.countReplicas(ctx, clusterName)
	if err != nil {
		return err
	}

	if uint64(len(hosts)) < replicas {
		slices.Sort(hosts)
		return errors.Errorf("database with UUID %q exists on %d of %d replicas of cluster %q: %s", uuid, len(hosts), replicas, clusterName, strings.Join(hosts, ", "))
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
