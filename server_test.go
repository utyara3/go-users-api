package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"
)

type FakeDB struct {
	User User
	Err  error
}

func (f *FakeDB) GetByID(ctx context.Context, id int64) (User, error) {
	return f.User, f.Err
}

func (f *FakeDB) Create(ctx context.Context, name string) (int64, error) {
	return f.User.ID, f.Err
}

func (f *FakeDB) Update(ctx context.Context, user User) error {
	return f.Err
}

func (f *FakeDB) Delete(ctx context.Context, id int64) error {
	return f.Err
}

func TestGetUserByIDHandler(t *testing.T) {
	type testCase struct {
		Name               string
		User               *User
		DBErr              error
		ExpectedStatusCode int
		ExpectedUser       *User
	}

	testCases := []testCase{
		{
			Name: "Get user with ID = 1",
			User: &User{
				ID:   1,
				Name: "Alex",
			},
			DBErr:              nil,
			ExpectedStatusCode: 200,
			ExpectedUser: &User{
				ID:   1,
				Name: "Alex",
			},
		},
		{
			Name: "User not in db (ID = 2)",
			User: &User{
				ID: 2,
			},
			DBErr:              pgx.ErrNoRows,
			ExpectedStatusCode: 404,
			ExpectedUser:       &User{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			srv := &Server{usersRepo: &FakeDB{User: *tc.User, Err: tc.DBErr}}

			mux := http.NewServeMux()
			mux.HandleFunc("GET /users/{id}", srv.GetUserByIDHandler)

			req := httptest.NewRequest(http.MethodGet, "/users/"+strconv.FormatInt(tc.User.ID, 10), nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			b, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("Error reading response body: %v", err)
			}

			if res.StatusCode != tc.ExpectedStatusCode {
				t.Fatalf("Expected status code %d, got %d\nServer response: %s", tc.ExpectedStatusCode, res.StatusCode, string(b))
			}

			if tc.ExpectedStatusCode == http.StatusNotFound {
				return
			}

			var user User
			if err := json.NewDecoder(bytes.NewReader(b)).Decode(&user); err != nil {
				t.Fatalf("Error decoding response body: %v", err)
			}

			if user.ID != tc.ExpectedUser.ID {
				t.Fatalf("Expected user ID %d, got %d", tc.ExpectedUser.ID, user.ID)
			}
		})
	}
}
