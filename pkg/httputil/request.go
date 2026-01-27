package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// ParseJSON decodes JSON from the request body into the destination
func ParseJSON(r *http.Request, dest interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// ParseJSONOrError decodes JSON and writes error response on failure
func ParseJSONOrError(w http.ResponseWriter, r *http.Request, dest interface{}) bool {
	if err := ParseJSON(r, dest); err != nil {
		WriteBadRequest(w, err.Error())
		return false
	}
	return true
}

// ParsePathInt extracts and parses an integer path parameter
func ParsePathInt(r *http.Request, key string) (int, error) {
	vars := mux.Vars(r)
	str := vars[key]
	if str == "" {
		return 0, fmt.Errorf("missing path parameter: %s", key)
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("invalid integer for %s: %s", key, str)
	}
	return val, nil
}

// ParsePathInt64 extracts and parses an int64 path parameter
func ParsePathInt64(r *http.Request, key string) (int64, error) {
	vars := mux.Vars(r)
	str := vars[key]
	if str == "" {
		return 0, fmt.Errorf("missing path parameter: %s", key)
	}
	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer for %s: %s", key, str)
	}
	return val, nil
}

// ParsePathIntOrError extracts an integer path parameter and writes error on failure
func ParsePathIntOrError(w http.ResponseWriter, r *http.Request, key string) (int, bool) {
	val, err := ParsePathInt(r, key)
	if err != nil {
		WriteBadRequest(w, err.Error())
		return 0, false
	}
	return val, true
}

// ParsePathInt64OrError extracts an int64 path parameter and writes error on failure
func ParsePathInt64OrError(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	val, err := ParsePathInt64(r, key)
	if err != nil {
		WriteBadRequest(w, err.Error())
		return 0, false
	}
	return val, true
}

// ParsePathString extracts a string path parameter
func ParsePathString(r *http.Request, key string) (string, error) {
	vars := mux.Vars(r)
	str := vars[key]
	if str == "" {
		return "", fmt.Errorf("missing path parameter: %s", key)
	}
	return str, nil
}

// ParsePathStringOrError extracts a string path parameter and writes error on failure
func ParsePathStringOrError(w http.ResponseWriter, r *http.Request, key string) (string, bool) {
	val, err := ParsePathString(r, key)
	if err != nil {
		WriteBadRequest(w, err.Error())
		return "", false
	}
	return val, true
}

// GetPathVars returns all path variables from the request
func GetPathVars(r *http.Request) map[string]string {
	return mux.Vars(r)
}

// ParseQueryInt extracts and parses an integer query parameter
func ParseQueryInt(r *http.Request, key string, defaultVal int) (int, error) {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultVal, nil
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("invalid integer for query param %s: %s", key, str)
	}
	return val, nil
}

// ParseQueryInt64 extracts and parses an int64 query parameter
func ParseQueryInt64(r *http.Request, key string, defaultVal int64) (int64, error) {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultVal, nil
	}
	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer for query param %s: %s", key, str)
	}
	return val, nil
}

// ParseQueryString extracts a string query parameter
func ParseQueryString(r *http.Request, key string, defaultVal string) string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// ParseQueryBool extracts and parses a boolean query parameter
func ParseQueryBool(r *http.Request, key string, defaultVal bool) (bool, error) {
	str := r.URL.Query().Get(key)
	if str == "" {
		return defaultVal, nil
	}
	val, err := strconv.ParseBool(str)
	if err != nil {
		return false, fmt.Errorf("invalid boolean for query param %s: %s", key, str)
	}
	return val, nil
}

// RequireNonEmpty validates that a string field is not empty
func RequireNonEmpty(w http.ResponseWriter, value, fieldName string) bool {
	if value == "" {
		WriteValidationError(w, fmt.Sprintf("%s is required", fieldName))
		return false
	}
	return true
}

// RequirePositive validates that an integer is positive
func RequirePositive(w http.ResponseWriter, value int64, fieldName string) bool {
	if value <= 0 {
		WriteValidationError(w, fmt.Sprintf("%s must be positive", fieldName))
		return false
	}
	return true
}

// RequireNonZero validates that an integer is not zero
func RequireNonZero(w http.ResponseWriter, value int64, fieldName string) bool {
	if value == 0 {
		WriteValidationError(w, fmt.Sprintf("%s is required", fieldName))
		return false
	}
	return true
}

// Validator is a function that validates a value and returns an error message if invalid
type Validator func() (bool, string)

// ValidateAll runs multiple validators and writes the first error
func ValidateAll(w http.ResponseWriter, validators ...Validator) bool {
	for _, validator := range validators {
		if valid, errMsg := validator(); !valid {
			WriteValidationError(w, errMsg)
			return false
		}
	}
	return true
}
