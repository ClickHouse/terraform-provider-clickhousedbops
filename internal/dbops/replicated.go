package dbops

import (
	"context"
	"strings"

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

// IsNamedCollectionsStorageReplicated reports whether named collections are stored in Keeper/ZooKeeper.
// This is independent from the RBAC storage checked by IsReplicatedStorage.
// The setting only exists since ClickHouse 26.3.26, older servers report false.
func (i *impl) IsNamedCollectionsStorageReplicated(ctx context.Context) (bool, error) {
	sql, err := querybuilder.
		NewSelect([]querybuilder.Field{querybuilder.NewField("value")}, "system.server_settings").
		Where(querybuilder.WhereEquals("name", "named_collections_storage.type")).
		Build()
	if err != nil {
		return false, errors.WithMessage(err, "error building query")
	}

	// One of: local, local_encrypted, keeper, keeper_encrypted, zookeeper, zookeeper_encrypted.
	storageType := ""

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		value, err := data.GetString("value")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'value' field")
		}

		storageType = value
		return nil
	})
	if err != nil {
		return false, errors.WithMessage(err, "error running query")
	}

	return strings.HasPrefix(storageType, "keeper") || strings.HasPrefix(storageType, "zookeeper"), nil
}
