package dbops

import (
	"context"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type Database struct {
	Name    string `json:"name"`
	Comment string `json:"comment" ch:"comment"`
}

func (i *impl) CreateDatabase(ctx context.Context, database Database) (*Database, error) {
	builder := querybuilder.NewCreateDatabase(database.Name)
	if database.Comment != "" {
		builder.With(querybuilder.Comment(database.Comment))
	}
	sql, err := builder.Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return i.GetDatabase(ctx, database.Name)
}

func (i *impl) GetDatabase(ctx context.Context, name string) (*Database, error) {
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{querybuilder.NewField("name"), querybuilder.NewField("comment")},
		"system.databases",
	).With(querybuilder.Where("name", name)).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var database *Database

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		n, err := data.Get("name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'name' field")
		}
		c, err := data.Get("comment")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'comment' field")
		}
		database = &Database{
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

func (i *impl) DeleteDatabase(ctx context.Context, name string) error {
	sql, err := querybuilder.NewDropDatabase(name).Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}
