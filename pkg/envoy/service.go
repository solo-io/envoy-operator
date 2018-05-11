package envoy

import (
	"github.com/operator-framework/operator-sdk/pkg/sdk/action"
	"github.com/operator-framework/operator-sdk/pkg/sdk/query"

	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"fmt"
	"log"
)

func reconcileEnvoyService(envoy *api.Envoy) error {
	desired := desiredService(envoy)

	existing := &v1.Service{
		TypeMeta:   desired.TypeMeta,
		ObjectMeta: desired.ObjectMeta,
	}

	needService := len(envoy.Spec.ServicePorts) != 0

	if err := query.Get(existing); err == nil {
		// we no longer need this service, remove it
		if !needService {
			log.Printf("deleting service %v", desired.Name)
			return action.Delete(existing)
		}
		if serviceEqualForOurPurposes(desired, existing) {
			return nil
		}
		if !ownedBy(existing, envoy.ObjectMeta) {
			log.Printf("Warning: an identical service exists that is not owned by this crd")
			return nil
		}
		log.Printf("updating service %v", desired.Name)
		desired.Spec.ClusterIP = existing.Spec.ClusterIP
		desired.ResourceVersion = existing.ResourceVersion
		return action.Update(desired)
	} else if !needService && apierrors.IsNotFound(err) {
		// race happened, service was already removed
		return nil
	}

	log.Printf("creating service %v", desired.Name)
	if err := action.Create(desired); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("prepare envoy config error: create new service (%s) failed: %v", desired.Name, err)
	}
	return nil
}

func desiredService(envoy *api.Envoy) *v1.Service {
	s := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      envoy.GetName(),
			Namespace: envoy.GetNamespace(),
			Labels:    labelsForEnvoy(envoy),
		},
		Spec: v1.ServiceSpec{
			Selector: labelsForEnvoy(envoy),
			Type:     v1.ServiceTypeLoadBalancer,
		},
	}

	addOwnerRefToObject(s, ownerRef(envoy.ObjectMeta))
	setServicePorts(envoy, s)
	return s
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

func serviceEqualForOurPurposes(s1, s2 *v1.Service) bool {
	if len(s1.Spec.Ports) != len(s2.Spec.Ports) {
		return false
	}
	for i := range s1.Spec.Ports {
		p1 := s1.Spec.Ports[i]
		p2 := s2.Spec.Ports[i]
		if p1.Protocol != p2.Protocol {
			return false
		}
		if p1.Port != p2.Port {
			return false
		}
	}
	return true
}