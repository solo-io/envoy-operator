package envoy

import (
	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk/action"
)

// Reconcile reconciles the vault cluster's state to the spec specified by vr
// by preparing the TLS secrets, deploying the etcd and vault cluster,
// and finally updating the vault deployment if needed.
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
