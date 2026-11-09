package user

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestAuthChanged(t *testing.T) {
	t.Parallel()

	// Write-only values are null in both plan and state, so the models below never set value_wo.
	sha256Hash := func(version int32, salt types.String) []Sha256HashModel {
		return []Sha256HashModel{{
			SecretMethodModel: SecretMethodModel{ValueWOVersion: types.Int32Value(version)},
			Salt:              salt,
		}}
	}
	user := func(auth *AuthModel, hostIPs ...string) User {
		u := User{
			Name:    types.StringValue("foo"),
			HostIPs: types.SetNull(types.StringType),
			Auth:    auth,
		}
		if hostIPs != nil {
			u.HostIPs, _ = types.SetValueFrom(context.Background(), types.StringType, hostIPs)
		}
		return u
	}

	tests := map[string]struct {
		state User
		plan  User
		want  bool
	}{
		"same value_wo_version": {
			state: user(&AuthModel{Sha256Hash: sha256Hash(1, types.StringNull())}),
			plan:  user(&AuthModel{Sha256Hash: sha256Hash(1, types.StringNull())}),
			want:  false,
		},
		"only host_ips changed": {
			state: user(&AuthModel{Sha256Hash: sha256Hash(1, types.StringNull())}, "127.0.0.1"),
			plan:  user(&AuthModel{Sha256Hash: sha256Hash(1, types.StringNull())}, "127.0.0.1", "10.0.0.1"),
			want:  false,
		},
		"value_wo_version bumped": {
			state: user(&AuthModel{Sha256Hash: sha256Hash(1, types.StringNull())}),
			plan:  user(&AuthModel{Sha256Hash: sha256Hash(2, types.StringNull())}),
			want:  true,
		},
		"salt changed": {
			state: user(&AuthModel{Sha256Hash: sha256Hash(1, types.StringNull())}),
			plan:  user(&AuthModel{Sha256Hash: sha256Hash(1, types.StringValue("salt"))}),
			want:  true,
		},
		"method added": {
			state: user(&AuthModel{Sha256Hash: sha256Hash(1, types.StringNull())}),
			plan: user(&AuthModel{
				Sha256Hash:     sha256Hash(1, types.StringNull()),
				SSLCertificate: []SSLCertificateModel{{CommonName: types.StringValue("cn")}},
			}),
			want: true,
		},
		"legacy password_sha256_hash changed": {
			state: User{Name: types.StringValue("foo"), HostIPs: types.SetNull(types.StringType), PasswordSha256Hash: types.StringValue("a")},
			plan:  User{Name: types.StringValue("foo"), HostIPs: types.SetNull(types.StringType), PasswordSha256Hash: types.StringValue("b")},
			want:  true,
		},
		"legacy password_sha256_hash_wo_version bumped": {
			state: User{Name: types.StringValue("foo"), HostIPs: types.SetNull(types.StringType), PasswordSha256HashVersionWO: types.Int32Value(1)},
			plan:  User{Name: types.StringValue("foo"), HostIPs: types.SetNull(types.StringType), PasswordSha256HashVersionWO: types.Int32Value(2)},
			want:  true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			schemaResponse := resource.SchemaResponse{}
			(&Resource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResponse)

			state := tfsdk.State{Schema: schemaResponse.Schema}
			diags := state.Set(ctx, test.state)
			require.False(t, diags.HasError(), diags.Errors())

			plan := tfsdk.Plan{Schema: schemaResponse.Schema}
			diags = plan.Set(ctx, test.plan)
			require.False(t, diags.HasError(), diags.Errors())

			got, diags := authChanged(ctx, plan, state)
			require.False(t, diags.HasError(), diags.Errors())
			require.Equal(t, test.want, got)
		})
	}
}
