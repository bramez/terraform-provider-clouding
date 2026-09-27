package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bramez/terraform-provider-clouding/internal/clouding"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/stretchr/testify/assert"
)

// readServer runs Read() against a fake API answering with the given server
// payload, starting from the fixed prior state, and returns the refreshed
// firewall_id.
func readServer(t *testing.T, firewalls string) (*resource.ReadResponse, string) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(fmt.Sprintf(`
		{
		  "id": "gNMqlnNkga127pyk",
		  "name": "kaito",
		  "hostname": "kaito",
		  "flavor": "2x8",
		  "volumeSizeGb": 50,
		  "image": {"id": "jG4bZNnE8zKYx7LP", "name": "Debian 13 (64 Bit)"},
		  "status": "Active",
		  "powerState": "Running",
		  "publicIp": "200.234.228.186",
		  "accessConfiguration": {"sshKeyId": "MJpLa2W4PodQ9YOX", "savePassword": false},
		  "firewalls": %s
		}
		`, firewalls)))
		if err != nil {
			t.Errorf("error writing the response: %s", err)
		}
	}))
	t.Cleanup(srv.Close)

	api, err := clouding.NewAPI("token", clouding.WithEndpoint(srv.URL))
	assert.NoError(t, err)

	r := &ServerResource{client: api}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	state := tfsdk.State{Schema: schemaResp.Schema, Raw: serverValue(priorName, priorFlavor, priorSSD)}

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	var refreshed ServerResourceModel
	resp.Diagnostics.Append(resp.State.Get(context.Background(), &refreshed)...)

	return resp, refreshed.FirewallID.ValueString()
}

// The Clouding API reports an empty firewall list on GET servers/{id} even for a
// server that has one applied. Refreshing firewall_id to "" out of that made
// every plan show a fake diff against the configuration and, because the
// attribute requires replacement, turned an unrelated change into a destroy.
func TestServerResourceReadKeepsFirewallWhenApiReportsNone(t *testing.T) {
	t.Parallel()

	resp, firewallID := readServer(t, `[]`)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)
	assert.Equal(t, priorFirewall, firewallID)
}

// A firewall reported by the API still wins, so a firewall swapped outside
// Terraform is detected as drift instead of being hidden.
func TestServerResourceReadTakesFirewallReportedByApi(t *testing.T) {
	t.Parallel()

	resp, firewallID := readServer(t, `[{"id": "AE1GadQjRkK4kzpW", "name": "cravilab-prod"}]`)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)
	assert.Equal(t, "AE1GadQjRkK4kzpW", firewallID)
}
