package envoy

import (
	"fmt"
	"reflect"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/sdk/action"
	"github.com/operator-framework/operator-sdk/pkg/sdk/query"

	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
	"github.com/solo-io/envoy-operator/pkg/kube"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/pkg/errors"
	"log"
)

func reconcileEnvoyConfigMap(envoy *api.Envoy) (bool, error) {
	desired, err := desiredConfigMap(envoy)
	if err != nil {
		return false, err
	}

	existing := &v1.ConfigMap{
		TypeMeta:   desired.TypeMeta,
		ObjectMeta: desired.ObjectMeta,
	}
	err = query.Get(existing)
	if err == nil {
		if cmEqualForOurPurposes(desired, existing) {
			return false, nil
		}
		if !ownedBy(existing, envoy.ObjectMeta) {
			log.Printf("Warning: an identical config map exists that is not owned by this crd")
			return false, nil
		}
		log.Printf("updating configmap %v", desired.Name)
		desired.ResourceVersion = existing.ResourceVersion
		return true, action.Update(desired)
	}

	log.Printf("creating configmap %v (err was %v)", desired.Name, err)
	if err := action.Create(desired); err != nil && !apierrors.IsAlreadyExists(err) {
		return false, fmt.Errorf("prepare envoy config error: create new configmap (%s) failed: %v", desired.Name, err)
	}
	return false, nil
}

func desiredConfigMap(envoy *api.Envoy) (*v1.ConfigMap, error) {
	var tlsSecret *v1.Secret
	if envoy.Spec.TLSSecretName != "" {
		sec := &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      envoy.Spec.TLSSecretName,
				Namespace: envoy.Namespace,
			},
		}
		if err := query.Get(sec); err != nil {
			return nil, errors.Wrap(err, "getting tls secret")
		}
		tlsSecret = sec
	}

	cfgData, err := kube.GenerateEnvoyConfig(envoy, tlsSecret)
	if err != nil {
		return nil, errors.Wrap(err, "generating bootstrap envoy config file")
	}

	cm := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: envoy.Namespace,
			Name:      configMapNameForEnvoy(envoy),
			Labels:    labelsForEnvoy(envoy),
		},
		Data: map[string]string{filepath.Base(envoyConfigFilePath): cfgData},
	}

	addOwnerRefToObject(cm, ownerRef(envoy.ObjectMeta))
	return cm, nil
}

func cmEqualForOurPurposes(cm1, cm2 *v1.ConfigMap) bool {
	return reflect.DeepEqual(cm1.Data, cm2.Data)
}
