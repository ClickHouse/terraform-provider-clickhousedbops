package database

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Database struct {
	Name    types.String `tfsdk:"name"`
	Comment types.String `tfsdk:"comment"`
}
