package querybuilder

import (
	"testing"
)

func Test_createSettingsProfileQueryBuilder_Build(t *testing.T) {
	tests := []struct {
		name        string
		profileName string
		clusterName *string
		inherit     *string
		settings    []setting
		want        string
		wantErr     bool
	}{
		{
			name:        "Single setting",
			profileName: "prf1",
			clusterName: nil,
			settings:    []setting{newSettingMock("mock_rendered_setting")},
			want:        "CREATE SETTINGS PROFILE `prf1` SETTINGS mock_rendered_setting;",
			wantErr:     false,
		},
		{
			name:        "Multiple settings",
			profileName: "prf1",
			clusterName: nil,
			settings:    []setting{newSettingMock("mock_rendered_setting1"), newSettingMock("mock_rendered_setting2")},
			want:        "CREATE SETTINGS PROFILE `prf1` SETTINGS mock_rendered_setting1, mock_rendered_setting2;",
			wantErr:     false,
		},
		{
			name:        "No settings",
			profileName: "prf1",
			clusterName: nil,
			settings:    nil,
			want:        "",
			wantErr:     true,
		},
		{
			name:        "on cluster",
			profileName: "prf1",
			clusterName: strPtr("cluster1"),
			settings:    []setting{newSettingMock("mock_rendered_setting")},
			want:        "CREATE SETTINGS PROFILE `prf1` ON CLUSTER 'cluster1' SETTINGS mock_rendered_setting;",
			wantErr:     false,
		},
		{
			name:        "Inherit",
			profileName: "prf1",
			clusterName: nil,
			inherit:     strPtr("default"),
			settings:    []setting{newSettingMock("mock_rendered_setting")},
			want:        "CREATE SETTINGS PROFILE `prf1` SETTINGS mock_rendered_setting INHERIT 'default';",
			wantErr:     false,
		},
		{
			name:        "Inherit with quote",
			profileName: "prf1",
			clusterName: nil,
			inherit:     strPtr("def'ault"),
			settings:    []setting{newSettingMock("mock_rendered_setting")},
			want:        "CREATE SETTINGS PROFILE `prf1` SETTINGS mock_rendered_setting INHERIT 'def\\'ault';",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &createSettingsProfileQueryBuilder{
				profileName:    tt.profileName,
				clusterName:    tt.clusterName,
				inheritProfile: tt.inherit,
				settings:       tt.settings,
			}
			got, err := q.Build()
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Build() got = %v, want %v", got, tt.want)
			}
		})
	}
}

type settingsMock struct {
	rendered string
}

func newSettingMock(rendered string) setting {
	return &settingsMock{rendered: rendered}
}

func (s *settingsMock) SQLDef() (string, error) {
	return s.rendered, nil
}
