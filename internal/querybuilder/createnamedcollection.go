package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// CreateNamedCollectionQueryBuilder is an interface to build CREATE NAMED COLLECTION SQL queries (already interpolated).
type CreateNamedCollectionQueryBuilder interface {
	QueryBuilder
	WithCluster(clusterName *string) CreateNamedCollectionQueryBuilder
	WithKey(name string, value string, overridable *bool) CreateNamedCollectionQueryBuilder
}

type namedCollectionKeyData struct {
	Name        string
	Value       string
	Overridable *bool
}

func (k *namedCollectionKeyData) SQLDef() (string, error) {
	if k.Name == "" {
		return "", errors.New("Name can't be empty")
	}

	tokens := []string{backtick(k.Name), "=", quote(k.Value)}
	if k.Overridable != nil {
		if *k.Overridable {
			tokens = append(tokens, "OVERRIDABLE")
		} else {
			tokens = append(tokens, "NOT OVERRIDABLE")
		}
	}

	return strings.Join(tokens, " "), nil
}

type createNamedCollectionQueryBuilder struct {
	collectionName string
	clusterName    *string
	keys           []namedCollectionKeyData
}

func NewCreateNamedCollection(name string) CreateNamedCollectionQueryBuilder {
	return &createNamedCollectionQueryBuilder{
		collectionName: name,
		keys:           make([]namedCollectionKeyData, 0),
	}
}

func (q *createNamedCollectionQueryBuilder) WithCluster(clusterName *string) CreateNamedCollectionQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *createNamedCollectionQueryBuilder) WithKey(name string, value string, overridable *bool) CreateNamedCollectionQueryBuilder {
	q.keys = append(q.keys, namedCollectionKeyData{
		Name:        name,
		Value:       value,
		Overridable: overridable,
	})
	return q
}

func (q *createNamedCollectionQueryBuilder) Build() (string, error) {
	if q.collectionName == "" {
		return "", errors.New("collectionName cannot be empty for CREATE NAMED COLLECTION queries")
	}
	if len(q.keys) == 0 {
		return "", errors.New("at least one key is required for CREATE NAMED COLLECTION queries")
	}

	tokens := []string{
		"CREATE",
		"NAMED COLLECTION",
		backtick(q.collectionName),
	}
	if q.clusterName != nil {
		tokens = append(tokens, "ON", "CLUSTER", quote(*q.clusterName))
	}

	each := make([]string, 0)
	for _, k := range q.keys {
		sql, err := k.SQLDef()
		if err != nil {
			return "", errors.WithMessage(err, "invalid key")
		}
		each = append(each, sql)
	}
	tokens = append(tokens, "AS", strings.Join(each, ", "))

	return strings.Join(tokens, " ") + ";", nil
}
