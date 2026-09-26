package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// volumeShrinkGuard rejects any reduction of the volume size at plan time.
// Clouding's POST servers/{id}/resize endpoint can only grow a volume ("the volume
// size must be equal or greater than the current"), so without this guard the user
// would get an apparently valid plan that blows up with a 400 half way through the
// apply. It is a plan modifier rather than a validator because it needs to compare
// against the prior state, which validators do not receive.
type volumeShrinkGuard struct{}

// Ensure the plan modifier satisfies the framework interface.
var _ planmodifier.Int64 = volumeShrinkGuard{}

func (m volumeShrinkGuard) Description(ctx context.Context) string {
	return "The volume size can only grow; shrinking it is rejected at plan time."
}

func (m volumeShrinkGuard) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m volumeShrinkGuard) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	// With no prior state (creation) or an unresolved value there is nothing to
	// compare against.
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	current := req.StateValue.ValueInt64()
	desired := req.PlanValue.ValueInt64()
	if desired >= current {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Volume cannot be shrunk",
		fmt.Sprintf(
			"The server has a %d GB volume and the configuration asks for %d GB. "+
				"The Clouding API can only grow a volume. If you really want a "+
				"smaller disk, the server has to be recreated explicitly "+
				"(`terraform apply -replace=...`), accepting the loss of the data "+
				"on the current volume.",
			current, desired,
		),
	)
}
