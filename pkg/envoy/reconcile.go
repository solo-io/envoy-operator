package envoy

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
	"log"

	"github.com/operator-framework/operator-sdk/pkg/sdk/action"
	"reflect"
)

// Reconcile reconciles the Envoy instance's state to the spec specified in the crd
func Reconcile(envoy *api.Envoy) (err error) {
	defer func() {
		if err != nil {
			log.Printf("Received error: %v\n", err)
		}
	}()
	envoy = envoy.DeepCopy()
	// Simulate initializer.
	changed := envoy.SetDefaults()
	if changed {
		return action.Update(envoy)
	}

	restartPods, err := reconcileEnvoyConfigMap(envoy)
	if err != nil {
		return err
	}

	err = reconcileEnvoyDeployment(restartPods, envoy)
	if err != nil {
		return err
	}

	err = reconcileEnvoyService(envoy)
	if err != nil {
		return err
	}

	return nil
}

func configMapNameForEnvoy(envoy *api.Envoy) string {
	return envoy.Name
}

func ownerRef(envoyMeta metav1.ObjectMeta) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: api.SchemeGroupVersion.String(),
		Kind:       api.EnvoyServiceKind,
		Name:       envoyMeta.Name,
		UID:        envoyMeta.UID,
		Controller: &trueVar,
	}
}

func addOwnerRefToObject(o metav1.Object, r metav1.OwnerReference) {
	o.SetOwnerReferences(append(o.GetOwnerReferences(), r))
}

func ownedBy(o metav1.Object, envoyMeta metav1.ObjectMeta) bool {
	ourRef := ownerRef(envoyMeta)
	for _, ref := range o.GetOwnerReferences() {
		 if reflect.DeepEqual(ref, ourRef) {
		 	return true
		 }
	}
	return false
}
