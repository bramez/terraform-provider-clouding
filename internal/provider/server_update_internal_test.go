package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bramez/terraform-provider-clouding/internal/clouding"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

// serverStateType mirrors the server resource schema, so a prior state and a plan
// can be built by hand without going through an acceptance test against the real
// API.
var serverStateType = tftypes.Object{
	AttributeTypes: map[string]tftypes.Type{
		"id":          tftypes.String,
		"name":        tftypes.String,
		"hostname":    tftypes.String,
		"flavor_id":   tftypes.String,
		"firewall_id": tftypes.String,
		"access_configuration": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"ssh_key_id":    tftypes.String,
			"password":      tftypes.String,
			"save_password": tftypes.Bool,
		}},
		"volume": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"source": tftypes.String,
			"id":     tftypes.String,
			"ssd_gb": tftypes.Number,
		}},
		"enable_private_network":           tftypes.Bool,
		"enable_strict_antiddos_filtering": tftypes.Bool,
		"user_data":                        tftypes.String,
		"backup_preference": tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"slots":     tftypes.Number,
			"frequency": tftypes.String,
		}},
		"last_updated": tftypes.String,
		"timeouts":     tftypes.Object{AttributeTypes: map[string]tftypes.Type{"create": tftypes.String}},
	},
}

func serverValue(name, flavorID string, ssdGB int64) tftypes.Value {
	return tftypes.NewValue(serverStateType, map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, "jG4bZNnE8zKYx7LP"),
		"name":        tftypes.NewValue(tftypes.String, name),
		"hostname":    tftypes.NewValue(tftypes.String, "kaito"),
		"flavor_id":   tftypes.NewValue(tftypes.String, flavorID),
		"firewall_id": tftypes.NewValue(tftypes.String, "LywOkvx5LWAp28NP"),
		"access_configuration": tftypes.NewValue(serverStateType.AttributeTypes["access_configuration"], map[string]tftypes.Value{
			"ssh_key_id":    tftypes.NewValue(tftypes.String, "Wl2ZE9mk6l37y1OZ"),
			"password":      tftypes.NewValue(tftypes.String, nil),
			"save_password": tftypes.NewValue(tftypes.Bool, false),
		}),
		"volume": tftypes.NewValue(serverStateType.AttributeTypes["volume"], map[string]tftypes.Value{
			"source": tftypes.NewValue(tftypes.String, "image"),
			"id":     tftypes.NewValue(tftypes.String, "lo1qJ9oZb1xGMEgD"),
			"ssd_gb": tftypes.NewValue(tftypes.Number, ssdGB),
		}),
		"enable_private_network":           tftypes.NewValue(tftypes.Bool, false),
		"enable_strict_antiddos_filtering": tftypes.NewValue(tftypes.Bool, false),
		"user_data":                        tftypes.NewValue(tftypes.String, ""),
		"backup_preference":                tftypes.NewValue(serverStateType.AttributeTypes["backup_preference"], nil),
		"last_updated":                     tftypes.NewValue(tftypes.String, "Friday, 26-Sep-26 17:11:50 CEST"),
		"timeouts":                         tftypes.NewValue(serverStateType.AttributeTypes["timeouts"], map[string]tftypes.Value{"create": tftypes.NewValue(tftypes.String, nil)}),
	})
}

// The prior state every test starts from: the server as Clouding reports it
// before the change under test. It is fixed so that each test only has to state
// what it changes.
const (
	priorName   = "kaito"
	priorFlavor = "2x8"
	priorSSD    = 40
)

type recordedRequest struct {
	method string
	path   string
	body   string
}

// updateServer runs Update() against a fake HTTP server, going from the fixed
// prior state to the given plan, and returns the requests the provider issued to
// the API.
func updateServer(t *testing.T, planName, planFlavor string, planSSD int64) (*resource.UpdateResponse, []recordedRequest) {
	t.Helper()

	var recorded []recordedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("error reading the request body: %s", err)
		}
		recorded = append(recorded, recordedRequest{method: r.Method, path: r.URL.Path, body: string(body)})

		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/actions/awqYZWO4njxQyOV0":
			w.WriteHeader(http.StatusOK)
			_, err = w.Write([]byte(`{"id":"awqYZWO4njxQyOV0","status":"completed","type":"resize"}`))
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusAccepted)
			_, err = w.Write([]byte(`{"id":"awqYZWO4njxQyOV0","status":"inProgress","type":"resize","resourceId":"jG4bZNnE8zKYx7LP","resourceType":"server"}`))
		default:
			w.WriteHeader(http.StatusNoContent)
		}
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
	plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: serverValue(planName, planFlavor, planSSD)}

	resp := &resource.UpdateResponse{State: state}
	r.Update(context.Background(), resource.UpdateRequest{State: state, Plan: plan}, resp)

	return resp, recorded
}

func findRequest(recorded []recordedRequest, method, path string) *recordedRequest {
	for _, request := range recorded {
		if request.method == method && request.path == path {
			return &request
		}
	}
	return nil
}

// Growing the volume must be settled with an in-place resize: before this,
// `ssd_gb` carried RequiresReplace and Terraform destroyed and recreated the
// server.
func TestServerResourceUpdateResizesVolumeInPlace(t *testing.T) {
	t.Parallel()

	resp, recorded := updateServer(t, priorName, priorFlavor, 50)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)

	resize := findRequest(recorded, http.MethodPost, "/v1/servers/jG4bZNnE8zKYx7LP/resize")
	if assert.NotNil(t, resize, "expected a resize call, got: %v", recorded) {
		// The flavor does not change, so it is omitted to keep the API off it.
		assert.JSONEq(t, `{"volumeSizeGb":50}`, resize.body)
	}
	// With no name change there is no reason to call rename.
	assert.Nil(t, findRequest(recorded, http.MethodPatch, "/v1/servers/jG4bZNnE8zKYx7LP/rename"))
	// And the async action has to complete before the apply is called done.
	assert.NotNil(t, findRequest(recorded, http.MethodGet, "/v1/actions/awqYZWO4njxQyOV0"))
}

// The same endpoint changes CPU/RAM, so the flavor is resized without recreating
// the server either.
func TestServerResourceUpdateResizesFlavorInPlace(t *testing.T) {
	t.Parallel()

	resp, recorded := updateServer(t, priorName, "4x16", priorSSD)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)

	resize := findRequest(recorded, http.MethodPost, "/v1/servers/jG4bZNnE8zKYx7LP/resize")
	if assert.NotNil(t, resize, "expected a resize call, got: %v", recorded) {
		assert.JSONEq(t, `{"flavorId":"4x16"}`, resize.body)
	}
}

// Changing flavor and volume at once is a single call, not two.
func TestServerResourceUpdateResizesFlavorAndVolumeTogether(t *testing.T) {
	t.Parallel()

	resp, recorded := updateServer(t, priorName, "4x16", 50)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)

	var resizeCalls int
	for _, request := range recorded {
		if request.method == http.MethodPost && request.path == "/v1/servers/jG4bZNnE8zKYx7LP/resize" {
			resizeCalls++
			assert.JSONEq(t, `{"flavorId":"4x16","volumeSizeGb":50}`, request.body)
		}
	}
	assert.Equal(t, 1, resizeCalls, fmt.Sprintf("expected exactly one resize call, got: %v", recorded))
}

// Rename still works, and when only the name changes no resize fires.
func TestServerResourceUpdateRenamesWithoutResizing(t *testing.T) {
	t.Parallel()

	resp, recorded := updateServer(t, "kaito-renamed", priorFlavor, priorSSD)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)

	rename := findRequest(recorded, http.MethodPatch, "/v1/servers/jG4bZNnE8zKYx7LP/rename")
	if assert.NotNil(t, rename, "expected a rename call, got: %v", recorded) {
		// The rename payload drags empty fields along from the Server struct (image,
		// cost, action); what matters is that it carries the new name.
		assert.Contains(t, rename.body, `"newServerName":"kaito-renamed"`)
	}
	assert.Nil(t, findRequest(recorded, http.MethodPost, "/v1/servers/jG4bZNnE8zKYx7LP/resize"))
}

// With no change to name, flavor or volume, Update must not touch the API.
func TestServerResourceUpdateWithoutChangesCallsNothing(t *testing.T) {
	t.Parallel()

	resp, recorded := updateServer(t, priorName, priorFlavor, priorSSD)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)
	assert.Empty(t, recorded, "expected no API calls")
}
