package querybuilder

import (
	"testing"
)

func Test_createuser(t *testing.T) {
	tests := []struct {
		name           string
		action         string
		resourceType   string
		resourceName   string
		identifiedWith Identification
		identifiedBy   string
		want           string
		wantErr        bool
	}{
		{
			name:         "Create user with simple name and no password",
			action:       actionCreate,
			resourceType: resourceTypeUser,
			resourceName: "john",
			want:         "CREATE USER `john`;",
			wantErr:      false,
		},
		{
			name:         "Create user with funky name and no password",
			action:       actionCreate,
			resourceType: resourceTypeUser,
			resourceName: "jo`hn",
			want:         "CREATE USER `jo\\`hn`;",
			wantErr:      false,
		},
		{
			name:           "Create user with simple name and password",
			action:         actionCreate,
			resourceType:   resourceTypeUser,
			resourceName:   "john",
			identifiedWith: IdentificationSHA256Hash,
			identifiedBy:   "blah",
			want:           "CREATE USER `john` IDENTIFIED WITH sha256_hash BY 'blah';",
			wantErr:        false,
		},
		{
			name:         "Create user fails when no user name is set",
			action:       actionCreate,
			resourceType: resourceTypeUser,
			resourceName: "",
			want:         "",
			wantErr:      true,
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
