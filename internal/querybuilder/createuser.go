package querybuilder

import (
	"fmt"
	"strings"

	"github.com/pingcap/errors"
)

// CreateUserQueryBuilder is an interface to build CREATE USER SQL queries (already interpolated).
type CreateUserQueryBuilder interface {
	QueryBuilder
	Identified(with Identification, by string) CreateUserQueryBuilder
	WithCluster(clusterName *string) CreateUserQueryBuilder
}

type Identification string

const (
	IdentificationSHA256Hash Identification = "sha256_hash"
)

type createUserQueryBuilder struct {
	resourceName string
	identified   string
	clusterName  *string
}

func NewCreateUser(resourceName string) CreateUserQueryBuilder {
	return &createUserQueryBuilder{
		resourceName: resourceName,
	}
}

func (q *createUserQueryBuilder) Identified(with Identification, by string) CreateUserQueryBuilder {
	q.identified = fmt.Sprintf("IDENTIFIED WITH %s BY %s", with, quote(by))
	return q
}

func (q *createUserQueryBuilder) WithCluster(clusterName *string) CreateUserQueryBuilder {
	q.clusterName = clusterName
	return q
}

func (q *createUserQueryBuilder) Build() (string, error) {
	if q.resourceName == "" {
		return "", errors.New("resourceName cannot be empty for CREATE USER queries")
	}

	tokens := []string{
		"CREATE",
		"USER",
		backtick(q.resourceName),
	}
	if q.clusterName != nil {
		tokens = append(tokens, "ON", "CLUSTER", quote(*q.clusterName))
	}
	if q.identified != "" {
		tokens = append(tokens, q.identified)
	}

	return strings.Join(tokens, " ") + ";", nil
}
