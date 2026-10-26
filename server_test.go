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

func (f *FakeDB) Update(ctx context.Context, user User) (User, error) {
	return f.User, f.Err
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
			DBErr:              ErrUserNotFound,
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

func TestUpdateUserNameHandler(t *testing.T) {
	type testCase struct {
		Name               string
		Req                *User
		DBErr              error
		ExpectedStatusCode int
		ExpectedUser       *User
	}

	testCases := []testCase{
		{
			Name: "Update user name",
			Req: &User{
				ID:   1,
				Name: "Alex Updated",
			},
			DBErr:              nil,
			ExpectedStatusCode: 200,
			ExpectedUser: &User{
				ID:   1,
				Name: "Alex Updated",
			},
		},
		{
			Name: "Wrong user name",
			Req: &User{
				ID:   1,
				Name: "  ",
			},
			DBErr:              nil,
			ExpectedStatusCode: 400,
			ExpectedUser:       &User{},
		},
		{
			Name: "Incorrect user ID",
			Req: &User{
				ID:   0,
				Name: "Alex Updated",
			},
			DBErr:              nil,
			ExpectedStatusCode: 400,
			ExpectedUser:       &User{},
		},
		{
			Name: "User not found",
			Req: &User{
				ID:   9999,
				Name: "Alex Updated",
			},
			DBErr:              ErrUserNotFound,
			ExpectedStatusCode: 404,
			ExpectedUser:       &User{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			srv := &Server{usersRepo: &FakeDB{User: *tc.Req, Err: tc.DBErr}}

			mux := http.NewServeMux()
			mux.HandleFunc("PUT /users/{id}", srv.ChangeUserNameHandler)

			pathID := strconv.FormatInt(tc.Req.ID, 10)

			bodyBytes, err := json.Marshal(ChangeUserNameRequest{Name: tc.Req.Name})
			if err != nil {
				t.Fatalf("Error marshalling request body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPut, "/users/"+pathID, bytes.NewBuffer(bodyBytes))
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

			if tc.ExpectedStatusCode != http.StatusOK {
				return
			}

			var user User
			if err := json.Unmarshal(b, &user); err != nil {
				t.Fatalf("Error unmarshalling response body: %v", err)
			}

			if user.ID != tc.ExpectedUser.ID {
				t.Fatalf("Expected user ID %d, got %d", tc.ExpectedUser.ID, user.ID)
			}

			if user.Name != tc.ExpectedUser.Name {
				t.Fatalf("Expected user name %s, got %s", tc.ExpectedUser.Name, user.Name)
			}
		})
	}
}
