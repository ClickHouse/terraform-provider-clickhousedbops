package errors

import (
	"fmt"
	"strings"
	"testing"
)

// Integration tests for error sanitization in realistic scenarios
func TestSanitizeError_IntegrationScenarios(t *testing.T) {
	tests := []struct {
		name             string
		simulatedError   error
		category         ErrorCategory
		shouldNotContain []string
		shouldContain    []string
	}{
		{
			name:           "CREATE USER SQL error with password hash",
			simulatedError: fmt.Errorf("error running query: Code: 516, e.message = Unknown user, executing query: CREATE USER test IDENTIFIED WITH sha256_hash BY '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08'"),
			category:       CategoryDatabase,
			shouldNotContain: []string{
				"9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", // Original hash
				"CREATE USER test",                                                     // SQL structure
			},
			shouldContain: []string{
				"operation failed",
				"[SQL QUERY]",
			},
		},
		{
			name:           "Connection string with password leak",
			simulatedError: fmt.Errorf("error building query: failed to connect clickhouse://admin:supersecretpassword@localhost:9000/users"),
			category:       CategoryDatabase,
			shouldNotContain: []string{
				"supersecretpassword", // Password should be sanitized
			},
			shouldContain: []string{
				"operation failed",
				"clickhouse://admin:[REDACTED]@", // Sanitized connection string
			},
		},
		{
			name:           "Stack trace with system paths",
			simulatedError: fmt.Errorf("runtime error occurred\n    at /usr/local/clickhouse/bin/server:1234\n    at /home/admin/terraform/main.go:567"),
			category:       CategoryGeneral,
			shouldNotContain: []string{
				"/usr/local/clickhouse/bin/server:1234", // System path
				"/home/admin/terraform/main.go:567",     // User path
			},
			shouldContain: []string{
				"operation failed",
				"runtime error occurred", // Original error preserved
			},
		},
		{
			name:           "Authentication error should remain informative",
			simulatedError: fmt.Errorf("authentication failed: access denied for user 'terraform_user'"),
			category:       CategoryAuthentication,
			shouldNotContain: []string{
				// No sensitive info to sanitize here
			},
			shouldContain: []string{
				"operation failed",
				"authentication failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.simulatedError, tt.category)
			resultStr := result.Error()

			// Check that sensitive information is removed
			for _, forbidden := range tt.shouldNotContain {
				if strings.Contains(resultStr, forbidden) {
					t.Errorf("SanitizeError() result contains sensitive information: %s\nResult: %s", forbidden, resultStr)
				}
			}

			// Check that important information is preserved
			for _, required := range tt.shouldContain {
				if !strings.Contains(resultStr, required) {
					t.Errorf("SanitizeError() result missing required information: %s\nResult: %s", required, resultStr)
				}
			}
		})
	}
}

func TestCreateSecureErrorMessage_UserFriendliness(t *testing.T) {
	tests := []struct {
		name         string
		operation    string
		resourceType string
		inputError   error
		wantUserFriendly bool
	}{
		{
			name:         "Database error should be user-friendly",
			operation:    "create",
			resourceType: "user",
			inputError:   fmt.Errorf("executing query: CREATE USER test IDENTIFIED WITH sha256_hash BY '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08'"),
			wantUserFriendly: true,
		},
		{
			name:         "Complex connection error should be user-friendly",
			operation:    "connect",
			resourceType: "database",
			inputError:   fmt.Errorf("dial tcp 127.0.0.1:9000: connection refused after 30 seconds with ssl handshake failure"),
			wantUserFriendly: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userMsg, techDetails := CreateSecureErrorMessage(tt.operation, tt.resourceType, tt.inputError)

			// Check user message is friendly and actionable
			if !strings.Contains(userMsg, "Failed to") {
				t.Errorf("CreateSecureErrorMessage() user message not user-friendly: %s", userMsg)
			}
			if !strings.Contains(userMsg, "Please check") {
				t.Errorf("CreateSecureErrorMessage() user message lacks actionable guidance: %s", userMsg)
			}

			// Check technical details are sanitized
			if strings.Contains(techDetails, "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08") {
				t.Errorf("CreateSecureErrorMessage() technical details contain sensitive hash: %s", techDetails)
			}

			// Ensure both messages are different (user-friendly vs technical)
			if userMsg == techDetails && techDetails != "" {
				t.Errorf("CreateSecureErrorMessage() user and technical messages should be different")
			}
		})
	}
}

func TestErrorCategorization_RealisticScenarios(t *testing.T) {
	tests := []struct {
		name        string
		error       error
		expectConnection bool
		expectAuth  bool
	}{
		{
			name:             "ClickHouse connection refused",
			error:            fmt.Errorf("dial tcp 127.0.0.1:9000: connection refused"),
			expectConnection: true,
			expectAuth:       false,
		},
		{
			name:             "Authentication failure",
			error:            fmt.Errorf("Code: 516, e.message = Authentication failed"),
			expectConnection: false,
			expectAuth:       true,
		},
		{
			name:             "Network timeout",
			error:            fmt.Errorf("network error: connection timeout after 30s"),
			expectConnection: true,
			expectAuth:       false,
		},
		{
			name:             "Permission denied",
			error:            fmt.Errorf("access denied for user 'readonly_user' to database 'admin'"),
			expectConnection: false,
			expectAuth:       true,
		},
		{
			name:             "Generic database error",
			error:            fmt.Errorf("table does not exist"),
			expectConnection: false,
			expectAuth:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isConnection := IsConnectionError(tt.error)
			isAuth := IsAuthenticationError(tt.error)

			if isConnection != tt.expectConnection {
				t.Errorf("IsConnectionError() = %v, want %v for error: %s", isConnection, tt.expectConnection, tt.error)
			}

			if isAuth != tt.expectAuth {
				t.Errorf("IsAuthenticationError() = %v, want %v for error: %s", isAuth, tt.expectAuth, tt.error)
			}
		})
	}
}