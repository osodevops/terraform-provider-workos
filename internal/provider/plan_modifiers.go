// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// useStateForUnknownIfConfigUnchanged is a plan modifier that preserves the
// state value when no user-configurable attributes have changed, but marks the
// value as unknown when they have, allowing the API response to set it.
type useStateForUnknownIfConfigUnchanged struct {
	configAttributes []path.Path
}

func (m useStateForUnknownIfConfigUnchanged) Description(_ context.Context) string {
	return "Uses state value when config is unchanged, otherwise marks as unknown."
}

func (m useStateForUnknownIfConfigUnchanged) MarkdownDescription(_ context.Context) string {
	return "Uses state value when config is unchanged, otherwise marks as unknown."
}

func (m useStateForUnknownIfConfigUnchanged) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// On create (no prior state), leave as unknown.
	if req.StateValue.IsNull() {
		return
	}

	// Compare each tracked config attribute between plan and state.
	for _, attrPath := range m.configAttributes {
		var planVal, stateVal attr.Value
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, attrPath, &planVal)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, attrPath, &stateVal)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !planVal.Equal(stateVal) {
			// Something changed — mark unknown so the API response value is accepted.
			resp.PlanValue = types.StringUnknown()
			return
		}
	}

	// Nothing changed — keep the state value (no diff).
	resp.PlanValue = req.StateValue
}
