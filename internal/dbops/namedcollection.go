package dbops

import (
	"context"
	"sort"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

// HiddenNamedCollectionValue is what ClickHouse returns instead of a named
// collection value when the current user lacks SHOW NAMED COLLECTIONS SECRETS.
const HiddenNamedCollectionValue = "[HIDDEN]"

type NamedCollectionKey struct {
	Value       string
	Overridable *bool
}

type NamedCollection struct {
	Name string
	// On Get, values can be HiddenNamedCollectionValue.
	Keys map[string]NamedCollectionKey
}

func (i *impl) CreateNamedCollection(ctx context.Context, collection NamedCollection, clusterName *string) (*NamedCollection, error) {
	builder := querybuilder.NewCreateNamedCollection(collection.Name).WithCluster(clusterName)
	for _, name := range sortedKeyNames(collection.Keys) {
		key := collection.Keys[name]
		builder = builder.WithKey(name, key.Value, key.Overridable)
	}

	sql, err := builder.Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return retryWithBackoff(ctx, "named collection", collection.Name, func() (*NamedCollection, error) {
		return i.GetNamedCollection(ctx, collection.Name, clusterName)
	}, i.readAfterWriteTimeoutArgs()...)
}

func (i *impl) GetNamedCollection(ctx context.Context, name string, clusterName *string) (*NamedCollection, error) {
	// system.named_collections stores the keys in a Map(String, String), which the
	// clickhouse clients can't decode, so LEFT ARRAY JOIN unrolls it into one row
	// per key. LEFT keeps a row for a collection with no keys, which is how
	// existence is detected without a second query.
	sql, err := querybuilder.
		NewSelect(
			[]querybuilder.Field{
				querybuilder.NewRawField("kv.1", "key_name"),
				querybuilder.NewRawField("kv.2", "key_value"),
			},
			"system.named_collections",
		).
		WithCluster(clusterName).
		LeftArrayJoin("collection", "kv").
		Where(querybuilder.WhereEquals("name", name)).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var found bool
	keys := make(map[string]NamedCollectionKey)

	// When querying a cluster, each key appears once per replica; the map dedupes.
	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		found = true

		keyName, err := data.GetString("key_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'key_name' field")
		}

		if keyName == "" {
			// Collection exists but has no keys.
			return nil
		}

		keyValue, err := data.GetString("key_value")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'key_value' field")
		}

		keys[keyName] = NamedCollectionKey{Value: keyValue}

		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	if !found {
		// NamedCollection not found
		return nil, nil
	}

	return &NamedCollection{
		Name: name,
		Keys: keys,
	}, nil
}

func (i *impl) UpdateNamedCollection(ctx context.Context, name string, set map[string]NamedCollectionKey, deleteKeys []string, clusterName *string) (*NamedCollection, error) {
	existing, err := i.GetNamedCollection(ctx, name, clusterName)
	if err != nil {
		return nil, errors.WithMessage(err, "unable to get existing named collection")
	}

	if existing == nil {
		return nil, nil
	}

	// DELETE runs before SET: resetting a key's overridable flag to the server
	// default requires deleting the key and re-adding it.
	if len(deleteKeys) > 0 {
		builder := querybuilder.NewAlterNamedCollection(name).WithCluster(clusterName)
		for _, keyName := range deleteKeys {
			builder = builder.Delete(keyName)
		}

		sql, err := builder.Build()
		if err != nil {
			return nil, errors.WithMessage(err, "error building query")
		}

		err = i.clickhouseClient.Exec(ctx, sql)
		if err != nil {
			return nil, errors.WithMessage(err, "error running query")
		}
	}

	if len(set) > 0 {
		builder := querybuilder.NewAlterNamedCollection(name).WithCluster(clusterName)
		for _, keyName := range sortedKeyNames(set) {
			key := set[keyName]
			builder = builder.Set(keyName, key.Value, key.Overridable)
		}

		sql, err := builder.Build()
		if err != nil {
			return nil, errors.WithMessage(err, "error building query")
		}

		err = i.clickhouseClient.Exec(ctx, sql)
		if err != nil {
			return nil, errors.WithMessage(err, "error running query")
		}
	}

	return i.GetNamedCollection(ctx, name, clusterName)
}

func (i *impl) DeleteNamedCollection(ctx context.Context, name string, clusterName *string) error {
	collection, err := i.GetNamedCollection(ctx, name, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error looking up named collection")
	}

	if collection == nil {
		// Desired status
		return nil
	}

	sql, err := querybuilder.NewDropNamedCollection(name).WithCluster(clusterName).Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}

func sortedKeyNames(keys map[string]NamedCollectionKey) []string {
	names := make([]string, 0, len(keys))
	for name := range keys {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
