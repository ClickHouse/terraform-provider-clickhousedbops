package dbops

import (
	"context"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type Database struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Comment string `json:"comment" ch:"comment"`
}

func (i *impl) CreateDatabase(ctx context.Context, database Database) (*Database, error) {
	builder := querybuilder.NewCreateDatabase(database.Name)
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

	// Get UUID of newly created database.
	var uuid string
	{
		sql, err := querybuilder.NewSelect(
			[]querybuilder.Field{querybuilder.NewField("uuid")},
			"system.databases",
		).Where(querybuilder.SimpleWhere("name", database.Name)).Build()
		if err != nil {
			return nil, errors.WithMessage(err, "error building query")
		}

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
	}

	return i.GetDatabase(ctx, uuid)
}

func (i *impl) GetDatabase(ctx context.Context, uuid string) (*Database, error) {
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{querybuilder.NewField("name"), querybuilder.NewField("comment")},
		"system.databases",
	).Where(querybuilder.SimpleWhere("uuid", uuid)).Build()
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

func (i *impl) DeleteDatabase(ctx context.Context, uuid string) error {
	database, err := i.GetDatabase(ctx, uuid)
	if err != nil {
		return errors.WithMessage(err, "error getting database name")
	}

	if database == nil {
		// This is desired state.
		return nil
	}

	sql, err := querybuilder.NewDropDatabase(database.Name).Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}

func (i *impl) FindDatabaseByName(ctx context.Context, name string) (*Database, error) {
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{querybuilder.NewField("uuid")},
		"system.databases",
	).Where(querybuilder.SimpleWhere("name", name)).Build()
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

	return i.GetDatabase(ctx, uuid)
}
