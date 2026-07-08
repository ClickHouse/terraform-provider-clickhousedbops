package querybuilder

import (
	"fmt"

	"github.com/pingcap/errors"
)

// SelectNamedCollectionKeysQueryBuilder builds a SELECT query returning one row
// per key of a named collection, as plain String columns 'key_name' and 'key_value'.
// system.named_collections stores keys in a Map(String, String) column, which the
// clickhouse clients can't decode; ARRAY JOIN unrolls it to scalar columns.
type SelectNamedCollectionKeysQueryBuilder interface {
	QueryBuilder
	WithCluster(clusterName *string) SelectNamedCollectionKeysQueryBuilder
}

type selectNamedCollectionKeysQueryBuilder struct {
	collectionName string
	clusterName    *string
}

func NewSelectNamedCollectionKeys(collectionName string) SelectNamedCollectionKeysQueryBuilder {
	return &selectNamedCollectionKeysQueryBuilder{
		collectionName: collectionName,
	}
}

func (q *selectNamedCollectionKeysQueryBuilder) WithCluster(clusterName *string) SelectNamedCollectionKeysQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *selectNamedCollectionKeysQueryBuilder) Build() (string, error) {
	if q.collectionName == "" {
		return "", errors.New("collectionName cannot be empty for SELECT named collection keys queries")
	}

	from := fmt.Sprintf("%s.%s", backtick("system"), backtick("named_collections"))
	if q.clusterName != nil {
		from = fmt.Sprintf("cluster(%s, %s)", quote(*q.clusterName), from)
	}

	return fmt.Sprintf(
		"SELECT kv.1 AS key_name, kv.2 AS key_value FROM %s ARRAY JOIN %s AS kv WHERE %s = %s;",
		from,
		backtick("collection"),
		backtick("name"),
		quote(q.collectionName),
	), nil
}
