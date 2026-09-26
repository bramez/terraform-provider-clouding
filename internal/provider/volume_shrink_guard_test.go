package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// The Clouding API can only grow a volume, so a smaller size has to fail in
// `terraform plan` with a clear message, instead of sending the API a request that
// will always come back as a 400.
func TestVolumeShrinkGuard(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		state       types.Int64
		plan        types.Int64
		expectError bool
	}{
		"grow":              {state: types.Int64Value(40), plan: types.Int64Value(50), expectError: false},
		"same size":         {state: types.Int64Value(40), plan: types.Int64Value(40), expectError: false},
		"shrink":            {state: types.Int64Value(50), plan: types.Int64Value(40), expectError: true},
		"create (no state)": {state: types.Int64Null(), plan: types.Int64Value(50), expectError: false},
		"unknown plan":      {state: types.Int64Value(40), plan: types.Int64Unknown(), expectError: false},
	}

	for name, test := range tests {
		// Go 1.21 shares the loop variable across iterations, and with t.Parallel()
		// the subtests would read it already overwritten.
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
			// The modifier never rewrites the plan, it only validates it.
			assert.Equal(t, test.plan, response.PlanValue)
		})
	}
}
