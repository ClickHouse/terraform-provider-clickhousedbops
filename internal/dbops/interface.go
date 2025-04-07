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
}
