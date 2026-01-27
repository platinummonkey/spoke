package httputil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestParseJSON(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		expectError bool
	}{
		{
			name:        "valid JSON",
			body:        `{"name": "test"}`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			body:        `{invalid}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(tt.body))
			var dest map[string]string

			err := ParseJSON(req, &dest)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "test", dest["name"])
			}
		})
	}
}

func TestParseJSONOrError(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		expectOK   bool
		expectCode int
	}{
		{
			name:     "valid JSON",
			body:     `{"name": "test"}`,
			expectOK: true,
		},
		{
			name:       "invalid JSON",
			body:       `{invalid}`,
			expectOK:   false,
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(tt.body))
			var dest map[string]string

			ok := ParseJSONOrError(w, req, &dest)

			assert.Equal(t, tt.expectOK, ok)
			if !tt.expectOK {
				assert.Equal(t, tt.expectCode, w.Code)
			}
		})
	}
}

func TestParsePathInt(t *testing.T) {
	tests := []struct {
		name        string
		pathValue   string
		expectValue int
		expectError bool
	}{
		{
			name:        "valid integer",
			pathValue:   "123",
			expectValue: 123,
			expectError: false,
		},
		{
			name:        "invalid integer",
			pathValue:   "abc",
			expectError: true,
		},
		{
			name:        "empty value",
			pathValue:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test/"+tt.pathValue, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.pathValue})

			val, err := ParsePathInt(req, "id")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectValue, val)
			}
		})
	}
}

func TestParsePathInt64(t *testing.T) {
	tests := []struct {
		name        string
		pathValue   string
		expectValue int64
		expectError bool
	}{
		{
			name:        "valid int64",
			pathValue:   "9223372036854775807",
			expectValue: 9223372036854775807,
			expectError: false,
		},
		{
			name:        "invalid int64",
			pathValue:   "abc",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test/"+tt.pathValue, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.pathValue})

			val, err := ParsePathInt64(req, "id")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectValue, val)
			}
		})
	}
}

func TestParsePathIntOrError(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test/123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})

	val, ok := ParsePathIntOrError(w, req, "id")

	assert.True(t, ok)
	assert.Equal(t, 123, val)
}

func TestParsePathIntOrError_Invalid(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})

	val, ok := ParsePathIntOrError(w, req, "id")

	assert.False(t, ok)
	assert.Equal(t, 0, val)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParsePathString(t *testing.T) {
	req := httptest.NewRequest("GET", "/test/myvalue", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "myvalue"})

	val, err := ParsePathString(req, "name")

	assert.NoError(t, err)
	assert.Equal(t, "myvalue", val)
}

func TestParseQueryInt(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?page=5", nil)

	val, err := ParseQueryInt(req, "page", 1)

	assert.NoError(t, err)
	assert.Equal(t, 5, val)
}

func TestParseQueryInt_Default(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)

	val, err := ParseQueryInt(req, "page", 1)

	assert.NoError(t, err)
	assert.Equal(t, 1, val)
}

func TestParseQueryString(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?filter=active", nil)

	val := ParseQueryString(req, "filter", "all")

	assert.Equal(t, "active", val)
}

func TestParseQueryString_Default(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)

	val := ParseQueryString(req, "filter", "all")

	assert.Equal(t, "all", val)
}

func TestParseQueryBool(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?enabled=true", nil)

	val, err := ParseQueryBool(req, "enabled", false)

	assert.NoError(t, err)
	assert.True(t, val)
}

func TestRequireNonEmpty(t *testing.T) {
	w := httptest.NewRecorder()

	ok := RequireNonEmpty(w, "", "username")

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "username is required")
}

func TestRequirePositive(t *testing.T) {
	w := httptest.NewRecorder()

	ok := RequirePositive(w, 0, "user_id")

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "user_id must be positive")
}

func TestRequireNonZero(t *testing.T) {
	w := httptest.NewRecorder()

	ok := RequireNonZero(w, 0, "count")

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "count is required")
}

func TestValidateAll(t *testing.T) {
	w := httptest.NewRecorder()

	validators := []Validator{
		func() (bool, string) { return true, "" },
		func() (bool, string) { return false, "validation failed" },
		func() (bool, string) { return true, "" },
	}

	ok := ValidateAll(w, validators...)

	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "validation failed")
}

func TestValidateAll_Success(t *testing.T) {
	w := httptest.NewRecorder()

	validators := []Validator{
		func() (bool, string) { return true, "" },
		func() (bool, string) { return true, "" },
	}

	ok := ValidateAll(w, validators...)

	assert.True(t, ok)
}

func TestGetPathVars(t *testing.T) {
	req := httptest.NewRequest("GET", "/test/123/users/456", nil)
	req = mux.SetURLVars(req, map[string]string{
		"orgId":  "123",
		"userId": "456",
	})

	vars := GetPathVars(req)

	assert.Equal(t, "123", vars["orgId"])
	assert.Equal(t, "456", vars["userId"])
}

// TestParseJSONComplexStruct tests parsing into a complex struct
func TestParseJSONComplexStruct(t *testing.T) {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	body := `{"name":"John","email":"john@example.com","age":30}`
	req := httptest.NewRequest("POST", "/users", bytes.NewBufferString(body))

	var user User
	err := ParseJSON(req, &user)

	assert.NoError(t, err)
	assert.Equal(t, "John", user.Name)
	assert.Equal(t, "john@example.com", user.Email)
	assert.Equal(t, 30, user.Age)
}

// TestParseJSONEmptyBody tests parsing an empty body
func TestParseJSONEmptyBody(t *testing.T) {
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(""))

	var dest map[string]string
	err := ParseJSON(req, &dest)

	assert.Error(t, err)
}

// BenchmarkWriteJSON benchmarks the WriteJSON function
func BenchmarkWriteJSON(b *testing.B) {
	data := map[string]string{"message": "benchmark"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		WriteJSON(w, http.StatusOK, data)
	}
}

// BenchmarkParseJSON benchmarks the ParseJSON function
func BenchmarkParseJSON(b *testing.B) {
	body, _ := json.Marshal(map[string]string{"name": "test"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(body))
		var dest map[string]string
		ParseJSON(req, &dest)
	}
}
