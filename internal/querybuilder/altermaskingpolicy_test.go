package querybuilder

import (
	"testing"
)

func Test_altermaskingpolicy(t *testing.T) {
	tests := []struct {
		name    string
		build   func() AlterMaskingPolicyQueryBuilder
		want    string
		wantErr bool
	}{
		{
			name: "full definition with where and priority",
			build: func() AlterMaskingPolicyQueryBuilder {
				return NewAlterMaskingPolicy("pii", "logs_production", "logs_unified_v4", []ColumnMask{
					{Column: "logMessage", Expression: "'** redacted **'"},
					{Column: "clientIp", Expression: "concat(splitByChar('.', clientIp)[1], '.x.x')"},
				}).
					WithWhere(strptr("ownerId NOT IN ('team_a', 'team_b')")).
					GranteeNames([]string{"analyst"}).
					WithPriority(i64ptr(5))
			},
			// columns sorted: clientIp before logMessage
			want: "ALTER MASKING POLICY `pii` ON `logs_production`.`logs_unified_v4` UPDATE `clientIp` = concat(splitByChar('.', clientIp)[1], '.x.x'), `logMessage` = '** redacted **' WHERE ownerId NOT IN ('team_a', 'team_b') TO `analyst` PRIORITY 5;",
		},
		{
			name: "rename plus full definition in one statement",
			build: func() AlterMaskingPolicyQueryBuilder {
				return NewAlterMaskingPolicy("old", "d", "t", []ColumnMask{{Column: "c", Expression: "'x'"}}).
					RenameTo("new").
					GranteeAll(true).
					WithPriority(i64ptr(3))
			},
			want: "ALTER MASKING POLICY `old` ON `d`.`t` RENAME TO `new` UPDATE `c` = 'x' TO ALL PRIORITY 3;",
		},
		{
			name: "rename omitted when unchanged",
			build: func() AlterMaskingPolicyQueryBuilder {
				return NewAlterMaskingPolicy("p", "d", "t", []ColumnMask{{Column: "c", Expression: "'x'"}}).
					RenameTo("p").
					GranteeAll(true)
			},
			want: "ALTER MASKING POLICY `p` ON `d`.`t` UPDATE `c` = 'x' TO ALL;",
		},
		{
			name: "without where keeps the stored condition",
			build: func() AlterMaskingPolicyQueryBuilder {
				return NewAlterMaskingPolicy("p", "d", "t", []ColumnMask{{Column: "c", Expression: "'x'"}}).
					GranteeAll(true).
					WithPriority(i64ptr(0))
			},
			want: "ALTER MASKING POLICY `p` ON `d`.`t` UPDATE `c` = 'x' TO ALL PRIORITY 0;",
		},
		{
			name: "TO ALL EXCEPT",
			build: func() AlterMaskingPolicyQueryBuilder {
				return NewAlterMaskingPolicy("p", "d", "t", []ColumnMask{{Column: "c", Expression: "'x'"}}).
					GranteeAll(true).
					GranteeAllExcept([]string{"admin"})
			},
			want: "ALTER MASKING POLICY `p` ON `d`.`t` UPDATE `c` = 'x' TO ALL EXCEPT `admin`;",
		},
		{
			name: "error: no grantee",
			build: func() AlterMaskingPolicyQueryBuilder {
				return NewAlterMaskingPolicy("p", "d", "t", []ColumnMask{{Column: "c", Expression: "'x'"}})
			},
			wantErr: true,
		},
		{
			name: "error: no masks",
			build: func() AlterMaskingPolicyQueryBuilder {
				return NewAlterMaskingPolicy("p", "d", "t", nil).GranteeNames([]string{"r"})
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
