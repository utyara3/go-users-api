package main

import "context"

type User struct {
	ID   int64
	Name string
}

type UserRepository interface {
	GetByID(ctx context.Context, id int64) (User, error)
	Create(ctx context.Context, name string) (User, error)
	Update(ctx context.Context, id int64, name string) (User, error)
	Delete(ctx context.Context, id int64) error
}

type UserService struct {
	userRepo UserRepository
}

type userService interface {
	GetByID(ctx context.Context, id int64) (User, error)
	Create(ctx context.Context, name string) (User, error)
	Update(ctx context.Context, id int64, name string) (User, error)
	Delete(ctx context.Context, id int64) error
}

type Server struct {
	userService userService
}

type createUserRequest struct {
	Name string `json:"name"`
}

type userResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
