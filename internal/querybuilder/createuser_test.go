package querybuilder

import (
	"testing"
)

func Test_createuser(t *testing.T) {
	tests := []struct {
		name            string
		action          string
		resourceName    string
		identifiedWith  Identification
		identifiedBy    string
		settingsProfile string
		want            string
		wantErr         bool
	}{
		{
			name:         "Create user with simple name and no password",
			resourceName: "john",
			want:         "CREATE USER `john`;",
			wantErr:      false,
		},
		{
			name:         "Create user with funky name and no password",
			resourceName: "jo`hn",
			want:         "CREATE USER `jo\\`hn`;",
			wantErr:      false,
		},
		{
			name:           "Create user with simple name and password",
			resourceName:   "john",
			identifiedWith: IdentificationSHA256Hash,
			identifiedBy:   "blah",
			want:           "CREATE USER `john` IDENTIFIED WITH sha256_hash BY 'blah';",
			wantErr:        false,
		},
		{
			name:         "Create user fails when no user name is set",
			resourceName: "",
			want:         "",
			wantErr:      true,
		},
		{
			name:            "Create user with settings profile",
			resourceName:    "foo",
			settingsProfile: "test",
			want:            "CREATE USER `foo` SETTINGS PROFILE 'test';",
			wantErr:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q CreateUserQueryBuilder
			q = &createUserQueryBuilder{
				resourceName: tt.resourceName,
			}

			if tt.identifiedWith != "" && tt.identifiedBy != "" {
				q = q.Identified(tt.identifiedWith, tt.identifiedBy)
			}

			if tt.settingsProfile != "" {
				q = q.WithSettingsProfile(&tt.settingsProfile)
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
