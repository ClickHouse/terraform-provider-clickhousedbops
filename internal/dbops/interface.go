package dbops

import (
	"context"
)

type Client interface {
	CreateDatabase(ctx context.Context, database Database) (*Database, error)
	GetDatabase(ctx context.Context, uuid string) (*Database, error)
	DeleteDatabase(ctx context.Context, uuid string) error

	CreateRole(ctx context.Context, role Role) (*Role, error)
	GetRole(ctx context.Context, id string) (*Role, error)
	DeleteRole(ctx context.Context, id string) error

	CreateUser(ctx context.Context, user User) (*User, error)
	GetUser(ctx context.Context, id string) (*User, error)
	DeleteUser(ctx context.Context, id string) error

	GrantRole(ctx context.Context, grantRole GrantRole) (*GrantRole, error)
	GetGrantRole(ctx context.Context, grantedRoleName string, granteeUserName *string, granteeRoleName *string) (*GrantRole, error)
	RevokeGrantRole(ctx context.Context, grantedRoleName string, granteeUserName *string, granteeRoleName *string) error

	GrantPrivilege(ctx context.Context, grantPrivilege GrantPrivilege) (*GrantPrivilege, error)
	GetGrantPrivilege(ctx context.Context, accessType string, database *string, table *string, column *string, granteeUserName *string, granteeRoleName *string) (*GrantPrivilege, error)
	RevokeGrantPrivilege(ctx context.Context, accessType string, database *string, table *string, column *string, granteeUserName *string, granteeRoleName *string) error
	GetAllGrantsForGrantee(ctx context.Context, granteeUsername *string, granteeRoleName *string) ([]GrantPrivilege, error)
}
