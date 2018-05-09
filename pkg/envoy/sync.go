package envoy

import (
	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/sdk/action"
	"github.com/operator-framework/operator-sdk/pkg/sdk/query"

	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func syncDeployment(e *api.Envoy) error {

	// get the envoy deployment

	d := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.GetName(),
			Namespace: e.GetNamespace(),
		}}

	err := query.Get(d)
	if err != nil {
		return fmt.Errorf("failed to get deployment (%s): %v", d.Name, err)
	}

	var reps int32
	reps = int32(e.Spec.Deployment.Replicas)

	if *d.Spec.Replicas != reps {
		d.Spec.Replicas = &reps
		err = action.Update(d)
		if err != nil {
			return fmt.Errorf("failed to update size of deployment (%s): %v", d.Name, err)
		}
	}

	return nil
}
