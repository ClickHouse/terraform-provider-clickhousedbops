package maskingpolicy

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/tfutils"
)

type MaskingPolicy struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Database         types.String `tfsdk:"database_name"`
	Table            types.String `tfsdk:"table_name"`
	Masks            types.Map    `tfsdk:"masks"`
	WhereExpression  types.String `tfsdk:"where_expression"`
	GranteeNames     types.Set    `tfsdk:"grantee_names"`
	GranteeAllExcept types.Set    `tfsdk:"grantee_all_except"`
	Priority         types.Int64  `tfsdk:"priority"`
}

func (m *MaskingPolicy) toDBOps(ctx context.Context) (dbops.MaskingPolicy, diag.Diagnostics) {
	var diags diag.Diagnostics

	masks, d := mapToColumnMasks(ctx, m.Masks)
	diags.Append(d...)
	names, d := tfutils.SetToStringSlice(ctx, m.GranteeNames)
	diags.Append(d...)
	allExcept, d := tfutils.SetToStringSlice(ctx, m.GranteeAllExcept)
	diags.Append(d...)
	if diags.HasError() {
		return dbops.MaskingPolicy{}, diags
	}

	return dbops.MaskingPolicy{
		ID:               m.ID.ValueString(),
		Name:             m.Name.ValueString(),
		Database:         m.Database.ValueString(),
		Table:            m.Table.ValueString(),
		Masks:            masks,
		Where:            m.WhereExpression.ValueString(),
		GranteeNames:     names,
		GranteeAll:       !m.GranteeAllExcept.IsNull(),
		GranteeAllExcept: allExcept,
		Priority:         m.Priority.ValueInt64Pointer(),
	}, diags
}

func (m *MaskingPolicy) fromDBOps(result *dbops.MaskingPolicy) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(result.ID)
	m.Name = types.StringValue(result.Name)
	m.Database = types.StringValue(result.Database)
	m.Table = types.StringValue(result.Table)

	if result.Where == "" {
		m.WhereExpression = types.StringNull()
	} else {
		m.WhereExpression = types.StringValue(result.Where)
	}

	if result.Priority != nil {
		m.Priority = types.Int64Value(*result.Priority)
	} else {
		m.Priority = types.Int64Value(0)
	}

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

func mapToColumnMasks(ctx context.Context, m types.Map) ([]dbops.ColumnMask, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	var raw map[string]string
	diags := m.ElementsAs(ctx, &raw, false)
	if diags.HasError() {
		return nil, diags
	}
	masks := make([]dbops.ColumnMask, 0, len(raw))
	for col, expr := range raw {
		masks = append(masks, dbops.ColumnMask{Column: col, Expression: expr})
	}
	sort.Slice(masks, func(i, j int) bool { return masks[i].Column < masks[j].Column })
	return masks, diags
}
