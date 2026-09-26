package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bramez/terraform-provider-clouding/internal/clouding"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

// firewallStateType mirrors the firewall resource schema, so a prior state can
// be built by hand without going through a full acceptance test.
var firewallStateType = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"id":           tftypes.String,
		"name":         tftypes.String,
		"description":  tftypes.String,
		"last_updated": tftypes.String,
	},
}

func firewallPriorState(t *testing.T, r *FirewallResource) tfsdk.State {
	t.Helper()

	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	return tfsdk.State{
		Schema: schemaResp.Schema,
		Raw: tftypes.NewValue(firewallStateType, map[string]tftypes.Value{
			"id":           tftypes.NewValue(tftypes.String, "LywOkvx5LWAp28NP"),
			"name":         tftypes.NewValue(tftypes.String, "cravilab-prod"),
			"description":  tftypes.NewValue(tftypes.String, "Firewall under test"),
			"last_updated": tftypes.NewValue(tftypes.String, "Friday, 26-Sep-26 17:11:50 CEST"),
		}),
	}
}

func readFirewall(t *testing.T, statusCode int, body string) *resource.ReadResponse {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("error writing the response: %s", err)
		}
	}))
	t.Cleanup(srv.Close)

	api, err := clouding.NewAPI("token", clouding.WithEndpoint(srv.URL))
	assert.NoError(t, err)

	r := &FirewallResource{client: api}
	state := firewallPriorState(t, r)

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	return resp
}

// A resource deleted outside Terraform must leave Read without an error and
// with the resource gone from state, so the next plan schedules it for
// creation. Before this, refresh raised a hard error and aborted every plan.
func TestFirewallResourceReadDropsDeletedResourceFromState(t *testing.T) {
	t.Parallel()

	resp := readFirewall(t, http.StatusNotFound,
		`{"title":"Not Found","status":404,"detail":"Resource not found"}`)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)
	assert.True(t, resp.State.Raw.IsNull(), "expected the resource to be removed from state")
}

// A transient failure must stay a hard error and leave state untouched, so a
// 500 or an expired token never silently drops a live resource.
func TestFirewallResourceReadKeepsStateOnServerError(t *testing.T) {
	t.Parallel()

	resp := readFirewall(t, http.StatusInternalServerError,
		`{"title":"Internal Server Error","status":500,"detail":"boom"}`)

	assert.True(t, resp.Diagnostics.HasError(), "expected a hard error on 500")
	assert.False(t, resp.State.Raw.IsNull(), "state must survive a transient failure")
}
