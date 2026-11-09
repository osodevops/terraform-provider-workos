package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

type JWTTemplateResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Content   types.String `tfsdk:"content"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}
