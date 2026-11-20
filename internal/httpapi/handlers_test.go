package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/utyara3/go-users-api/internal/httpapi"
	"github.com/utyara3/go-users-api/internal/user"
)

type FakeDB struct {
	User     user.User
	CreateID int64
	Err      error
}

func (f *FakeDB) Get(ctx context.Context, id int64) (*user.User, error) {
	if f.Err != nil {
		return nil, f.Err
	}

	user := f.User
	return &user, nil
}

func (f *FakeDB) Create(ctx context.Context, name string) (*user.User, error) {
	if f.Err != nil {
		return nil, f.Err
	}

	return &user.User{ID: f.CreateID, Name: name}, nil
}

func (f *FakeDB) Update(ctx context.Context, id int64, name string) (*user.User, error) {
	if f.Err != nil {
		return nil, f.Err
	}

	user := user.User{ID: id, Name: name}
	return &user, nil
}

func (f *FakeDB) Delete(ctx context.Context, id int64) error {
	return f.Err
}

// newTestServer wires a Server the same way main.go does, with a logger that
// discards output. For assertions on the logs themselves see
// TestHandleErrorLogsInternalErrors.
func newTestServer(repo *FakeDB) *httpapi.Server {
	return httpapi.NewServer(
		user.NewUserService(repo),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
}

// doRequest sends a request through the same mux the server uses in production.
func doRequest(t *testing.T, srv *httpapi.Server, method, target, body string) (*httptest.ResponseRecorder, []byte) {
	t.Helper()

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, target, bodyReader)
	rec := httptest.NewRecorder()

	httpapi.NewMux(srv).ServeHTTP(rec, req)

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

	var jsonErr httpapi.JSONError
	if err := json.Unmarshal(body, &jsonErr); err != nil {
		t.Fatalf("expected JSON error body, got %q: %v", string(body), err)
	}

	if jsonErr.Err != want {
		t.Fatalf("expected error %q, got %q", want, jsonErr.Err)
	}
}

// checkErrorResponse verifies status and error message for non-success responses.
func checkErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, body []byte, wantStatus int, wantMsg string) {
	t.Helper()

	checkStatus(t, rec, wantStatus)

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected application/json content type, got %q", rec.Header().Get("Content-Type"))
	}

	checkJSONError(t, body, wantMsg)
}

func checkUser(t *testing.T, body []byte, wantID int64, wantName string) {
	t.Helper()

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("error unmarshalling response body: %v", err)
	}

	idRaw, ok := resp["id"]
	if !ok {
		t.Fatalf(`expected JSON key "id", body: %s`, body)
	}

	nameRaw, ok := resp["name"]
	if !ok {
		t.Fatalf(`expected JSON key "name", body: %s`, body)
	}

	var gotID int64
	if err := json.Unmarshal(idRaw, &gotID); err != nil {
		t.Fatalf("error unmarshalling id: %v, body: %s", err, body)
	}

	if gotID != wantID {
		t.Fatalf("expected user ID %d, got %d", wantID, gotID)
	}

	var gotName string
	if err := json.Unmarshal(nameRaw, &gotName); err != nil {
		t.Fatalf("error unmarshalling name: %v, body: %s", err, body)
	}

	if gotName != wantName {
		t.Fatalf("expected user name %q, got %q", wantName, gotName)
	}
}

func TestGetUserHandler(t *testing.T) {
	type testCase struct {
		Name               string
		Path               string
		User               user.User
		DBErr              error
		ExpectedStatusCode int
		ExpectedErrMsg     string
		ExpectedUser       *user.User
	}

	testCases := []testCase{
		{
			Name:               "Get user with ID = 1",
			Path:               "/users/1",
			User:               user.User{ID: 1, Name: "Alex"},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusOK,
			ExpectedUser:       &user.User{ID: 1, Name: "Alex"},
		},
		{
			Name:               "User not in db (ID = 2)",
			Path:               "/users/2",
			User:               user.User{ID: 2},
			DBErr:              user.ErrUserNotFound,
			ExpectedStatusCode: http.StatusNotFound,
			ExpectedErrMsg:     "User not found",
		},
		{
			Name:               "User ID is not a number",
			Path:               "/users/abc",
			User:               user.User{ID: 1},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedErrMsg:     "Wrong user ID",
		},
		{
			Name:               "User ID is not positive",
			Path:               "/users/0",
			User:               user.User{ID: 1},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedErrMsg:     "Wrong user ID",
		},
		{
			Name:               "Database error",
			Path:               "/users/1",
			User:               user.User{ID: 1},
			DBErr:              errors.New("connection refused"),
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedErrMsg:     "Internal server error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			srv := newTestServer(&FakeDB{User: tc.User, Err: tc.DBErr})

			rec, body := doRequest(t, srv, http.MethodGet, tc.Path, "")

			if tc.ExpectedStatusCode >= 400 {
				checkErrorResponse(t, rec, body, tc.ExpectedStatusCode, tc.ExpectedErrMsg)
				return
			}

			checkStatus(t, rec, tc.ExpectedStatusCode)
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
		ExpectedErrMsg     string
		ExpectedUser       *user.User
	}

	testCases := []testCase{
		{
			Name:               "Create user",
			Body:               `{"name": "Alex"}`,
			CreateID:           1,
			DBErr:              nil,
			ExpectedStatusCode: http.StatusCreated,
			ExpectedUser:       &user.User{ID: 1, Name: "Alex"},
		},
		{
			Name:               "Name is trimmed",
			Body:               `{"name": "  Alex  "}`,
			CreateID:           1,
			DBErr:              nil,
			ExpectedStatusCode: http.StatusCreated,
			ExpectedUser:       &user.User{ID: 1, Name: "Alex"},
		},
		{
			Name:               "Name is empty",
			Body:               `{"name": "   "}`,
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedErrMsg:     "Wrong user name",
		},
		{
			Name:               "Name field is missing",
			Body:               `{}`,
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedErrMsg:     "Wrong user name",
		},
		{
			Name:               "Malformed JSON",
			Body:               `{"name":`,
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedErrMsg:     "Invalid JSON",
		},
		{
			Name:               "Database error",
			Body:               `{"name": "Alex"}`,
			DBErr:              errors.New("connection refused"),
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedErrMsg:     "Internal server error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			srv := newTestServer(&FakeDB{CreateID: tc.CreateID, Err: tc.DBErr})

			rec, body := doRequest(t, srv, http.MethodPost, "/users", tc.Body)

			if tc.ExpectedStatusCode >= 400 {
				checkErrorResponse(t, rec, body, tc.ExpectedStatusCode, tc.ExpectedErrMsg)
				return
			}

			checkStatus(t, rec, tc.ExpectedStatusCode)

			if rec.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("expected application/json content type, got %q", rec.Header().Get("Content-Type"))
			}

			checkUser(t, body, tc.ExpectedUser.ID, tc.ExpectedUser.Name)
		})
	}
}

func TestUpdateUserHandler(t *testing.T) {
	type testCase struct {
		Name               string
		Path               string
		Body               string
		User               user.User
		DBErr              error
		ExpectedStatusCode int
		ExpectedErrMsg     string
		ExpectedUser       *user.User
	}

	testCases := []testCase{
		{
			Name:               "Update user name",
			Path:               "/users/1",
			Body:               `{"name": "Alex Updated"}`,
			User:               user.User{ID: 1, Name: "Alex Updated"},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusOK,
			ExpectedUser:       &user.User{ID: 1, Name: "Alex Updated"},
		},
		{
			Name:               "Name is trimmed",
			Path:               "/users/1",
			Body:               `{"name": "  Alex Updated  "}`,
			User:               user.User{ID: 1, Name: "Alex Updated"},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusOK,
			ExpectedUser:       &user.User{ID: 1, Name: "Alex Updated"},
		},
		{
			Name:               "Wrong user name",
			Path:               "/users/1",
			Body:               `{"name": "  "}`,
			User:               user.User{ID: 1},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedErrMsg:     "Wrong user name",
		},
		{
			Name:               "Malformed JSON",
			Path:               "/users/1",
			Body:               `not a json`,
			User:               user.User{ID: 1},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedErrMsg:     "Invalid JSON",
		},
		{
			Name:               "Incorrect user ID",
			Path:               "/users/0",
			Body:               `{"name": "Alex Updated"}`,
			User:               user.User{ID: 1},
			DBErr:              nil,
			ExpectedStatusCode: http.StatusBadRequest,
			ExpectedErrMsg:     "Wrong user ID",
		},
		{
			Name:               "User not found",
			Path:               "/users/9999",
			Body:               `{"name": "Alex Updated"}`,
			User:               user.User{ID: 9999},
			DBErr:              user.ErrUserNotFound,
			ExpectedStatusCode: http.StatusNotFound,
			ExpectedErrMsg:     "User not found",
		},
		{
			Name:               "Database error",
			Path:               "/users/1",
			Body:               `{"name": "Alex Updated"}`,
			User:               user.User{ID: 1},
			DBErr:              errors.New("connection refused"),
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedErrMsg:     "Internal server error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			srv := newTestServer(&FakeDB{User: tc.User, Err: tc.DBErr})

			rec, body := doRequest(t, srv, http.MethodPut, tc.Path, tc.Body)

			if tc.ExpectedStatusCode >= 400 {
				checkErrorResponse(t, rec, body, tc.ExpectedStatusCode, tc.ExpectedErrMsg)
				return
			}

			checkStatus(t, rec, tc.ExpectedStatusCode)
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
		ExpectedErrMsg     string
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
			ExpectedErrMsg:     "Wrong user ID",
		},
		{
			Name:               "User not found",
			Path:               "/users/9999",
			DBErr:              user.ErrUserNotFound,
			ExpectedStatusCode: http.StatusNotFound,
			ExpectedErrMsg:     "User not found",
		},
		{
			Name:               "Database error",
			Path:               "/users/1",
			DBErr:              errors.New("connection refused"),
			ExpectedStatusCode: http.StatusInternalServerError,
			ExpectedErrMsg:     "Internal server error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			srv := newTestServer(&FakeDB{Err: tc.DBErr})

			rec, body := doRequest(t, srv, http.MethodDelete, tc.Path, "")

			if tc.ExpectedStatusCode >= 400 {
				checkErrorResponse(t, rec, body, tc.ExpectedStatusCode, tc.ExpectedErrMsg)
				return
			}

			checkStatus(t, rec, tc.ExpectedStatusCode)

			// 204 No Content must come without a body
			if tc.ExpectedStatusCode == http.StatusNoContent && len(body) != 0 {
				t.Fatalf("expected empty body for 204 response, got %q", string(body))
			}
		})
	}
}

// TestHandleErrorLogsInternalErrors ensures unexpected errors are visible in
// the logs as structured JSON with enough request context to debug them.
func TestHandleErrorLogsInternalErrors(t *testing.T) {
	var logBuf bytes.Buffer

	srv := httpapi.NewServer(
		user.NewUserService(&FakeDB{Err: errors.New("connection refused")}),
		slog.New(slog.NewJSONHandler(&logBuf, nil)),
	)

	rec, body := doRequest(t, srv, http.MethodGet, "/users/1", "")
	checkErrorResponse(t, rec, body, http.StatusInternalServerError, "Internal server error")

	var entry map[string]any
	if err := json.Unmarshal(logBuf.Bytes(), &entry); err != nil {
		t.Fatalf("expected a JSON log record, got %q: %v", logBuf.String(), err)
	}

	if entry["level"] != "ERROR" {
		t.Fatalf("expected log level ERROR, got %v", entry["level"])
	}

	if entry["msg"] != "Internal server error" {
		t.Fatalf("expected log msg %q, got %v", "Internal server error", entry["msg"])
	}

	if entry["error"] != "connection refused" {
		t.Fatalf("expected original error in the log, got %v", entry["error"])
	}

	if entry["method"] != http.MethodGet || entry["path"] != "/users/1" {
		t.Fatalf("expected request context in the log, got method=%v path=%v", entry["method"], entry["path"])
	}
}
