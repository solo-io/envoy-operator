package envoy

import (
	"fmt"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/sdk/action"
	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
	"github.com/solo-io/envoy-operator/pkg/kube"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func prepareEnvoyConfig(e *api.Envoy) error {
	cfgData, err := kube.GenerateEnvoyConfig(e)
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
		return fmt.Errorf("prepare vault config error: create new configmap (%s) failed: %v", cm.Name, err)
	}
	return nil
}
