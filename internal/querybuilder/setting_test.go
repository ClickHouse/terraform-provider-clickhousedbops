package querybuilder

import (
	"testing"
)

func Test_setting_SQLDef(t *testing.T) {
	tests := []struct {
		name    string
		setting setting
		want    string
		wantErr bool
	}{
		{
			name: "Empty name",
			setting: &settingData{
				Name: "",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "No values",
			setting: &settingData{
				Name: "test",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Only value",
			setting: &settingData{
				Name:  "test",
				Value: strPtr("123"),
			},
			want:    "`test` = '123'",
			wantErr: false,
		},
		{
			name: "Only min",
			setting: &settingData{
				Name: "test",
				Min:  strPtr("456"),
			},
			want:    "`test` MIN '456'",
			wantErr: false,
		},
		{
			name: "Only max",
			setting: &settingData{
				Name: "test",
				Max:  strPtr("789"),
			},
			want:    "`test` MAX '789'",
			wantErr: false,
		},
		{
			name: "Only min and max",
			setting: &settingData{
				Name: "test",
				Min:  strPtr("10"),
				Max:  strPtr("100"),
			},
			want:    "`test` MIN '10' MAX '100'",
			wantErr: false,
		},
		{
			name: "Value, min and max",
			setting: &settingData{
				Name:  "test",
				Value: strPtr("50"),
				Min:   strPtr("10"),
				Max:   strPtr("100"),
			},
			want:    "`test` = '50' MIN '10' MAX '100'",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.setting.SQLDef()
			if (err != nil) != tt.wantErr {
				t.Errorf("SQLDef() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SQLDef() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func strPtr(val string) *string {
	return &val
}
