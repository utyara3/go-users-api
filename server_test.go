package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type FakeUser struct {
	User  User
	Err   error
	Calls int
}

func (f *FakeUser) GetUserByID(ctx context.Context, id int64) (User, error) {
	f.Calls++
	return f.User, f.Err
}

func TestGetUserByIDHandler(t *testing.T) {
	fake := &FakeUser{
		User: User{
			ID:   1,
			Name: "Alice",
		},
	}

	srv := &Server{users: fake}

	req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
	req.SetPathValue("id", "1")

	rec := httptest.NewRecorder()

	srv.GetUserByIDHandler(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected %v status code, got %v", http.StatusOK, res.StatusCode)
	}

	expectedContentType := "application/json"
	gotContentType := res.Header.Get("Content-Type")
	if gotContentType != expectedContentType {
		t.Fatalf("expected %v content type, got %v", expectedContentType, gotContentType)
	}

	var user User
	if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if user.ID != fake.User.ID || user.Name != fake.User.Name {
		t.Fatalf("got wrong user %+v, expected %+v", user, fake.User)
	}
}
