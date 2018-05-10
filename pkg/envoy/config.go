package envoy

import (
	"fmt"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/sdk/action"
	"github.com/operator-framework/operator-sdk/pkg/sdk/query"

	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
	"github.com/solo-io/envoy-operator/pkg/kube"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func prepareEnvoyConfig(e *api.Envoy) error {
	var tlsSecret *v1.Secret
	if e.Spec.TLSSecretName != "" {
		sec := &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      e.Spec.TLSSecretName,
				Namespace: e.Namespace,
			},
		}
		err := query.Get(sec)
		if err != nil {
			return err
		}
		tlsSecret = sec

	}

	cfgData, err := kube.GenerateEnvoyConfig(e, tlsSecret)
	if err != nil {
		return err
	}

	cm := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: e.Namespace,
			Name:      configMapNameForEnvoy(e),
		},
	}

	cm.Labels = labelsForEnvoy(e.Name)

	cm.Data = map[string]string{filepath.Base(envoyConfigFilePath): cfgData}
	addOwnerRefToObject(cm, asOwner(&e.ObjectMeta))

	// TODO: check if config map changed?
	if err := action.Create(cm); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("prepare envoy config error: create new configmap (%s) failed: %v", cm.Name, err)
	}
	return nil
}
