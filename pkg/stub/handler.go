package stub

import (
	"github.com/solo-io/envoy-operator/pkg/envoy"

	"github.com/operator-framework/operator-sdk/pkg/sdk/handler"
	"github.com/operator-framework/operator-sdk/pkg/sdk/types"

	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
)

func NewHandler(namespace string) handler.Handler {
	return &Handler{namespace: namespace}
}

type Handler struct {
	namespace string
}

func (h *Handler) Handle(ctx types.Context, event types.Event) error {
	switch o := event.Object.(type) {
	case *api.Envoy:
		// injections must be cleaned up
		// other deleted things will get GC'ed by kube.
		if event.Deleted {
			return envoy.DeleteEnvoyInjection(h.namespace, o)
		}
		return envoy.Reconcile(h.namespace, o)
	}
	return nil
}
