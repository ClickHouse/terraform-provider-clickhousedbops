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
	WithSettingsProfile(profileName *string) CreateUserQueryBuilder
	WithCluster(clusterName *string) CreateUserQueryBuilder
	HostIPs(ips []string) CreateUserQueryBuilder
}

type Identification string

const (
	IdentificationSHA256Hash        Identification = "sha256_hash"
	IdentificationSSLCertificate    Identification = "ssl_certificate"
	IdentificationPlaintextPassword Identification = "plaintext_password"
	IdentificationBcryptHash        Identification = "bcrypt_hash"
	IdentificationDoubleSHA1Hash    Identification = "double_sha1_hash"
	IdentificationNoPassword        Identification = "no_password"
)

// identificationUseCN returns true for auth types that use CN syntax instead of BY.
func identificationUseCN(id Identification) bool {
	return id == IdentificationSSLCertificate
}

// identificationNoValue returns true for auth types that take no value.
func identificationNoValue(id Identification) bool {
	return id == IdentificationNoPassword
}

type createUserQueryBuilder struct {
	resourceName    string
	identified      string
	hostIPs         []string
	settingsProfile *string
	clusterName     *string
}

func NewCreateUser(resourceName string) CreateUserQueryBuilder {
	return &createUserQueryBuilder{
		resourceName: resourceName,
	}
}

func (q *createUserQueryBuilder) Identified(with Identification, by string) CreateUserQueryBuilder {
	switch {
	case identificationNoValue(with):
		q.identified = fmt.Sprintf("IDENTIFIED WITH %s", with)
	case identificationUseCN(with):
		q.identified = fmt.Sprintf("IDENTIFIED WITH %s CN %s", with, quote(by))
	default:
		q.identified = fmt.Sprintf("IDENTIFIED WITH %s BY %s", with, quote(by))
	}
	return q
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
