// Copyright (c) OSO DevOps
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/osodevops/terraform-provider-workos/internal/client"
)

func connectApplicationClientSecretSchema(t *testing.T) schema.Schema {
	t.Helper()

	resp := &fwresource.SchemaResponse{}
	(&ConnectApplicationClientSecretResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func connectApplicationClientSecretPlan(t *testing.T, s schema.Schema, applicationID string) tfsdk.Plan {
	t.Helper()

	objectType, ok := s.Type().TerraformType(context.Background()).(tftypes.Object)
	if !ok {
		t.Fatalf("expected object type for schema, got %T", s.Type().TerraformType(context.Background()))
	}

	values := make(map[string]tftypes.Value, len(objectType.AttributeTypes))
	for name, attributeType := range objectType.AttributeTypes {
		if name == "application_id" {
			values[name] = tftypes.NewValue(tftypes.String, applicationID)
			continue
		}
		values[name] = tftypes.NewValue(attributeType, tftypes.UnknownValue)
	}

	return tfsdk.Plan{
		Schema: s,
		Raw:    tftypes.NewValue(objectType, values),
	}
}

func createConnectApplicationForSecretTests(t *testing.T, workosClient *client.Client) *client.ConnectApplication {
	t.Helper()

	app, err := workosClient.CreateConnectApplication(context.Background(), &client.ConnectApplicationCreateRequest{
		ApplicationType: "m2m",
		Name:            "billing",
		OrganizationID:  "org_00000001",
	})
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}
	return app
}

func TestConnectApplicationClientSecretResourceCreate_StoresPlaintextOnce(t *testing.T) {
	ctx := context.Background()
	_, server := newConnectApplicationMockAPI(t)

	workosClient, err := client.NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	app := createConnectApplicationForSecretTests(t, workosClient)

	r := &ConnectApplicationClientSecretResource{client: workosClient}
	s := connectApplicationClientSecretSchema(t)
	plan := connectApplicationClientSecretPlan(t, s, app.ID)

	createResp := &fwresource.CreateResponse{State: tfsdk.State{
		Schema: s,
		Raw:    tftypes.NewValue(s.Type().TerraformType(ctx), nil),
	}}
	r.Create(ctx, fwresource.CreateRequest{Plan: plan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create returned errors: %v", createResp.Diagnostics)
	}

	var state ConnectApplicationClientSecretResourceModel
	if diags := createResp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("failed to read resulting state: %v", diags)
	}
	if state.Secret.ValueString() == "" {
		t.Fatal("expected plaintext secret in state after create")
	}
	createdPlaintext := state.Secret.ValueString()

	readResp := &fwresource.ReadResponse{State: createResp.State}
	r.Read(ctx, fwresource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read returned errors: %v", readResp.Diagnostics)
	}
	if diags := readResp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("failed to read refreshed state: %v", diags)
	}
	if state.Secret.ValueString() != createdPlaintext {
		t.Fatalf("expected plaintext to be preserved from state, got %q", state.Secret.ValueString())
	}
}

func TestConnectApplicationClientSecretResourceRead_RemovesMissingSecret(t *testing.T) {
	ctx := context.Background()
	_, server := newConnectApplicationMockAPI(t)

	workosClient, err := client.NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	app := createConnectApplicationForSecretTests(t, workosClient)

	r := &ConnectApplicationClientSecretResource{client: workosClient}
	s := connectApplicationClientSecretSchema(t)
	plan := connectApplicationClientSecretPlan(t, s, app.ID)

	createResp := &fwresource.CreateResponse{State: tfsdk.State{
		Schema: s,
		Raw:    tftypes.NewValue(s.Type().TerraformType(ctx), nil),
	}}
	r.Create(ctx, fwresource.CreateRequest{Plan: plan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create returned errors: %v", createResp.Diagnostics)
	}

	var state ConnectApplicationClientSecretResourceModel
	if diags := createResp.State.Get(ctx, &state); diags.HasError() {
		t.Fatalf("failed to read resulting state: %v", diags)
	}

	if err := workosClient.DeleteConnectApplicationClientSecret(ctx, state.ID.ValueString()); err != nil {
		t.Fatalf("failed to revoke secret: %v", err)
	}

	readResp := &fwresource.ReadResponse{State: createResp.State}
	r.Read(ctx, fwresource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read returned errors: %v", readResp.Diagnostics)
	}
	if !readResp.State.Raw.IsNull() {
		t.Fatal("expected missing secret to be removed from state")
	}
}

func TestConnectApplicationClientSecretResourceCreate_AcceptsClientID(t *testing.T) {
	ctx := context.Background()
	_, server := newConnectApplicationMockAPI(t)

	workosClient, err := client.NewClient("sk_test", "", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	app := createConnectApplicationForSecretTests(t, workosClient)

	r := &ConnectApplicationClientSecretResource{client: workosClient}
	s := connectApplicationClientSecretSchema(t)
	plan := connectApplicationClientSecretPlan(t, s, app.ClientID)

	createResp := &fwresource.CreateResponse{State: tfsdk.State{
		Schema: s,
		Raw:    tftypes.NewValue(s.Type().TerraformType(ctx), nil),
	}}
	r.Create(ctx, fwresource.CreateRequest{Plan: plan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create returned errors: %v", createResp.Diagnostics)
	}

	readResp := &fwresource.ReadResponse{State: createResp.State}
	r.Read(ctx, fwresource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read returned errors: %v", readResp.Diagnostics)
	}
	if readResp.State.Raw.IsNull() {
		t.Fatal("expected secret created with client_id to still be readable")
	}
}

func TestAccConnectApplicationClientSecretResource_MockAPI(t *testing.T) {
	_, server := newConnectApplicationMockAPI(t)

	t.Setenv("WORKOS_API_KEY", "sk_test")
	t.Setenv("WORKOS_BASE_URL", server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccConnectApplicationClientSecretMockConfig(server.URL, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("workos_connect_application_client_secret.current", "id"),
					resource.TestCheckResourceAttrSet("workos_connect_application_client_secret.current", "secret"),
					resource.TestCheckResourceAttrSet("workos_connect_application_client_secret.current", "secret_hint"),
					resource.TestCheckResourceAttrPair(
						"workos_connect_application_client_secret.current", "application_id",
						"workos_connect_application.m2m", "id",
					),
				),
			},
			{
				ResourceName:            "workos_connect_application_client_secret.current",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["workos_connect_application_client_secret.current"]
					if !ok {
						return "", fmt.Errorf("resource not found: workos_connect_application_client_secret.current")
					}
					return fmt.Sprintf("%s/%s",
						rs.Primary.Attributes["application_id"],
						rs.Primary.Attributes["id"],
					), nil
				},
			},
			{
				Config: testAccConnectApplicationClientSecretMockConfig(server.URL, 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("workos_connect_application_client_secret.current", "id"),
					resource.TestCheckResourceAttrSet("workos_connect_application_client_secret.previous", "id"),
				),
			},
		},
	})
}

func testAccConnectApplicationClientSecretMockConfig(baseURL string, secretCount int) string {
	config := fmt.Sprintf(`
provider "workos" {
  api_key  = "sk_test"
  base_url = %[1]q
}

resource "workos_organization" "public" {
  name = "Example Public"
}

resource "workos_connect_application" "m2m" {
  name             = "billing-worker"
  application_type = "m2m"
  organization_id  = workos_organization.public.id
  scopes           = ["billing:read"]
}

resource "workos_connect_application_client_secret" "current" {
  application_id = workos_connect_application.m2m.id
}
`, baseURL)

	if secretCount > 1 {
		config += `
resource "workos_connect_application_client_secret" "previous" {
  application_id = workos_connect_application.m2m.id
}
`
	}

	return config
}
