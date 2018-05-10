package envoy

import (
	"github.com/operator-framework/operator-sdk/pkg/sdk/action"
	"github.com/operator-framework/operator-sdk/pkg/sdk/query"

	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func syncService(e *api.Envoy) error {
	needService := len(e.Spec.ServicePorts) != 0
	// get the envoy deployment

	s := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.GetName(),
			Namespace: e.GetNamespace(),
		}}

	err := query.Get(s)

	if !needService {
		// not needed service exists - get rid of it:
		if err == nil {
			// TODO: should we confirm ownership?
			return action.Delete(s)
		}
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// service is needed and exists; make sure it is up-to-date
	if err == nil && needsUpdate(e, s) {
		return syncPorts(e, s)
	}

	if !apierrors.IsNotFound(err) {
		return err
	}
	// service doesnt exist: create it
	return createService(e)
}

func syncPorts(e *api.Envoy, s *v1.Service) error {
	setServicePorts(e, s)
	return action.Update(s)
}

func createService(e *api.Envoy) error {
	s := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.GetName(),
			Namespace: e.GetNamespace(),
			Labels:    labelsForEnvoy(e),
		},
		Spec: v1.ServiceSpec{
			Selector: labelsForEnvoy(e),
			Type:     v1.ServiceTypeLoadBalancer,
		},
	}

	addOwnerRefToObject(s, asOwner(&e.ObjectMeta))
	setServicePorts(e, s)
	return action.Create(s)
}

func setServicePorts(e *api.Envoy, s *v1.Service) {
	s.Spec.Ports = nil
	for name, port := range e.Spec.ServicePorts {
		// TODO: support IANA ports?
		s.Spec.Ports = append(s.Spec.Ports, v1.ServicePort{
			Name:     name,
			Port:     port,
			Protocol: v1.ProtocolTCP,
		})
	}
}

func needsUpdate(e *api.Envoy, s *v1.Service) bool {
	if len(e.Spec.ServicePorts) != len(s.Spec.Ports) {
		return true
	}
	for _, p := range s.Spec.Ports {
		if p.Protocol != v1.ProtocolTCP {
			return true
		}
		servicePort, ok := e.Spec.ServicePorts[p.Name]
		if !ok{
			return true
		}
		if servicePort != p.Port {
			return true
		}
	}
	return false
}