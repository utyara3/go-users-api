package main

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

type FakeUser struct {
	User  User
	Err   error
	Calls int
}

type testCase struct {
	name           string
	method         string
	path           string
	fake           *FakeUser
	wantStatusCode int
	wantUser       *User
	wantError      string
	wantCalls      int
}

func (f *FakeUser) GetUserByID(ctx context.Context, id int64) (User, error) {
	f.Calls++
	return f.User, f.Err
}

func (f *FakeUser) CreateNewUser(ctx context.Context, name string) (int64, error) {
	f.Calls++
	return int64(rand.IntN(100) + 1), nil
}

func testingTableDrivenCases(
	t *testing.T, 
	tests []testCase,
	testFunc func(w http.ResponseWriter, r *http.Request),
) {
	for _, tc := range tests {
		t.Run(
			tc.name,
			func(t *testing.T) {
				srv := &Server{userGet: tc.fake, userCreate: tc.fake}

				req := httptest.NewRequest(tc.method, tc.path, nil)
				req.SetPathValue("id", strings.TrimPrefix(tc.path, "/users/"))
				rec := httptest.NewRecorder()

				srv.GetUserByIDHandler(rec, req)

				res := rec.Result()
				defer res.Body.Close()

				if res.StatusCode != tc.wantStatusCode {
					t.Fatalf("expected %v status code, got %v", tc.wantStatusCode, res.StatusCode)
				}

				if tc.wantUser != nil {
					var user User
					if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
						t.Fatalf("invalid user JSON: %v", err)
					}

					if user != *tc.wantUser {
						t.Fatalf("got wrong user %+v, expected %+v", user, *tc.wantUser)
					}
				} else {
					var gotError JSONError

					if err := json.NewDecoder(res.Body).Decode(&gotError); err != nil {
						t.Fatalf("invalid error JSON: %v", err)
					}

					if gotError.Err != tc.wantError {
						t.Fatalf("expected error %v, got %v", tc.wantError, gotError.Err)
					}
				}

				if tc.fake.Calls != tc.wantCalls {
					t.Fatalf("expected %d storage calls, got %d", tc.wantCalls, tc.fake.Calls)
				}
			},
		)
	}
}

func TestGetUserByIDHandler(t *testing.T) {
	tests := []testCase{
		{
			name:   "get existing user",
			method: http.MethodGet,
			path:   "/users/1",
			fake: &FakeUser{
				User: User{
					ID:   1,
					Name: "Alice",
				},
				Err: nil,
			},
			wantStatusCode: http.StatusOK,
			wantUser:       &User{ID: 1, Name: "Alice"},
			wantError:      "",
			wantCalls:      1,
		},
		{
			name:   "user not found",
			method: http.MethodGet,
			path:   "/users/999",
			fake: &FakeUser{
				Err: pgx.ErrNoRows,
			},
			wantStatusCode: http.StatusNotFound,
			wantUser:       nil,
			wantError:      "User not found",
			wantCalls:      1,
		},
		{
			name:           "invalid user ID",
			method:         http.MethodGet,
			path:           "/users/USERID",
			fake:           &FakeUser{},
			wantStatusCode: http.StatusBadRequest,
			wantUser:       nil,
			wantError:      "Invalid user ID",
			wantCalls:      0,
		},
		{
			name:   "storage error",
			method: http.MethodGet,
			path:   "/users/1",
			fake: &FakeUser{
				Err: errors.New("connection failed"),
			},
			wantStatusCode: http.StatusInternalServerError,
			wantUser:       nil,
			wantError:      "Database error",
			wantCalls:      1,
		},
	}

	testingTableDrivenCases(t, tests)
}

func TestCreateNewUserHandler(t *testing.T) {
	testCases := []testCase{
		{
			name: "create user",
			method: http.MethodGet,
			path: "/users",
			fake: &FakeUser{},
			wantStatusCode: http.StatusCreated,
			want
		},
	}
}
