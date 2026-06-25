// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestEnvironmentRoleResourceModifyPlanRejectsDestroy(t *testing.T) {
	r := &EnvironmentRoleResource{}
	resp := &resource.ModifyPlanResponse{}

	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{
		Plan: tfsdk.Plan{
			Raw: tftypes.NewValue(tftypes.DynamicPseudoType, nil),
		},
		State: tfsdk.State{
			Raw: tftypes.NewValue(tftypes.String, "present"),
		},
	}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected destroy plan diagnostic error")
	}
}

func TestEnvironmentRoleResourceModifyPlanAllowsCreate(t *testing.T) {
	r := &EnvironmentRoleResource{}
	resp := &resource.ModifyPlanResponse{}

	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{
		Plan: tfsdk.Plan{
			Raw: tftypes.NewValue(tftypes.String, "planned"),
		},
		State: tfsdk.State{
			Raw: tftypes.NewValue(tftypes.DynamicPseudoType, nil),
		},
	}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostic error: %v", resp.Diagnostics)
	}
}
