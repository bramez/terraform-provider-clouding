package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// volumeShrinkGuard rechaza en tiempo de plan cualquier reducción del tamaño del
// disco. El endpoint POST servers/{id}/resize de Clouding solo permite crecer
// ("the volume size must be equal or greater than the current"), así que sin este
// guard el usuario vería un plan aparentemente válido que revienta con un 400 a
// mitad del apply. Es un plan modifier y no un validator porque necesita comparar
// contra el estado previo, que los validators no reciben.
type volumeShrinkGuard struct{}

// Ensure the plan modifier satisfies the framework interface.
var _ planmodifier.Int64 = volumeShrinkGuard{}

func (m volumeShrinkGuard) Description(ctx context.Context) string {
	return "El tamaño del disco solo puede crecer; reducirlo se rechaza en el plan."
}

func (m volumeShrinkGuard) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m volumeShrinkGuard) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	// Sin estado previo (creación) o con un valor todavía sin resolver no hay
	// nada que comparar.
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
		"El disco no se puede reducir",
		fmt.Sprintf(
			"El servidor tiene un disco de %d GB y la configuración pide %d GB. "+
				"La API de Clouding solo permite hacer crecer el volumen. "+
				"Si de verdad quieres un disco menor hay que recrear el servidor "+
				"de forma explícita (`terraform apply -replace=...`), asumiendo la "+
				"pérdida de los datos del disco actual.",
			current, desired,
		),
	)
}
