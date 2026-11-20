package user

import "errors"

type User struct {
	ID   int64
	Name string
}

var ErrUserNotFound = errors.New("user not found")
