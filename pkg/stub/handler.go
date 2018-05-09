package stub

import (
	"github.com/solo-io/envoy-operator/pkg/envoy"

	"github.com/operator-framework/operator-sdk/pkg/sdk/handler"
	"github.com/operator-framework/operator-sdk/pkg/sdk/types"

	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
)

func NewHandler() handler.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx types.Context, event types.Event) error {

	// deleted things will get GC'ed by kube.
	if event.Deleted {
		return nil
	}
	switch o := event.Object.(type) {
	case *api.Envoy:
		return envoy.Reconcile(o)
	}
	return nil
}
