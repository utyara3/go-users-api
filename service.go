package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type UserService struct {
	userRepo UserRepository
}

type userService interface {
	Get(ctx context.Context, input GetUserInput) (*User, error)
	Create(ctx context.Context, input CreateUserInput) (*User, error)
	Update(ctx context.Context, input UpdateUserInput) (*User, error)
	Delete(ctx context.Context, input DeleteUserInput) error
}

type CreateUserInput struct {
	Name string
}

type GetUserInput struct {
	ID int64
}

type UpdateUserInput struct {
	ID   int64
	Name string
}

type DeleteUserInput struct {
	ID int64
}

var (
	ErrWrongUserID   error = errors.New("wrong user id")
	ErrWrongUserName error = errors.New("wrong user name")

	// In compile stage check that PostgresUsers realise UserRepository interface
	_ userService = (*UserService)(nil)
)

func NewUserService(userRepo UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func validateUserName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrWrongUserName
	}

	return nil
}

func validateUserID(id int64) error {
	if id <= 0 {
		return ErrWrongUserID
	}

	return nil
}

func (us *UserService) Get(ctx context.Context, input GetUserInput) (*User, error) {
	id := input.ID

	if err := validateUserID(id); err != nil {
		return nil, err
	}

	user, err := us.userRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (us *UserService) Create(ctx context.Context, input CreateUserInput) (*User, error) {
	name := strings.TrimSpace(input.Name)

	if err := validateUserName(name); err != nil {
		return nil, err
	}

	user, err := us.userRepo.Create(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (us *UserService) Update(ctx context.Context, input UpdateUserInput) (*User, error) {
	id := input.ID
	name := strings.TrimSpace(input.Name)

	if err := validateUserID(id); err != nil {
		return nil, err
	}
	if err := validateUserName(name); err != nil {
		return nil, err
	}

	user, err := us.userRepo.Update(ctx, id, name)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (us *UserService) Delete(ctx context.Context, input DeleteUserInput) error {
	id := input.ID

	if err := validateUserID(id); err != nil {
		return err
	}

	if err := us.userRepo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}
