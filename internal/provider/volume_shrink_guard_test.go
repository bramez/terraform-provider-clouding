package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// La API de Clouding solo sabe hacer crecer el disco, así que reducir el tamaño
// tiene que fallar en `terraform plan` con un mensaje claro, en vez de mandar a
// la API una petición que siempre devolverá 400.
func TestVolumeShrinkGuard(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		state       types.Int64
		plan        types.Int64
		expectError bool
	}{
		"crecer":                {state: types.Int64Value(40), plan: types.Int64Value(50), expectError: false},
		"mismo tamaño":          {state: types.Int64Value(40), plan: types.Int64Value(40), expectError: false},
		"encoger":               {state: types.Int64Value(50), plan: types.Int64Value(40), expectError: true},
		"creación (sin estado)": {state: types.Int64Null(), plan: types.Int64Value(50), expectError: false},
		"plan desconocido":      {state: types.Int64Value(40), plan: types.Int64Unknown(), expectError: false},
	}

	for name, test := range tests {
		// Go 1.21 comparte la variable de bucle entre iteraciones, y con
		// t.Parallel() los subtests la leerían ya sobrescrita.
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			request := planmodifier.Int64Request{
				Path:       path.Root("volume").AtName("ssd_gb"),
				StateValue: test.state,
				PlanValue:  test.plan,
			}
			response := &planmodifier.Int64Response{PlanValue: test.plan}

			volumeShrinkGuard{}.PlanModifyInt64(context.Background(), request, response)

			assert.Equal(t, test.expectError, response.Diagnostics.HasError())
			if test.expectError {
				assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), "40")
				assert.Contains(t, response.Diagnostics.Errors()[0].Detail(), "50")
			}
			// El modificador nunca reescribe el plan, solo lo valida.
			assert.Equal(t, test.plan, response.PlanValue)
		})
	}
}
