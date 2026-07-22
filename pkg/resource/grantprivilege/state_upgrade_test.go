package grantprivilege

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestUpgradeState(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		currentGrants types.Bool
		expected      bool
	}{
		"defaults state from before current_grants to false": {
			currentGrants: types.BoolNull(),
			expected:      false,
		},
		"preserves current_grants from version 1.11 state": {
			currentGrants: types.BoolValue(true),
			expected:      true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			r := &Resource{}
			upgrader := r.UpgradeState(ctx)[0]
			require.NotNil(t, upgrader.PriorSchema)

			priorState := tfsdk.State{Schema: *upgrader.PriorSchema}
			diags := priorState.Set(ctx, GrantPrivilege{
				Privilege:       types.StringValue("SELECT"),
				GranteeRoleName: types.StringValue("reader"),
				CurrentGrants:   test.currentGrants,
			})
			require.False(t, diags.HasError(), diags.Errors())

			schemaResponse := resource.SchemaResponse{}
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
			require.Equal(t, int64(1), schemaResponse.Schema.Version)

			response := resource.UpgradeStateResponse{
				State: tfsdk.State{Schema: schemaResponse.Schema},
			}
			upgrader.StateUpgrader(ctx, resource.UpgradeStateRequest{State: &priorState}, &response)
			require.False(t, response.Diagnostics.HasError(), response.Diagnostics.Errors())

			var state GrantPrivilege
			diags = response.State.Get(ctx, &state)
			require.False(t, diags.HasError(), diags.Errors())
			require.Equal(t, test.expected, state.CurrentGrants.ValueBool())

			// Other fields must round-trip unchanged through the upgrader.
			require.Equal(t, types.StringValue("SELECT"), state.Privilege)
			require.Equal(t, types.StringValue("reader"), state.GranteeRoleName)
		})
	}
}
