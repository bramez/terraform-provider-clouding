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
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

// readServer runs Read() against a fake API reporting the given firewalls,
// starting from the prior state with no backup strategy.
func readServer(t *testing.T, firewalls string) (*resource.ReadResponse, ServerResourceModel) {
	t.Helper()

	return readServerFrom(t, serverValue(priorName, priorFlavor, priorSSD), firewalls, `null`)
}

// readServerFrom is the same for the cases that need a different prior state or a
// backup strategy in the API response.
func readServerFrom(t *testing.T, stateRaw tftypes.Value, firewalls, backups string) (*resource.ReadResponse, ServerResourceModel) {
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
		  "firewalls": %s,
		  "backupPreferences": %s
		}
		`, firewalls, backups)))
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

	state := tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw}

	resp := &resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, resp)

	var refreshed ServerResourceModel
	resp.Diagnostics.Append(resp.State.Get(context.Background(), &refreshed)...)

	return resp, refreshed
}

// The Clouding API reports an empty firewall list on GET servers/{id} even for a
// server that has one applied. Refreshing firewall_id to "" out of that made
// every plan show a fake diff against the configuration and, because the
// attribute requires replacement, turned an unrelated change into a destroy.
func TestServerResourceReadKeepsFirewallWhenApiReportsNone(t *testing.T) {
	t.Parallel()

	resp, refreshed := readServer(t, `[]`)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)
	assert.Equal(t, priorFirewall, refreshed.FirewallID.ValueString())
}

// A firewall reported by the API still wins, so a firewall swapped outside
// Terraform is detected as drift instead of being hidden.
func TestServerResourceReadTakesFirewallReportedByApi(t *testing.T) {
	t.Parallel()

	resp, refreshed := readServer(t, `[{"id": "AE1GadQjRkK4kzpW", "name": "cravilab-prod"}]`)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)
	assert.Equal(t, "AE1GadQjRkK4kzpW", refreshed.FirewallID.ValueString())
}

// GET servers/{id} does not report the backup strategy in a shape this provider
// can read back, so an absent one means "no information" and the strategy already
// in state is kept. Clearing it made every plan want to reconfigure an unchanged
// strategy, which the API rejects with "No change required".
func TestServerResourceReadKeepsBackupsWhenApiReportsNone(t *testing.T) {
	t.Parallel()

	resp, refreshed := readServerFrom(t, serverValueBackups(7, "oneDay"), `[]`, `null`)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)
	if assert.NotNil(t, refreshed.BackupPreference) {
		assert.Equal(t, int64(7), refreshed.BackupPreference.Slots.ValueInt64())
		assert.Equal(t, "oneDay", refreshed.BackupPreference.Frequency.ValueString())
	}
}

// A strategy reported without a frequency is partial information — the API names
// that field differently from its published schema — so it is discarded too
// rather than written into state as an empty frequency.
func TestServerResourceReadKeepsBackupsWhenApiReportsPartial(t *testing.T) {
	t.Parallel()

	resp, refreshed := readServerFrom(t, serverValueBackups(7, "oneDay"), `[]`,
		`{"slots": 7, "createBackupEvery": "oneDay"}`)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)
	if assert.NotNil(t, refreshed.BackupPreference) {
		assert.Equal(t, "oneDay", refreshed.BackupPreference.Frequency.ValueString())
	}
}

// And a strategy reported by the API lands in state with the API's own values.
func TestServerResourceReadTakesBackupsReportedByApi(t *testing.T) {
	t.Parallel()

	resp, refreshed := readServerFrom(t, serverValue(priorName, priorFlavor, priorSSD), `[]`,
		`{"slots": 14, "frequency": "twoDays"}`)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)
	if assert.NotNil(t, refreshed.BackupPreference) {
		assert.Equal(t, int64(14), refreshed.BackupPreference.Slots.ValueInt64())
		assert.Equal(t, "twoDays", refreshed.BackupPreference.Frequency.ValueString())
	}
}
