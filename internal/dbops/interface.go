package dbops

import (
	"context"
)

type Client interface {
	CreateDatabase(ctx context.Context, database Database, clusterName *string) (*Database, error)
	GetDatabase(ctx context.Context, uuid string, clusterName *string) (*Database, error)
	DeleteDatabase(ctx context.Context, uuid string, clusterName *string) error
	FindDatabaseByName(ctx context.Context, name string, clusterName *string) (*Database, error)

	CreateRole(ctx context.Context, role Role, clusterName *string) (*Role, error)
	GetRole(ctx context.Context, id string, clusterName *string) (*Role, error)
	DeleteRole(ctx context.Context, id string, clusterName *string) error
	FindRoleByName(ctx context.Context, name string, clusterName *string) (*Role, error)

	CreateUser(ctx context.Context, user User, clusterName *string) (*User, error)
	GetUser(ctx context.Context, id string, clusterName *string) (*User, error)
	DeleteUser(ctx context.Context, id string, clusterName *string) error
	FindUserByName(ctx context.Context, name string, clusterName *string) (*User, error)

	GrantRole(ctx context.Context, grantRole GrantRole, clusterName *string) (*GrantRole, error)
	GetGrantRole(ctx context.Context, grantedRoleName string, granteeUserName *string, granteeRoleName *string, clusterName *string) (*GrantRole, error)
	RevokeGrantRole(ctx context.Context, grantedRoleName string, granteeUserName *string, granteeRoleName *string, clusterName *string) error

	GrantPrivilege(ctx context.Context, grantPrivilege GrantPrivilege, clusterName *string) (*GrantPrivilege, error)
	GetGrantPrivilege(ctx context.Context, accessType string, database *string, table *string, column *string, granteeUserName *string, granteeRoleName *string, clusterName *string) (*GrantPrivilege, error)
	RevokeGrantPrivilege(ctx context.Context, accessType string, database *string, table *string, column *string, granteeUserName *string, granteeRoleName *string, clusterName *string) error
	GetAllGrantsForGrantee(ctx context.Context, granteeUsername *string, granteeRoleName *string, clusterName *string) ([]GrantPrivilege, error)

	CreateSettingsProfile(ctx context.Context, profile SettingsProfile, clusterName *string) (*SettingsProfile, error)
	GetSettingsProfile(ctx context.Context, name string, clusterName *string) (*SettingsProfile, error)
	DeleteSettingsProfile(ctx context.Context, name string, clusterName *string) error

	IsReplicatedStorage(ctx context.Context) (bool, error)
}
