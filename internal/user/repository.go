package user

import "context"

type UserRepository interface {
	Get(ctx context.Context, id int64) (*User, error)
	Create(ctx context.Context, name string) (*User, error)
	Update(ctx context.Context, id int64, name string) (*User, error)
	Delete(ctx context.Context, id int64) error
}
