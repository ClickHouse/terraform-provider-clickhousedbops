package maskingpolicy

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func Test_mapToColumnMasks_sortsByColumn(t *testing.T) {
	ctx := context.Background()
	m := types.MapValueMust(types.StringType, map[string]attr.Value{
		"logMessage": types.StringValue("'** redacted **'"),
		"clientIp":   types.StringValue("concat(x)"),
		"a":          types.StringValue("'z'"),
	})

	masks, err := mapToColumnMasks(ctx, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(masks) != 3 {
		t.Fatalf("expected 3 masks, got %d", len(masks))
	}
	want := []string{"a", "clientIp", "logMessage"}
	for i, w := range want {
		if masks[i].Column != w {
			t.Errorf("masks[%d].Column = %q, want %q", i, masks[i].Column, w)
		}
	}
	if masks[2].Expression != "'** redacted **'" {
		t.Errorf("logMessage expression = %q", masks[2].Expression)
	}
}

func Test_mapToColumnMasks_nilMap(t *testing.T) {
	masks, err := mapToColumnMasks(context.Background(), types.MapNull(types.StringType))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if masks != nil {
		t.Errorf("expected nil masks for null map, got %v", masks)
	}
}
