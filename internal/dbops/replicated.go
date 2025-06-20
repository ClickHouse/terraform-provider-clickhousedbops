package dbops

import (
	"context"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

// IsReplicatedStorage queries system tables and checks if the highest priority storage system for users and roles is 'replicated'.
func (i *impl) IsReplicatedStorage(ctx context.Context) (bool, error) {
	sql, err := querybuilder.
		NewSelect([]querybuilder.Field{querybuilder.NewField("type"), querybuilder.NewField("precedence")}, "system.user_directories").
		Where(querybuilder.WhereDiffers("type", "users_xml")).
		Build()
	if err != nil {
		return false, errors.WithMessage(err, "error building query")
	}

	currentType := ""
	currentPrecedence := ^uint64(0)

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		udType, err := data.GetString("type")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'type' field")
		}
		precedence, err := data.GetUInt64("precedence")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'precedence' field")
		}

		if precedence < currentPrecedence {
			currentPrecedence = precedence
			currentType = udType
		}

		return nil
	})
	if err != nil {
		return false, errors.WithMessage(err, "error running query")
	}

	return currentType == "replicated", nil
}
