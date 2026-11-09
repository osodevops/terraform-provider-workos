package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

type AuthKitOAuthResourceResourceModel struct {
	ID        types.String `tfsdk:"id"`
	URI       types.String `tfsdk:"uri"`
	Default   types.Bool   `tfsdk:"default"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}
