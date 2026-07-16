package querybuilder

import (
	"strings"

	"github.com/pingcap/errors"
)

// CreateUserQueryBuilder is an interface to build CREATE USER SQL queries (already interpolated).
type CreateUserQueryBuilder interface {
	QueryBuilder
	Identified(methods []AuthMethod) CreateUserQueryBuilder
	WithSettingsProfile(profileName *string) CreateUserQueryBuilder
	WithCluster(clusterName *string) CreateUserQueryBuilder
	HostIPs(ips []string) CreateUserQueryBuilder
	Parameters() map[string]string
}

type createUserQueryBuilder struct {
	resourceName    string
	identified      string
	params          map[string]string
	hostIPs         []string
	settingsProfile *string
	clusterName     *string
}

func NewCreateUser(resourceName string) CreateUserQueryBuilder {
	return &createUserQueryBuilder{
		resourceName: resourceName,
	}
}

func (q *createUserQueryBuilder) Identified(methods []AuthMethod) CreateUserQueryBuilder {
	q.identified, q.params = identifiedClause(methods)
	return q
}

func (q *createUserQueryBuilder) Parameters() map[string]string {
	return q.params
}

func (q *createUserQueryBuilder) HostIPs(ips []string) CreateUserQueryBuilder {
	q.hostIPs = ips
	return q
}

func (q *createUserQueryBuilder) WithSettingsProfile(profileName *string) CreateUserQueryBuilder {
	q.settingsProfile = profileName
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
	if len(q.hostIPs) > 0 {
		for _, ip := range q.hostIPs {
			tokens = append(tokens, "HOST", "IP", quote(ip))
		}
	}
	if q.identified != "" {
		tokens = append(tokens, q.identified)
	}
	if q.settingsProfile != nil {
		tokens = append(tokens, "SETTINGS", "PROFILE", quote(*q.settingsProfile))
	}

	return strings.Join(tokens, " ") + ";", nil
}
