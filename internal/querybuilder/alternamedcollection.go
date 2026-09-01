package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// AlterNamedCollectionQueryBuilder is an interface to build ALTER NAMED COLLECTION SQL queries (already interpolated).
// A query either sets keys or deletes them, never both: resetting a key's
// overridable flag needs the DELETE to run before the SET, and ClickHouse gives
// no ordering guarantee within a single statement.
type AlterNamedCollectionQueryBuilder interface {
	QueryBuilder
	WithCluster(clusterName *string) AlterNamedCollectionQueryBuilder
	Set(name string, value string, overridable *bool) AlterNamedCollectionQueryBuilder
	Delete(name string) AlterNamedCollectionQueryBuilder
}

type alterNamedCollectionQueryBuilder struct {
	collectionName string
	clusterName    *string
	setKeys        []namedCollectionKeyData
	deleteKeys     []string
}

func NewAlterNamedCollection(name string) AlterNamedCollectionQueryBuilder {
	return &alterNamedCollectionQueryBuilder{
		collectionName: name,
		setKeys:        make([]namedCollectionKeyData, 0),
		deleteKeys:     make([]string, 0),
	}
}

func (q *alterNamedCollectionQueryBuilder) WithCluster(clusterName *string) AlterNamedCollectionQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *alterNamedCollectionQueryBuilder) Set(name string, value string, overridable *bool) AlterNamedCollectionQueryBuilder {
	q.setKeys = append(q.setKeys, namedCollectionKeyData{
		Name:        name,
		Value:       value,
		Overridable: overridable,
	})
	return q
}

func (q *alterNamedCollectionQueryBuilder) Delete(name string) AlterNamedCollectionQueryBuilder {
	q.deleteKeys = append(q.deleteKeys, name)
	return q
}

func (q *alterNamedCollectionQueryBuilder) Build() (string, error) {
	if q.collectionName == "" {
		return "", errors.New("collectionName cannot be empty for ALTER NAMED COLLECTION queries")
	}
	if len(q.setKeys) == 0 && len(q.deleteKeys) == 0 {
		return "", errors.New("at least one SET or DELETE key is required for ALTER NAMED COLLECTION queries")
	}
	if len(q.setKeys) > 0 && len(q.deleteKeys) > 0 {
		return "", errors.New("cannot SET and DELETE keys in the same ALTER NAMED COLLECTION query")
	}

	tokens := []string{
		"ALTER",
		"NAMED COLLECTION",
		backtick(q.collectionName),
	}
	if q.clusterName != nil {
		tokens = append(tokens, "ON", "CLUSTER", quote(*q.clusterName))
	}

	if len(q.setKeys) > 0 {
		each := make([]string, 0)
		for _, k := range q.setKeys {
			sql, err := k.SQLDef()
			if err != nil {
				return "", errors.WithMessage(err, "invalid key")
			}
			each = append(each, sql)
		}
		tokens = append(tokens, "SET", strings.Join(each, ", "))
	}

	if len(q.deleteKeys) > 0 {
		tokens = append(tokens, "DELETE", strings.Join(backtickAll(q.deleteKeys), ", "))
	}

	return strings.Join(tokens, " ") + ";", nil
}
