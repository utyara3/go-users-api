package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// FakeDB is an in-memory UserRepository. It lets us test HTTP handlers without
// touching PostgreSQL.
type FakeDB struct {
	User     User
	CreateID int64
	Err      error
}

func (f *FakeDB) GetByID(ctx context.Context, id int64) (User, error) {
	if f.Err != nil {
		return User{}, f.Err
	}

	return f.User, nil
}

func (f *FakeDB) Create(ctx context.Context, name string) (int64, error) {
	return f.CreateID, f.Err
}

func (f *FakeDB) Update(ctx context.Context, user User) (User, error) {
	if f.Err != nil {
		return User{}, f.Err
	}

	return f.User, nil
}

func (f *FakeDB) Delete(ctx context.Context, id int64) error {
	return f.Err
}

// doRequest sends a request through the same mux the server uses in production.
func doRequest(t *testing.T, srv *Server, method, target, body string) (*httptest.ResponseRecorder, []byte) {
	t.Helper()

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, target, bodyReader)
	rec := httptest.NewRecorder()

	newMux(srv).ServeHTTP(rec, req)

	return rec, rec.Body.Bytes()
}

func checkStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()

	if rec.Code != want {
		t.Fatalf("expected status %d, got %d, body: %s", want, rec.Code, rec.Body.String())
	}
}

func checkJSONError(t *testing.T, body []byte, want string) {
	t.Helper()

	var jsonErr JSONError
	if err := json.Unmarshal(body, &jsonErr); err != nil {
		t.Fatalf("expected JSON error body, got %q: %v", string(body), err)
	}

	if jsonErr.Err != want {
		t.Fatalf("expected error %q, got %q", want, jsonErr.Err)
	}
}

func checkUser(t *testing.T, body []byte, wantID int64, wantName string) {
	t.Helper()

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		t.Fatalf("error unmarshalling response body: %v", err)
	}

	if user.ID != wantID {
		t.Fatalf("expected user ID %d, got %d", wantID, user.ID)
	}

	if user.Name != wantName {
		t.Fatalf("expected user name %q, got %q", wantName, user.Name)
	}
}

func TestGetUserByIDHandler(t *testing.T) {
	type testCase struct {
		Name               string
		Path               string
		User               User
		DBErr              error
		ExpectedStatusCode int
		ExpectedUser       *User
	}

	testCases := []testCase{
		{
			Name:               "Get user with ID = 1",
			Path:               "/users/1",
			User:               User{ID: 1, Name: "Alex"},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusOK,
			ExpectedUser:       &User{ID: 1, Name: "Alex"},
		},
		{
			Name:               "User not in db (ID = 2)",
			Path:               "/users/2",
			User:               User{ID: 2},
			DBErr:              ErrUserNotFound,
			ExpectedStatusCode: http.StatusNotFound,
			ExpectedUser:       &User{},
		},
		{
			Name:               "User ID is not a number",
			Path:               "/users/abc",
			User:               User{ID: 1},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedUser:       &User{},
		},
		{
			Name:               "User ID is not positive",
			Path:               "/users/0",
			User:               User{ID: 1},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedUser:       &User{},
		},
		{
			Name:               "Database error",
			Path:               "/users/1",
			User:               User{ID: 1},
			DBErr:              errors.New("connection refused"),
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedUser:       &User{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			srv := &Server{usersRepo: &FakeDB{User: tc.User, Err: tc.DBErr}}

			rec, body := doRequest(t, srv, http.MethodGet, tc.Path, "")
			checkStatus(t, rec, tc.ExpectedStatusCode)

			if tc.ExpectedStatusCode == http.StatusNotFound {
				checkJSONError(t, body, "User not found")
				return
			}

			if tc.ExpectedStatusCode != http.StatusOK {
				return
			}

			checkUser(t, body, tc.ExpectedUser.ID, tc.ExpectedUser.Name)
		})
	}
}

func TestCreateNewUserHandler(t *testing.T) {
	type testCase struct {
		Name               string
		Body               string
		CreateID           int64
		DBErr              error
		ExpectedStatusCode int
		ExpectedUser       *User
	}

	testCases := []testCase{
		{
			Name:               "Create user",
			Body:               `{"name": "Alex"}`,
			CreateID:           1,
			DBErr:              nil,
			ExpectedStatusCode: http.StatusCreated,
			ExpectedUser:       &User{ID: 1, Name: "Alex"},
		},
		{
			Name:               "Name is trimmed",
			Body:               `{"name": "  Alex  "}`,
			CreateID:           1,
			DBErr:              nil,
			ExpectedStatusCode: http.StatusCreated,
			ExpectedUser:       &User{ID: 1, Name: "Alex"},
		},
		{
			Name:               "Name is empty",
			Body:               `{"name": "   "}`,
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedUser:       &User{},
		},
		{
			Name:               "Name field is missing",
			Body:               `{}`,
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedUser:       &User{},
		},
		{
			Name:               "Malformed JSON",
			Body:               `{"name":`,
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedUser:       &User{},
		},
		{
			Name:               "Database error",
			Body:               `{"name": "Alex"}`,
			DBErr:              errors.New("connection refused"),
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedUser:       &User{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			srv := &Server{usersRepo: &FakeDB{CreateID: tc.CreateID, Err: tc.DBErr}}

			rec, body := doRequest(t, srv, http.MethodPost, "/users", tc.Body)
			checkStatus(t, rec, tc.ExpectedStatusCode)

			if tc.ExpectedStatusCode != http.StatusCreated {
				return
			}

			if rec.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("expected application/json content type, got %q", rec.Header().Get("Content-Type"))
			}

			checkUser(t, body, tc.ExpectedUser.ID, tc.ExpectedUser.Name)
		})
	}
}

func TestUpdateUserNameHandler(t *testing.T) {
	type testCase struct {
		Name               string
		Path               string
		Body               string
		User               User
		DBErr              error
		ExpectedStatusCode int
		ExpectedUser       *User
	}

	testCases := []testCase{
		{
			Name:               "Update user name",
			Path:               "/users/1",
			Body:               `{"name": "Alex Updated"}`,
			User:               User{ID: 1, Name: "Alex Updated"},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusOK,
			ExpectedUser:       &User{ID: 1, Name: "Alex Updated"},
		},
		{
			Name:               "Name is trimmed",
			Path:               "/users/1",
			Body:               `{"name": "  Alex Updated  "}`,
			User:               User{ID: 1, Name: "Alex Updated"},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusOK,
			ExpectedUser:       &User{ID: 1, Name: "Alex Updated"},
		},
		{
			Name:               "Wrong user name",
			Path:               "/users/1",
			Body:               `{"name": "  "}`,
			User:               User{ID: 1},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedUser:       &User{},
		},
		{
			Name:               "Malformed JSON",
			Path:               "/users/1",
			Body:               `not a json`,
			User:               User{ID: 1},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedUser:       &User{},
		},
		{
			Name:               "Incorrect user ID",
			Path:               "/users/0",
			Body:               `{"name": "Alex Updated"}`,
			User:               User{ID: 1},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedUser:       &User{},
		},
		{
			Name:               "User not found",
			Path:               "/users/9999",
			Body:               `{"name": "Alex Updated"}`,
			User:               User{ID: 9999},
			DBErr:              ErrUserNotFound,
			ExpectedStatusCode: http.StatusNotFound,
			ExpectedUser:       &User{},
		},
		{
			Name:               "Database error",
			Path:               "/users/1",
			Body:               `{"name": "Alex Updated"}`,
			User:               User{ID: 1},
			DBErr:              errors.New("connection refused"),
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedUser:       &User{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			srv := &Server{usersRepo: &FakeDB{User: tc.User, Err: tc.DBErr}}

			rec, body := doRequest(t, srv, http.MethodPut, tc.Path, tc.Body)
			checkStatus(t, rec, tc.ExpectedStatusCode)

			if tc.ExpectedStatusCode == http.StatusNotFound {
				checkJSONError(t, body, "User not found")
				return
			}

			if tc.ExpectedStatusCode != http.StatusOK {
				return
			}

			checkUser(t, body, tc.ExpectedUser.ID, tc.ExpectedUser.Name)
		})
	}
}

func TestDeleteUserHandler(t *testing.T) {
	type testCase struct {
		Name               string
		Path               string
		DBErr              error
		ExpectedStatusCode int
	}

	testCases := []testCase{
		{
			Name:               "Delete user",
			Path:               "/users/1",
			DBErr:              nil,
			ExpectedStatusCode: http.StatusNoContent,
		},
		{
			Name:               "Incorrect user ID",
			Path:               "/users/not-a-number",
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
		},
		{
			Name:               "User not found",
			Path:               "/users/9999",
			DBErr:              ErrUserNotFound,
			ExpectedStatusCode: http.StatusNotFound,
		},
		{
			Name:               "Database error",
			Path:               "/users/1",
			DBErr:              errors.New("connection refused"),
			ExpectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			srv := &Server{usersRepo: &FakeDB{Err: tc.DBErr}}

			rec, body := doRequest(t, srv, http.MethodDelete, tc.Path, "")
			checkStatus(t, rec, tc.ExpectedStatusCode)

			if tc.ExpectedStatusCode == http.StatusNotFound {
				checkJSONError(t, body, "User not found")
				return
			}

			// 204 No Content must come without a body
			if tc.ExpectedStatusCode == http.StatusNoContent && len(body) != 0 {
				t.Fatalf("expected empty body for 204 response, got %q", string(body))
			}
		})
	}
}
