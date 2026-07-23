package dbops

import (
	"context"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type Database struct {
	UUID             string            `json:"uuid"`
	Name             string            `json:"name"`
	Comment          string            `json:"comment" ch:"comment"`
	Engine           string            `json:"engine"`
	EngineArguments  []string          `json:"-"`
	EngineSettings   map[string]string `json:"-"`
	EngineParameters map[string]string `json:"-"`
}

func (i *impl) CreateDatabase(ctx context.Context, database Database, clusterName *string) (*Database, error) {
	builder := querybuilder.NewCreateDatabase(database.Name).WithCluster(clusterName)
	if database.Comment != "" {
		builder.WithComment(database.Comment)
	}
	if database.Engine != "" {
		builder.WithEngine(database.Engine, database.EngineArguments, database.EngineSettings).
			WithParameters(database.EngineParameters)
	}
	sql, err := builder.Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var execErr error
	if len(database.EngineParameters) > 0 {
		redactedSQL, err := builder.RedactedQuery()
		if err != nil {
			return nil, errors.WithMessage(err, "error building redacted query")
		}
		sensitiveValues := make([]string, 0, len(database.EngineParameters))
		for _, value := range database.EngineParameters {
			sensitiveValues = append(sensitiveValues, value)
		}
		execErr = i.clickhouseClient.ExecSensitive(ctx, sql, redactedSQL, sensitiveValues)
	} else {
		execErr = i.clickhouseClient.Exec(ctx, sql)
	}
	if execErr != nil {
		return nil, errors.WithMessage(execErr, "error running query")
	}

	return retryWithBackoff(ctx, "database", database.Name, func() (*Database, error) {
		return i.FindDatabaseByName(ctx, database.Name, clusterName)
	})
}

func (i *impl) GetDatabase(ctx context.Context, uuid string, clusterName *string) (*Database, error) {
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{querybuilder.NewField("name"), querybuilder.NewField("comment"), querybuilder.NewField("engine")},
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
		e, err := data.GetString("engine")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'engine' field")
		}
		candidate := &Database{
			UUID:    uuid,
			Name:    n,
			Comment: c,
			Engine:  e,
		}
		if database == nil {
			database = candidate
			return nil
		}
		if database.Name != candidate.Name {
			return errors.Errorf(
				"database UUID %q matches multiple database names (%q and %q); import and manage this database by name instead",
				uuid, database.Name, candidate.Name,
			)
		}
		if database.Comment != candidate.Comment || database.Engine != candidate.Engine {
			return errors.Errorf(
				"database %q has inconsistent metadata across cluster replicas (comments %q/%q, engines %q/%q)",
				database.Name, database.Comment, candidate.Comment, database.Engine, candidate.Engine,
			)
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

func (i *impl) DeleteDatabase(ctx context.Context, name string, clusterName *string) error {
	database, err := i.FindDatabaseByName(ctx, name, clusterName)
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
		[]querybuilder.Field{
			querybuilder.NewField("uuid").ToString(),
			querybuilder.NewField("name"),
			querybuilder.NewField("comment"),
			querybuilder.NewField("engine"),
		},
		"system.databases",
	).WithCluster(clusterName).Where(querybuilder.WhereEquals("name", name)).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var database *Database

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		uuid, err := data.GetString("uuid")
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
		if database == nil {
			database = &Database{UUID: uuid, Name: n, Comment: c, Engine: e}
			return nil
		}

		if database.Name != n || database.Comment != c || database.Engine != e {
			return errors.Errorf(
				"database %q has inconsistent metadata across cluster replicas (comments %q/%q, engines %q/%q)",
				name, database.Comment, c, database.Engine, e,
			)
		}
		// UUIDs for local engines such as Atomic may legitimately differ by replica.
		// Pick a deterministic representative when resolving a name for the first time.
		if uuid < database.UUID {
			database.UUID = uuid
		}

		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	if database == nil {
		return nil, nil
	}

	return database, nil
}
