package querybuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateRowPolicy_Build(t *testing.T) {
	tests := []struct {
		name    string
		builder *CreateRowPolicy
		want    string
		wantErr bool
	}{
		{
			name: "permissive to role",
			builder: NewCreateRowPolicy("reader_rows", "default", "tbl1").
				SelectFilter("1 = 1").
				GranteeNames([]string{"reader"}),
			want: "CREATE ROW POLICY `reader_rows` ON `default`.`tbl1` USING 1 = 1 AS PERMISSIVE TO `reader`",
		},
		{
			name: "restrictive to user with cluster",
			builder: NewCreateRowPolicy("john_rows", "default", "tbl1").
				WithCluster(stringPtr("cluster1")).
				SelectFilter("owner_id = 'john'").
				IsRestrictive(true).
				GranteeNames([]string{"john"}),
			want: "CREATE ROW POLICY `john_rows` ON CLUSTER `cluster1` ON `default`.`tbl1` USING owner_id = 'john' AS RESTRICTIVE TO `john`",
		},
		{
			name: "to all",
			builder: NewCreateRowPolicy("p", "default", "t").
				SelectFilter("1").
				GranteeAll(true),
			want: "CREATE ROW POLICY `p` ON `default`.`t` USING 1 AS PERMISSIVE TO ALL",
		},
		{
			name: "to all except",
			builder: NewCreateRowPolicy("p", "default", "t").
				SelectFilter("1").
				GranteeAllExcept([]string{"admin"}),
			want: "CREATE ROW POLICY `p` ON `default`.`t` USING 1 AS PERMISSIVE TO ALL EXCEPT `admin`",
		},
		{
			name:    "error: no grantee",
			builder: NewCreateRowPolicy("p", "default", "t").SelectFilter("1"),
			wantErr: true,
		},
		{
			name:    "error: empty name",
			builder: NewCreateRowPolicy("", "default", "t").SelectFilter("1").GranteeAll(true),
			wantErr: true,
		},
		{
			name:    "error: empty filter",
			builder: NewCreateRowPolicy("p", "default", "t").GranteeAll(true),
			wantErr: true,
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

func TestDropRowPolicy_Build(t *testing.T) {
	tests := []struct {
		name    string
		builder *DropRowPolicy
		want    string
		wantErr bool
	}{
		{
			name:    "drop if exists",
			builder: NewDropRowPolicy("p", "default", "t").IfExists(true),
			want:    "DROP ROW POLICY IF EXISTS `p` ON `default`.`t`",
		},
		{
			name:    "drop if exists with cluster",
			builder: NewDropRowPolicy("p", "default", "t").WithCluster(stringPtr("cluster1")).IfExists(true),
			want:    "DROP ROW POLICY IF EXISTS `p` ON CLUSTER `cluster1` ON `default`.`t`",
		},
		{
			name:    "drop without if exists",
			builder: NewDropRowPolicy("p", "default", "t"),
			want:    "DROP ROW POLICY `p` ON `default`.`t`",
		},
		{
			name:    "error: empty name",
			builder: NewDropRowPolicy("", "default", "t"),
			wantErr: true,
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
