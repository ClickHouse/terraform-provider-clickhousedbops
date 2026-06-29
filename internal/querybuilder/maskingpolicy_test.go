package querybuilder

import (
	"testing"
)

func i64ptr(i int64) *int64 { return &i }

func Test_createmaskingpolicy(t *testing.T) {
	tests := []struct {
		name    string
		build   func() CreateMaskingPolicyQueryBuilder
		want    string
		wantErr bool
	}{
		{
			name: "single column to a role",
			build: func() CreateMaskingPolicyQueryBuilder {
				return NewCreateMaskingPolicy("pii", "default", "build_logs_v1", []ColumnMask{
					{Column: "message", Expression: "'** redacted **'"},
				}).WithGrantees(nil, []string{"clickstate_sql_access_readonly"}, false, nil)
			},
			want: "CREATE MASKING POLICY `pii` ON `default`.`build_logs_v1` UPDATE `message` = '** redacted **' TO `clickstate_sql_access_readonly`;",
		},
		{
			name: "or replace with where and multiple columns sorted deterministically",
			build: func() CreateMaskingPolicyQueryBuilder {
				return NewCreateMaskingPolicy("pii", "logs_production", "logs_unified_v4", []ColumnMask{
					{Column: "logMessage", Expression: "'** redacted **'"},
					{Column: "clientIp", Expression: "concat(splitByChar('.', clientIp)[1], '.x.x')"},
				}).
					OrReplace(true).
					WithWhere(strptr("ownerId NOT IN ('team_a', 'team_b')")).
					WithGrantees(nil, []string{"clickstate_sql_access_readonly"}, false, nil)
			},
			// columns sorted: clientIp before logMessage
			want: "CREATE OR REPLACE MASKING POLICY `pii` ON `logs_production`.`logs_unified_v4` UPDATE `clientIp` = concat(splitByChar('.', clientIp)[1], '.x.x'), `logMessage` = '** redacted **' WHERE ownerId NOT IN ('team_a', 'team_b') TO `clickstate_sql_access_readonly`;",
		},
		{
			name: "if not exists with priority and TO ALL EXCEPT",
			build: func() CreateMaskingPolicyQueryBuilder {
				return NewCreateMaskingPolicy("pii", "default", "t", []ColumnMask{
					{Column: "c", Expression: "'x'"},
				}).
					IfNotExists(true).
					WithGrantees(nil, nil, true, []string{"admin"}).
					WithPriority(i64ptr(10))
			},
			want: "CREATE MASKING POLICY IF NOT EXISTS `pii` ON `default`.`t` UPDATE `c` = 'x' TO ALL EXCEPT `admin` PRIORITY 10;",
		},
		{
			name: "TO ALL",
			build: func() CreateMaskingPolicyQueryBuilder {
				return NewCreateMaskingPolicy("p", "d", "t", []ColumnMask{{Column: "c", Expression: "'x'"}}).
					WithGrantees(nil, nil, true, nil)
			},
			want: "CREATE MASKING POLICY `p` ON `d`.`t` UPDATE `c` = 'x' TO ALL;",
		},
		{
			name: "error: no masks",
			build: func() CreateMaskingPolicyQueryBuilder {
				return NewCreateMaskingPolicy("p", "d", "t", nil).WithGrantees(nil, []string{"r"}, false, nil)
			},
			wantErr: true,
		},
		{
			name: "error: no grantee",
			build: func() CreateMaskingPolicyQueryBuilder {
				return NewCreateMaskingPolicy("p", "d", "t", []ColumnMask{{Column: "c", Expression: "'x'"}})
			},
			wantErr: true,
		},
		{
			name: "error: empty expression",
			build: func() CreateMaskingPolicyQueryBuilder {
				return NewCreateMaskingPolicy("p", "d", "t", []ColumnMask{{Column: "c", Expression: "  "}}).
					WithGrantees(nil, []string{"r"}, false, nil)
			},
			wantErr: true,
		},
		{
			name: "error: or replace and if not exists together",
			build: func() CreateMaskingPolicyQueryBuilder {
				return NewCreateMaskingPolicy("p", "d", "t", []ColumnMask{{Column: "c", Expression: "'x'"}}).
					OrReplace(true).IfNotExists(true).WithGrantees(nil, []string{"r"}, false, nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.build().Build()
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Build()\n got = %v\nwant = %v", got, tt.want)
			}
		})
	}
}

func Test_dropmaskingpolicy(t *testing.T) {
	tests := []struct {
		name    string
		build   func() DropMaskingPolicyQueryBuilder
		want    string
		wantErr bool
	}{
		{
			name:  "drop",
			build: func() DropMaskingPolicyQueryBuilder { return NewDropMaskingPolicy("pii", "default", "t") },
			want:  "DROP MASKING POLICY `pii` ON `default`.`t`;",
		},
		{
			name: "drop if exists",
			build: func() DropMaskingPolicyQueryBuilder {
				return NewDropMaskingPolicy("pii", "default", "t").IfExists(true)
			},
			want: "DROP MASKING POLICY IF EXISTS `pii` ON `default`.`t`;",
		},
		{
			name:    "error: empty name",
			build:   func() DropMaskingPolicyQueryBuilder { return NewDropMaskingPolicy("", "default", "t") },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.build().Build()
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Build()\n got = %v\nwant = %v", got, tt.want)
			}
		})
	}
}
