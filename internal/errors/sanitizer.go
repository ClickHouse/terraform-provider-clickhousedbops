package errors

import (
	"fmt"
	"regexp"
	"strings"
)

// SensitivePatterns contains regex patterns for sensitive information that should be sanitized
var SensitivePatterns = []*regexp.Regexp{
	// Password hashes in SQL queries - fixed pattern
	regexp.MustCompile(`(?i)(IDENTIFIED\s+WITH\s+\w+\s+BY\s+['"])([a-fA-F0-9]{64})(['"])`),
	// Generic password patterns
	regexp.MustCompile(`(?i)(password\s*[:=]\s*['"])([^'"]*?)(['"])`),
	// SHA256 hash patterns (64 hex characters) - standalone
	regexp.MustCompile(`(?i)\b([a-fA-F0-9]{64})\b`),
	// Connection strings with credentials
	regexp.MustCompile(`(?i)((?:mysql|postgres|clickhouse)://[^:]+:)([^@]+)(@)`),
	// Basic auth in URLs
	regexp.MustCompile(`(?i)(https?://[^:]+:)([^@]+)(@)`),
	// API keys and tokens (common patterns)
	regexp.MustCompile(`(?i)((?:api[_-]?key|token|secret)['":\s=]+['"])([^'"]{8,})(['"])`),
}

// DatabaseErrorPatterns contains patterns for database-specific errors that need sanitization
var DatabaseErrorPatterns = []*regexp.Regexp{
	// ClickHouse specific error patterns that might contain sensitive info
	regexp.MustCompile(`(?i)(Code: \d+[^,]*,\s*e\.message\s*=\s*)([^,]+)(,.*)`),
	regexp.MustCompile(`(?i)(Exception:\s*)([^,\n]+)(\s*\(version.*)`),
}

// ErrorCategory represents the type of error for contextual sanitization
type ErrorCategory int

const (
	// CategoryGeneral for general errors
	CategoryGeneral ErrorCategory = iota
	// CategoryDatabase for database-related errors
	CategoryDatabase
	// CategoryAuthentication for auth-related errors
	CategoryAuthentication
	// CategoryValidation for validation errors
	CategoryValidation
)

// SanitizeError removes sensitive information from error messages while preserving useful debugging context
func SanitizeError(err error, category ErrorCategory) error {
	if err == nil {
		return nil
	}

	message := err.Error()
	sanitized := sanitizeString(message, category)

	// Return a new error with the sanitized message
	return fmt.Errorf("operation failed: %s", sanitized)
}

// sanitizeString sanitizes a string by removing sensitive patterns
func sanitizeString(input string, category ErrorCategory) string {
	result := input

	// Apply general sensitive pattern sanitization
	for _, pattern := range SensitivePatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			submatches := pattern.FindStringSubmatch(match)
			if len(submatches) >= 3 {
				// Keep the structure but replace sensitive content
				if len(submatches) == 4 {
					// Three capture groups: prefix, sensitive content, suffix
					return submatches[1] + "[REDACTED]" + submatches[3]
				} else if len(submatches) == 2 {
					// One capture group: just the sensitive content
					return "[REDACTED]"
				}
			}
			return "[REDACTED]"
		})
	}

	// Apply database-specific sanitization if needed
	if category == CategoryDatabase {
		for _, pattern := range DatabaseErrorPatterns {
			result = pattern.ReplaceAllStringFunc(result, func(match string) string {
				submatches := pattern.FindStringSubmatch(match)
				if len(submatches) >= 3 {
					// Keep error code/prefix but sanitize the message
					return submatches[1] + "[DATABASE ERROR]" + submatches[len(submatches)-1]
				}
				return "[DATABASE ERROR]"
			})
		}
	}

	// Additional sanitization for known ClickHouse error patterns
	result = sanitizeClickHouseErrors(result)

	return result
}

// sanitizeClickHouseErrors handles ClickHouse-specific error message patterns
func sanitizeClickHouseErrors(message string) string {
	result := message

	// Remove stack traces that might contain sensitive file paths or internal details
	stackTracePattern := regexp.MustCompile(`(?s)\n\s*at\s+.*?(\n|$)`)
	result = stackTracePattern.ReplaceAllString(result, "")

	// Sanitize SQL query fragments
	sqlPattern := regexp.MustCompile(`(?i)(executing query:\s*)([^;]+;?)`)
	result = sqlPattern.ReplaceAllStringFunc(result, func(match string) string {
		return "executing query: [SQL QUERY]"
	})

	// Remove specific file paths and line numbers that could reveal system architecture
	pathPattern := regexp.MustCompile(`(/[^\s:]+:\d+)`)
	result = pathPattern.ReplaceAllString(result, "[FILE:LINE]")

	// Clean up common error prefixes to provide consistent messaging
	cleanupPatterns := map[string]string{
		"error building query: ": "",
		"error running query: ":  "",
		"error getting ":         "unable to retrieve ",
		"error creating ":        "unable to create ",
		"error updating ":        "unable to update ",
		"error deleting ":        "unable to delete ",
	}

	for old, new := range cleanupPatterns {
		result = strings.ReplaceAll(result, old, new)
	}

	return result
}

// CreateSecureErrorMessage creates a user-friendly error message while optionally preserving
// technical details for internal logging
func CreateSecureErrorMessage(operation, resourceType string, err error) (userMessage string, technicalDetails string) {
	if err == nil {
		return "", ""
	}

	// Create a generic user-friendly message
	userMessage = fmt.Sprintf("Failed to %s %s. Please check your configuration and try again.", operation, resourceType)

	// Preserve technical details (sanitized) for internal logging
	technicalDetails = sanitizeString(err.Error(), CategoryDatabase)

	return userMessage, technicalDetails
}

// IsConnectionError checks if the error is related to database connectivity issues
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	connectionKeywords := []string{
		"connection refused",
		"connection timeout",
		"connection failed",
		"network error",
		"dial tcp",
		"no such host",
		"connection reset",
	}

	for _, keyword := range connectionKeywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}

	return false
}

// IsAuthenticationError checks if the error is related to authentication issues
func IsAuthenticationError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	authKeywords := []string{
		"authentication failed",
		"access denied",
		"unauthorized",
		"invalid credentials",
		"login failed",
		"permission denied",
	}

	for _, keyword := range authKeywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}

	return false
}
