package database

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

func Test_engineState(t *testing.T) {
	replicated := &dbops.Database{
		EngineName: "Replicated",
		Engine:     "Replicated('/clickhouse/databases/{uuid}', '{shard}', '{replica}')",
	}

	tests := []struct {
		name  string
		known types.String
		db    *dbops.Database
		want  types.String
	}{
		{
			name:  "unset engine takes engine_full",
			known: types.StringUnknown(),
			db:    replicated,
			want:  types.StringValue(replicated.Engine),
		},
		{
			name:  "null engine (import, or state from an older provider) takes engine_full",
			known: types.StringNull(),
			db:    replicated,
			want:  types.StringValue(replicated.Engine),
		},
		{
			name:  "configured engine without arguments is kept",
			known: types.StringValue("Replicated"),
			db:    replicated,
			want:  types.StringValue("Replicated"),
		},
		{
			name:  "configured engine with different formatting is kept",
			known: types.StringValue("Replicated('/clickhouse/databases/{uuid}','{shard}','{replica}')"),
			db:    replicated,
			want:  types.StringValue("Replicated('/clickhouse/databases/{uuid}','{shard}','{replica}')"),
		},
		{
			name:  "different engine on the server is drift",
			known: types.StringValue("Atomic"),
			db:    replicated,
			want:  types.StringValue(replicated.Engine),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := engineState(tt.known, tt.db); !got.Equal(tt.want) {
				t.Errorf("engineState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_engineNameChanged(t *testing.T) {
	tests := []struct {
		name  string
		state string
		plan  string
		want  bool
	}{
		{
			name:  "imported engine_full against the bare engine name",
			state: "Replicated('/clickhouse/databases/{database}', '{shard}', '{replica}')",
			plan:  "Replicated",
			want:  false,
		},
		{
			name:  "different formatting of the same engine",
			state: "Replicated('/clickhouse/databases/{database}', '{shard}', '{replica}')",
			plan:  "Replicated('/clickhouse/databases/{database}','{shard}','{replica}')",
			want:  false,
		},
		{
			name: "state from a provider version without engine",
			plan: "Replicated",
			want: false,
		},
		{
			name:  "different engine name",
			state: "Atomic",
			plan:  "Replicated('/clickhouse/databases/{database}', '{shard}', '{replica}')",
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := types.StringNull()
			if tt.state != "" {
				state = types.StringValue(tt.state)
			}
			req := planmodifier.StringRequest{StateValue: state, PlanValue: types.StringValue(tt.plan)}
			resp := &stringplanmodifier.RequiresReplaceIfFuncResponse{}
			engineNameChanged(context.Background(), req, resp)
			if resp.RequiresReplace != tt.want {
				t.Errorf("RequiresReplace = %v, want %v", resp.RequiresReplace, tt.want)
			}
		})
	}
}
