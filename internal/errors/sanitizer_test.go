package errors

import (
	"fmt"
	"strings"
	"testing"
)

// Real SHA256 hash for testing
const testHash = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		category ErrorCategory
		want     string
	}{
		{
			name:     "password hash in SQL query",
			input:    fmt.Errorf("executing query: CREATE USER test IDENTIFIED WITH sha256_hash BY '%s'", testHash),
			category: CategoryDatabase,
			want:     "[SQL QUERY]", // SQL sanitization takes precedence
		},
		{
			name:     "connection error with credentials",
			input:    fmt.Errorf("failed to connect to clickhouse://user:secret123@localhost:9000/default"),
			category: CategoryDatabase,
			want:     "clickhouse://user:[REDACTED]@",
		},
		{
			name:     "standalone SHA256 hash",
			input:    fmt.Errorf("validation failed for hash %s", testHash),
			category: CategoryValidation,
			want:     "[REDACTED]",
		},
		{
			name:     "nil error",
			input:    nil,
			category: CategoryGeneral,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.input, tt.category)
			if tt.input == nil {
				if result != nil {
					t.Errorf("SanitizeError() returned non-nil for nil input")
				}
				return
			}

			resultStr := result.Error()
			if !strings.Contains(resultStr, tt.want) && tt.want != "" {
				t.Errorf("SanitizeError() = %v, want to contain %v", resultStr, tt.want)
			}

			// Ensure no sensitive patterns remain - check for original hash specifically
			if strings.Contains(resultStr, testHash) {
				t.Errorf("SanitizeError() still contains sensitive hash: %v", resultStr)
			}
			if strings.Contains(resultStr, "secret123") {
				t.Errorf("SanitizeError() still contains credentials: %v", resultStr)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		category       ErrorCategory
		want           string
		mustNotContain []string
	}{
		{
			name:           "CREATE USER with password hash",
			input:          fmt.Sprintf("CREATE USER test IDENTIFIED WITH sha256_hash BY '%s'", testHash),
			category:       CategoryDatabase,
			want:           "[REDACTED]",
			mustNotContain: []string{testHash},
		},
		{
			name:           "ClickHouse error with SQL query",
			input:          "Code: 516, e.message = Unknown user, executing query: CREATE USER test IDENTIFIED WITH sha256_hash BY 'hash123'",
			category:       CategoryDatabase,
			want:           "executing query: [SQL QUERY]",
			mustNotContain: []string{"hash123", "CREATE USER test"},
		},
		{
			name:           "Stack trace with file paths",
			input:          "error occurred\n    at /usr/local/lib/clickhouse/user.go:123\n    at /home/user/app/main.go:456",
			category:       CategoryGeneral,
			want:           "error occurred",
			mustNotContain: []string{"/usr/local/lib", "/home/user"},
		},
		{
			name:           "API key in configuration",
			input:          "invalid api_key: 'sk_test_1234567890abcdef'",
			category:       CategoryAuthentication,
			want:           "[REDACTED]",
			mustNotContain: []string{"sk_test_1234567890abcdef"},
		},
		{
			name:           "Connection string with password",
			input:          "failed to connect to clickhouse://user:mypassword123@localhost:9000/default",
			category:       CategoryDatabase,
			want:           "[REDACTED]",
			mustNotContain: []string{"mypassword123"},
		},
		{
			name:           "Standalone hash without SQL context",
			input:          fmt.Sprintf("Invalid hash provided: %s for authentication", testHash),
			category:       CategoryGeneral,
			want:           "[REDACTED]",
			mustNotContain: []string{testHash},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeString(tt.input, tt.category)

			if tt.want != "" && !strings.Contains(result, tt.want) {
				t.Errorf("sanitizeString() = %v, want to contain %v", result, tt.want)
			}

			for _, forbidden := range tt.mustNotContain {
				if strings.Contains(result, forbidden) {
					t.Errorf("sanitizeString() = %v, must not contain %v", result, forbidden)
				}
			}
		})
	}
}

func TestCreateSecureErrorMessage(t *testing.T) {
	tests := []struct {
		name                   string
		operation              string
		resourceType           string
		inputError             error
		expectedUserMessage    string
		expectedTechnicalEmpty bool
	}{
		{
			name:                   "database error with sensitive info",
			operation:              "create",
			resourceType:           "user",
			inputError:             fmt.Errorf("executing query: CREATE USER test IDENTIFIED WITH sha256_hash BY '%s'", testHash),
			expectedUserMessage:    "Failed to create user. Please check your configuration and try again.",
			expectedTechnicalEmpty: false,
		},
		{
			name:                   "nil error",
			operation:              "update",
			resourceType:           "role",
			inputError:             nil,
			expectedUserMessage:    "",
			expectedTechnicalEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userMsg, techDetails := CreateSecureErrorMessage(tt.operation, tt.resourceType, tt.inputError)

			if userMsg != tt.expectedUserMessage {
				t.Errorf("CreateSecureErrorMessage() userMessage = %v, want %v", userMsg, tt.expectedUserMessage)
			}

			isEmpty := techDetails == ""
			if isEmpty != tt.expectedTechnicalEmpty {
				t.Errorf("CreateSecureErrorMessage() technical details empty = %v, want %v", isEmpty, tt.expectedTechnicalEmpty)
			}

			// Ensure technical details don't contain sensitive info
			if !isEmpty {
				if strings.Contains(techDetails, testHash) {
					t.Errorf("CreateSecureErrorMessage() technical details contain sensitive hash: %v", techDetails)
				}
			}
		})
	}
}

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name  string
		input error
		want  bool
	}{
		{
			name:  "connection refused",
			input: fmt.Errorf("dial tcp 127.0.0.1:9000: connection refused"),
			want:  true,
		},
		{
			name:  "connection timeout",
			input: fmt.Errorf("connection timeout after 30s"),
			want:  true,
		},
		{
			name:  "regular error",
			input: fmt.Errorf("invalid user name"),
			want:  false,
		},
		{
			name:  "nil error",
			input: nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConnectionError(tt.input)
			if result != tt.want {
				t.Errorf("IsConnectionError() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestIsAuthenticationError(t *testing.T) {
	tests := []struct {
		name  string
		input error
		want  bool
	}{
		{
			name:  "authentication failed",
			input: fmt.Errorf("authentication failed: invalid credentials"),
			want:  true,
		},
		{
			name:  "access denied",
			input: fmt.Errorf("access denied for user 'test'"),
			want:  true,
		},
		{
			name:  "regular error",
			input: fmt.Errorf("user not found"),
			want:  false,
		},
		{
			name:  "nil error",
			input: nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthenticationError(tt.input)
			if result != tt.want {
				t.Errorf("IsAuthenticationError() = %v, want %v", result, tt.want)
			}
		})
	}
}
