package envoy

import (
	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk/action"
)

// Reconcile reconciles the Envoy instance's state to the spec specified in the crd
func Reconcile(e *api.Envoy) (err error) {
	e = e.DeepCopy()
	// Simulate initializer.
	changed := e.SetDefaults()
	if changed {
		return action.Update(e)
	}

	err = prepareEnvoyConfig(e)
	if err != nil {
		return err
	}

	err = deployEnvoy(e)
	if err != nil {
		return err
	}

	if e.Spec.Deployment != nil {
		err = syncDeployment(e)
		if err != nil {
			return err
		}
	}

	return nil
}
