package dbops

import (
	"context"
)

type Client interface {
	CreateDatabase(ctx context.Context, database Database) (*Database, error)
	GetDatabase(ctx context.Context, name string) (*Database, error)
	DeleteDatabase(ctx context.Context, name string) error
}
