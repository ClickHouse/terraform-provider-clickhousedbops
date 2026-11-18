package querybuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlterSettingsProfile_Build(t *testing.T) {
	tests := []struct {
		name    string
		builder AlterSettingsProfileQueryBuilder
		want    string
		wantErr bool
	}{
		{
			name:    "empty name",
			builder: NewAlterSettingsProfile(""),
			wantErr: true,
		},
		{
			name:    "rename",
			builder: NewAlterSettingsProfile("my_profile").RenameTo(new("other")),
			want:    "ALTER SETTINGS PROFILE `my_profile` RENAME TO `other`;",
		},
		{
			name:    "apply to one user",
			builder: NewAlterSettingsProfile("my_profile").ApplyTo([]string{"analyst"}),
			want:    "ALTER SETTINGS PROFILE `my_profile` TO `analyst`;",
		},
		{
			name:    "apply to a JWT user name",
			builder: NewAlterSettingsProfile("my_profile").ApplyTo([]string{"analyst", "JWT::https://idp.example.com::my-audience::jane::abc123"}),
			want:    "ALTER SETTINGS PROFILE `my_profile` TO `analyst`, `JWT::https://idp.example.com::my-audience::jane::abc123`;",
		},
		{
			name:    "apply to a name with a backtick",
			builder: NewAlterSettingsProfile("my_profile").ApplyTo([]string{"a`b"}),
			want:    "ALTER SETTINGS PROFILE `my_profile` TO `a\\`b`;",
		},
		{
			name:    "apply to nobody",
			builder: NewAlterSettingsProfile("my_profile").ApplyTo(nil),
			want:    "ALTER SETTINGS PROFILE `my_profile` TO NONE;",
		},
		{
			name:    "apply to on cluster",
			builder: NewAlterSettingsProfile("my_profile").WithCluster(new("cluster1")).ApplyTo([]string{"analyst"}),
			want:    "ALTER SETTINGS PROFILE `my_profile` ON CLUSTER 'cluster1' TO `analyst`;",
		},
		{
			name:    "inherit and apply to",
			builder: NewAlterSettingsProfile("my_profile").InheritFrom([]string{"base"}).ApplyTo([]string{"analyst"}),
			want:    "ALTER SETTINGS PROFILE `my_profile` DROP ALL PROFILES ADD PROFILES `base` TO `analyst`;",
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
