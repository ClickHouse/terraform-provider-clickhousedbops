package rowpolicy

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/tfutils"
)

type RowPolicy struct {
	ID               types.String `tfsdk:"id"`
	ClusterName      types.String `tfsdk:"cluster_name"`
	Name             types.String `tfsdk:"name"`
	Database         types.String `tfsdk:"database_name"`
	Table            types.String `tfsdk:"table_name"`
	SelectFilter     types.String `tfsdk:"select_filter"`
	IsRestrictive    types.Bool   `tfsdk:"is_restrictive"`
	GranteeNames     types.Set    `tfsdk:"grantee_names"`
	GranteeAllExcept types.Set    `tfsdk:"grantee_all_except"`
}

func (m *RowPolicy) toDBOps(ctx context.Context) (dbops.RowPolicy, diag.Diagnostics) {
	var diags diag.Diagnostics

	granteeNames, d := tfutils.SetToStringSlice(ctx, m.GranteeNames)
	diags.Append(d...)

	granteeAllExcept, d := tfutils.SetToStringSlice(ctx, m.GranteeAllExcept)
	diags.Append(d...)

	return dbops.RowPolicy{
		ID:               m.ID.ValueString(),
		Name:             m.Name.ValueString(),
		Database:         m.Database.ValueString(),
		Table:            m.Table.ValueString(),
		SelectFilter:     m.SelectFilter.ValueString(),
		IsRestrictive:    m.IsRestrictive.ValueBool(),
		GranteeNames:     granteeNames,
		GranteeAll:       !m.GranteeAllExcept.IsNull(),
		GranteeAllExcept: granteeAllExcept,
	}, diags
}

func (m *RowPolicy) fromDBOps(result *dbops.RowPolicy) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(result.ID)
	m.Name = types.StringValue(result.Name)
	m.Database = types.StringValue(result.Database)
	m.Table = types.StringValue(result.Table)
	m.SelectFilter = types.StringValue(result.SelectFilter)
	m.IsRestrictive = types.BoolValue(result.IsRestrictive)
	m.GranteeNames = types.SetNull(types.StringType)
	m.GranteeAllExcept = types.SetNull(types.StringType)
	if result.GranteeAll {
		elements := make([]attr.Value, len(result.GranteeAllExcept))
		for i, s := range result.GranteeAllExcept {
			elements[i] = types.StringValue(s)
		}
		set, d := types.SetValue(types.StringType, elements)
		diags.Append(d...)
		m.GranteeAllExcept = set
		return diags
	}

	set, d := tfutils.StringSliceToSet(result.GranteeNames)
	diags.Append(d...)
	m.GranteeNames = set
	return diags
}
