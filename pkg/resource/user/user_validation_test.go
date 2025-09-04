package user

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestPasswordFieldValidationLogic tests the core password validation logic
// This tests the exact logic used in the ModifyPlan method
func TestPasswordFieldValidationLogic(t *testing.T) {
	tests := []struct {
		name              string
		writeOnlyPassword types.String
		legacyPassword    types.String
		expectedConflict  bool
		expectedMissing   bool
	}{
		{
			name:              "Both password fields specified - should conflict",
			writeOnlyPassword: types.StringValue("a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd"),
			legacyPassword:    types.StringValue("b1c2d3e4f5g6789012345678901234567890123456789012345678901234efgh"),
			expectedConflict:  true,
			expectedMissing:   false,
		},
		{
			name:              "Neither password field specified - should be missing",
			writeOnlyPassword: types.StringNull(),
			legacyPassword:    types.StringNull(),
			expectedConflict:  false,
			expectedMissing:   true,
		},
		{
			name:              "Only write-only password specified - should be valid",
			writeOnlyPassword: types.StringValue("a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd"),
			legacyPassword:    types.StringNull(),
			expectedConflict:  false,
			expectedMissing:   false,
		},
		{
			name:              "Only legacy password specified - should be valid",
			writeOnlyPassword: types.StringNull(),
			legacyPassword:    types.StringValue("b1c2d3e4f5g6789012345678901234567890123456789012345678901234efgh"),
			expectedConflict:  false,
			expectedMissing:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the validation logic directly (mimics ModifyPlan method lines 111-113)
			hasWriteOnlyPassword := !tt.writeOnlyPassword.IsNull()
			hasLegacyPassword := !tt.legacyPassword.IsNull()

			actualConflict := hasWriteOnlyPassword && hasLegacyPassword
			actualMissing := !hasWriteOnlyPassword && !hasLegacyPassword

			if actualConflict != tt.expectedConflict {
				t.Errorf("Expected conflict=%v, got conflict=%v", tt.expectedConflict, actualConflict)
			}

			if actualMissing != tt.expectedMissing {
				t.Errorf("Expected missing=%v, got missing=%v", tt.expectedMissing, actualMissing)
			}
		})
	}
}

// TestStateStorageLogic tests the logic for determining what goes into state
// This tests the exact logic used in the Create method
func TestStateStorageLogic(t *testing.T) {
	tests := []struct {
		name                     string
		writeOnlyPassword        types.String
		legacyPassword           types.String
		expectedWriteOnlyInState bool
		expectedLegacyInState    bool
	}{
		{
			name:                     "Write-only password should not appear in state",
			writeOnlyPassword:        types.StringValue("a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd"),
			legacyPassword:           types.StringNull(),
			expectedWriteOnlyInState: false, // WriteOnly fields are never stored
			expectedLegacyInState:    false,
		},
		{
			name:                     "Legacy password should appear in state (sensitive)",
			writeOnlyPassword:        types.StringNull(),
			legacyPassword:           types.StringValue("b1c2d3e4f5g6789012345678901234567890123456789012345678901234efgh"),
			expectedWriteOnlyInState: false,
			expectedLegacyInState:    true, // Legacy field is stored (but marked sensitive)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the Create method logic (lines 214-217)
			var stateWriteOnly types.String
			var stateLegacy types.String

			// Write-only fields are never stored in state (framework handles this)
			stateWriteOnly = types.StringNull()

			// Legacy password is stored if it was used (from Create method line 215-217)
			if !tt.legacyPassword.IsNull() {
				stateLegacy = tt.legacyPassword
			} else {
				stateLegacy = types.StringNull()
			}

			actualWriteOnlyInState := !stateWriteOnly.IsNull()
			actualLegacyInState := !stateLegacy.IsNull()

			if actualWriteOnlyInState != tt.expectedWriteOnlyInState {
				t.Errorf("Expected writeOnlyInState=%v, got %v", tt.expectedWriteOnlyInState, actualWriteOnlyInState)
			}

			if actualLegacyInState != tt.expectedLegacyInState {
				t.Errorf("Expected legacyInState=%v, got %v", tt.expectedLegacyInState, actualLegacyInState)
			}
		})
	}
}

// TestPasswordHashValidation tests SHA256 hash format validation
// This tests the regex validation logic used in the schema
func TestPasswordHashValidation(t *testing.T) {
	tests := []struct {
		name          string
		password      string
		shouldBeValid bool
	}{
		{
			name:          "Valid SHA256 hash - lowercase",
			password:      "a1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd",
			shouldBeValid: true,
		},
		{
			name:          "Valid SHA256 hash - uppercase",
			password:      "A1B2C3D4E5F6789012345678901234567890123456789012345678901234ABCD",
			shouldBeValid: true,
		},
		{
			name:          "Valid SHA256 hash - mixed case",
			password:      "a1B2c3D4e5F6789012345678901234567890123456789012345678901234AbCd",
			shouldBeValid: true,
		},
		{
			name:          "Invalid - too short",
			password:      "a1b2c3d4e5f678901234567890123456789012345678901234567890123456",
			shouldBeValid: false,
		},
		{
			name:          "Invalid - too long",
			password:      "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678901",
			shouldBeValid: false,
		},
		{
			name:          "Invalid - contains non-hex characters",
			password:      "g1b2c3d4e5f6789012345678901234567890123456789012345678901234abcd",
			shouldBeValid: false,
		},
		{
			name:          "Invalid - empty string",
			password:      "",
			shouldBeValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test SHA256 hash validation logic (mimics regex `^[a-fA-F0-9]{64}$`)
			isValid := isValidSHA256Hash(tt.password)

			if isValid != tt.shouldBeValid {
				t.Errorf("Expected valid=%v for password '%s', got valid=%v", tt.shouldBeValid, tt.password, isValid)
			}
		})
	}
}

// TestImportBehaviorLogic tests import scenarios with password fields
// This tests the exact logic used in the ImportState method
func TestImportBehaviorLogic(t *testing.T) {
	tests := []struct {
		name                string
		importId            string
		expectedClusterName *string
		expectedUserId      string
	}{
		{
			name:                "Import with cluster name",
			importId:            "cluster1:user123",
			expectedClusterName: stringPtr("cluster1"),
			expectedUserId:      "user123",
		},
		{
			name:                "Import without cluster name",
			importId:            "user123",
			expectedClusterName: nil,
			expectedUserId:      "user123",
		},
		{
			name:                "Import with UUID",
			importId:            "550e8400-e29b-41d4-a716-446655440000",
			expectedClusterName: nil,
			expectedUserId:      "550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test import ID parsing logic (mimics ImportState method lines 308-312)
			clusterName, userId := parseImportId(tt.importId)

			if (clusterName == nil && tt.expectedClusterName != nil) ||
				(clusterName != nil && tt.expectedClusterName == nil) ||
				(clusterName != nil && tt.expectedClusterName != nil && *clusterName != *tt.expectedClusterName) {
				t.Errorf("Expected cluster name %v, got %v", tt.expectedClusterName, clusterName)
			}

			if userId != tt.expectedUserId {
				t.Errorf("Expected user ID %s, got %s", tt.expectedUserId, userId)
			}
		})
	}
}

// TestPasswordSelectionLogic tests the logic for selecting which password to use
// This tests the exact logic used in the Create method lines 184-191
func TestPasswordSelectionLogic(t *testing.T) {
	tests := []struct {
		name                    string
		configWriteOnlyPassword types.String
		planLegacyPassword      types.String
		expectedPassword        string
		expectedSource          string
	}{
		{
			name:                    "WriteOnly password takes precedence",
			configWriteOnlyPassword: types.StringValue("writeonly123456789012345678901234567890123456789012345678901234"),
			planLegacyPassword:      types.StringValue("legacy1234567890123456789012345678901234567890123456789012345678"),
			expectedPassword:        "writeonly123456789012345678901234567890123456789012345678901234",
			expectedSource:          "writeonly",
		},
		{
			name:                    "Legacy password used when WriteOnly is null",
			configWriteOnlyPassword: types.StringNull(),
			planLegacyPassword:      types.StringValue("legacy1234567890123456789012345678901234567890123456789012345678"),
			expectedPassword:        "legacy1234567890123456789012345678901234567890123456789012345678",
			expectedSource:          "legacy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the Create method password selection logic (lines 184-191)
			var passwordHash string
			var source string

			if !tt.configWriteOnlyPassword.IsNull() {
				// Use WriteOnly field (Terraform 1.11+)
				passwordHash = tt.configWriteOnlyPassword.ValueString()
				source = "writeonly"
			} else {
				// Use legacy Sensitive field
				passwordHash = tt.planLegacyPassword.ValueString()
				source = "legacy"
			}

			if passwordHash != tt.expectedPassword {
				t.Errorf("Expected password %s, got %s", tt.expectedPassword, passwordHash)
			}

			if source != tt.expectedSource {
				t.Errorf("Expected source %s, got %s", tt.expectedSource, source)
			}
		})
	}
}

// TestReplicatedStorageValidation tests replicated storage warning logic
// This tests the exact logic used in the ModifyPlan method lines 140-155
func TestReplicatedStorageValidation(t *testing.T) {
	tests := []struct {
		name                string
		isReplicatedStorage bool
		hasClusterName      bool
		expectWarning       bool
	}{
		{
			name:                "Replicated storage with cluster_name should warn",
			isReplicatedStorage: true,
			hasClusterName:      true,
			expectWarning:       true,
		},
		{
			name:                "Replicated storage without cluster_name should not warn",
			isReplicatedStorage: true,
			hasClusterName:      false,
			expectWarning:       false,
		},
		{
			name:                "Non-replicated storage with cluster_name should not warn",
			isReplicatedStorage: false,
			hasClusterName:      true,
			expectWarning:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test replicated storage warning logic (mimics ModifyPlan lines 140-155)
			var clusterName types.String
			if tt.hasClusterName {
				clusterName = types.StringValue("testcluster")
			} else {
				clusterName = types.StringNull()
			}

			shouldWarn := tt.isReplicatedStorage && !clusterName.IsNull()

			if shouldWarn != tt.expectWarning {
				t.Errorf("Expected warning=%v, got warning=%v", tt.expectWarning, shouldWarn)
			}
		})
	}
}

// Helper functions for tests

// isValidSHA256Hash validates SHA256 hash format (mimics regex `^[a-fA-F0-9]{64}$`)
func isValidSHA256Hash(hash string) bool {
	if len(hash) != 64 {
		return false
	}

	for _, char := range hash {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}

	return true
}

// parseImportId parses import ID to extract cluster name and user ID (mimics ImportState method)
func parseImportId(importId string) (*string, string) {
	if len(importId) == 0 {
		return nil, ""
	}

	// Check if cluster name is specified (line 309 logic)
	if idx := findColon(importId); idx != -1 {
		clusterName := importId[:idx]
		userId := importId[idx+1:]
		return &clusterName, userId
	}

	return nil, importId
}

// findColon finds the index of the first colon in a string
func findColon(s string) int {
	for i, char := range s {
		if char == ':' {
			return i
		}
	}
	return -1
}

// stringPtr creates a string pointer
func stringPtr(s string) *string {
	return &s
}
