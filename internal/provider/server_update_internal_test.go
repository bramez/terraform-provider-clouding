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

// serverStateType replica el schema del recurso server, para poder construir a
// mano un estado previo y un plan sin pasar por un test de aceptación contra la
// API real.
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

type recordedRequest struct {
	method string
	path   string
	body   string
}

// updateServer ejecuta Update() con el estado previo (stateName/stateFlavor/stateSSD)
// y el plan deseado, contra un servidor HTTP falso, y devuelve las peticiones que
// el provider ha lanzado a la API.
func updateServer(t *testing.T, stateName, stateFlavor string, stateSSD int64, planName, planFlavor string, planSSD int64) (*resource.UpdateResponse, []recordedRequest) {
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

	state := tfsdk.State{Schema: schemaResp.Schema, Raw: serverValue(stateName, stateFlavor, stateSSD)}
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

// Agrandar el disco tiene que resolverse con un resize in-place: antes de esto
// `ssd_gb` llevaba RequiresReplace y Terraform destruía y recreaba el servidor.
func TestServerResourceUpdateResizesVolumeInPlace(t *testing.T) {
	t.Parallel()

	resp, recorded := updateServer(t, "kaito", "2x8", 40, "kaito", "2x8", 50)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)

	resize := findRequest(recorded, http.MethodPost, "/v1/servers/jG4bZNnE8zKYx7LP/resize")
	if assert.NotNil(t, resize, "expected a resize call, got: %v", recorded) {
		// El flavor no cambia, así que se omite para que la API no lo toque.
		assert.JSONEq(t, `{"volumeSizeGb":50}`, resize.body)
	}
	// Sin cambio de nombre no hay que llamar al rename.
	assert.Nil(t, findRequest(recorded, http.MethodPatch, "/v1/servers/jG4bZNnE8zKYx7LP/rename"))
	// Y hay que esperar a que la acción asíncrona termine antes de dar el apply por bueno.
	assert.NotNil(t, findRequest(recorded, http.MethodGet, "/v1/actions/awqYZWO4njxQyOV0"))
}

// El mismo endpoint cambia CPU/RAM, así que el flavor también se redimensiona
// sin recrear el servidor.
func TestServerResourceUpdateResizesFlavorInPlace(t *testing.T) {
	t.Parallel()

	resp, recorded := updateServer(t, "kaito", "2x8", 40, "kaito", "4x16", 40)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)

	resize := findRequest(recorded, http.MethodPost, "/v1/servers/jG4bZNnE8zKYx7LP/resize")
	if assert.NotNil(t, resize, "expected a resize call, got: %v", recorded) {
		assert.JSONEq(t, `{"flavorId":"4x16"}`, resize.body)
	}
}

// Cambiar flavor y disco a la vez es una sola llamada, no dos.
func TestServerResourceUpdateResizesFlavorAndVolumeTogether(t *testing.T) {
	t.Parallel()

	resp, recorded := updateServer(t, "kaito", "2x8", 40, "kaito", "4x16", 50)

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

// El rename sigue funcionando y, si solo cambia el nombre, no se dispara ningún
// resize.
func TestServerResourceUpdateRenamesWithoutResizing(t *testing.T) {
	t.Parallel()

	resp, recorded := updateServer(t, "kaito", "2x8", 40, "kaito-nuevo", "2x8", 40)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)

	rename := findRequest(recorded, http.MethodPatch, "/v1/servers/jG4bZNnE8zKYx7LP/rename")
	if assert.NotNil(t, rename, "expected a rename call, got: %v", recorded) {
		// El payload del rename arrastra campos vacíos del struct Server (image,
		// cost, action); lo que importa es que lleve el nombre nuevo.
		assert.Contains(t, rename.body, `"newServerName":"kaito-nuevo"`)
	}
	assert.Nil(t, findRequest(recorded, http.MethodPost, "/v1/servers/jG4bZNnE8zKYx7LP/resize"))
}

// Sin cambios de nombre, flavor ni disco, Update no debe tocar la API.
func TestServerResourceUpdateWithoutChangesCallsNothing(t *testing.T) {
	t.Parallel()

	resp, recorded := updateServer(t, "kaito", "2x8", 40, "kaito", "2x8", 40)

	assert.False(t, resp.Diagnostics.HasError(), "expected no error, got: %v", resp.Diagnostics)
	assert.Empty(t, recorded, "expected no API calls")
}
