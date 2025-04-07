package role

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Role struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}
