package querybuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlterRowPolicy_Basic(t *testing.T) {
	tests := []struct {
		name    string
		builder *AlterRowPolicy
		want    string
		wantErr bool
	}{
		{
			name: "alter row policy select filter only",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				SelectFilter("user_id = 'alice'"),
			want: "ALTER ROW POLICY `my_policy` ON `default`.`users` USING user_id = 'alice'",
		},
		{
			name: "alter row policy is_restrictive only",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				IsRestrictive(true),
			want: "ALTER ROW POLICY `my_policy` ON `default`.`users` AS RESTRICTIVE",
		},
		{
			name: "alter row policy both select filter and is_restrictive",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				SelectFilter("1").
				IsRestrictive(false),
			want: "ALTER ROW POLICY `my_policy` ON `default`.`users` AS PERMISSIVE USING 1",
		},
		{
			name: "alter row policy with cluster",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				WithCluster(stringPtr("cluster1")).
				SelectFilter("tenant_id = 'abc'"),
			want: "ALTER ROW POLICY `my_policy` ON CLUSTER `cluster1` ON `default`.`users` USING tenant_id = 'abc'",
		},
		{
			name:    "alter row policy no changes",
			builder: NewAlterRowPolicy("my_policy", "default", "users"),
			wantErr: true,
		},
		{
			name: "alter row policy grantee user names",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				GranteeUserNames([]string{"alice", "bob"}),
			want: "ALTER ROW POLICY `my_policy` ON `default`.`users` TO `alice`, `bob`",
		},
		{
			name: "alter row policy grantee role names",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				GranteeRoleNames([]string{"admin", "editor"}),
			want: "ALTER ROW POLICY `my_policy` ON `default`.`users` TO `admin`, `editor`",
		},
		{
			name: "alter row policy grantee all",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				GranteeAll(true),
			want: "ALTER ROW POLICY `my_policy` ON `default`.`users` TO ALL",
		},
		{
			name: "alter row policy grantee all except",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				GranteeAll(true).
				GranteeAllExcept([]string{"readonly"}),
			want: "ALTER ROW POLICY `my_policy` ON `default`.`users` TO ALL EXCEPT `readonly`",
		},
		{
			name: "alter row policy with select filter and grantee",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				SelectFilter("user_id = 'alice'").
				GranteeUserNames([]string{"alice"}),
			want: "ALTER ROW POLICY `my_policy` ON `default`.`users` USING user_id = 'alice' TO `alice`",
		},
		{
			name: "alter row policy for operations only",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				ForOperations([]string{"SELECT"}),
			want: "ALTER ROW POLICY `my_policy` ON `default`.`users` FOR SELECT",
		},
		{
			name: "alter row policy for operations with using",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				ForOperations([]string{"SELECT"}).
				SelectFilter("user_id = 'bob'"),
			want: "ALTER ROW POLICY `my_policy` ON `default`.`users` FOR SELECT USING user_id = 'bob'",
		},
		{
			name: "alter row policy for operations with as clause",
			builder: NewAlterRowPolicy("my_policy", "default", "users").
				ForOperations([]string{"SELECT"}).
				IsRestrictive(true),
			want: "ALTER ROW POLICY `my_policy` ON `default`.`users` FOR SELECT AS RESTRICTIVE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.builder.Build()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
