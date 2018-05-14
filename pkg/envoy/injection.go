package envoy

import (
	"fmt"
	//"log"

	"k8s.io/api/core/v1"
	"github.com/operator-framework/operator-sdk/pkg/sdk/query"
	appsv1 "k8s.io/api/apps/v1"
	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
	"github.com/pkg/errors"
	"github.com/mitchellh/hashstructure"
	"github.com/operator-framework/operator-sdk/pkg/sdk/action"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/solo-io/gloo/pkg/log"
)

func reconcileEnvoyInjection(namespace string, envoy *api.Envoy) error {
	if envoy.Spec.Injection == nil {
		return DeleteEnvoyInjection(namespace, envoy)
	}

	deployments := &appsv1.DeploymentList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
	}
	err := query.List(namespace, deployments)
	if err != nil {
		return errors.Wrap(err, "failed to list existing deployments")
	}

	for _, deployment := range deployments.Items {
		if deployment.Name == envoy.Name || deployment.Name == "envoy-operator" {
			// don't inject those
			continue
		}
		if err := updateDeploymentInjection(deployment, envoy); err != nil {
			return errors.Wrapf(err, "failed to inject deployment %v", deployment.Name)
		}
	}

	return nil
}

func updateDeploymentInjection(deployment appsv1.Deployment, envoy *api.Envoy) error {
	initContainer, sidecarContainer, envoyVolumes, err := createInjectableResources(envoy)
	if err != nil {
		return errors.Wrap(err, "creating injectable resources")
	}
	podSpec := deployment.Spec.Template.Spec

	// init container
	initContainers, initUpdated := updateContainers(podSpec.InitContainers, *initContainer)

	// sidecar
	containers, containersUpdated := updateContainers(podSpec.Containers, *sidecarContainer)

	// volumes
	volumes, volumesUpdated := updateVolumes(podSpec.Volumes, envoyVolumes)

	if !(initUpdated || containersUpdated || volumesUpdated) {
		// nothing changed
		return nil
	}

	podSpec.InitContainers = initContainers
	podSpec.Containers = containers
	podSpec.Volumes = volumes

	deployment.Spec.Template.Spec = podSpec

	log.Printf("injecting deployment %v: %v", deployment.Name, deployment.Spec.Template.Spec)
	if err := action.Update(&deployment); err != nil {
		return errors.Wrap(err, "updating deployment")
	}
	log.Printf("injecting deployment %v successful", deployment.Name)
	return nil
}

func DeleteEnvoyInjection(namespace string, envoy *api.Envoy) error {
	var deployments appsv1.DeploymentList
	err := query.List(namespace, &deployments)
	if err != nil {
		return errors.Wrap(err, "failed to list existing deployments")
	}

	for _, deployment := range deployments.Items {
		if deployment.Name == envoy.Name {
			// this is a standalone pod, don't inject those
			continue
		}
		if err := deleteDeploymentInjection(deployment); err != nil {
			return errors.Wrapf(err, "failed to un-inject deployment %v", deployment.Name)
		}
	}

	return nil
}

func deleteDeploymentInjection(deployment appsv1.Deployment) error {
	podSpec := deployment.Spec.Template.Spec

	var modified bool
	podSpec.InitContainers, modified = deleteContainer(podSpec.InitContainers, initContainerName)

	containers, mod := deleteContainer(podSpec.Containers, envoyContainerName)
	podSpec.Containers = containers
	modified = modified || mod

	volumes, mod := deleteVolume(podSpec.Volumes, downwardVolName)
	podSpec.Volumes = volumes
	modified = modified || mod

	volumes, mod = deleteVolume(podSpec.Volumes, envoyTLSVolName)
	podSpec.Volumes = volumes
	modified = modified || mod

	volumes, mod = deleteVolume(podSpec.Volumes, envoyConfigTmpVolName)
	podSpec.Volumes = volumes
	modified = modified || mod

	volumes, mod = deleteVolume(podSpec.Volumes, envoyConfigVolName)
	podSpec.Volumes = volumes
	modified = modified || mod

	if !modified {
		return nil
	}

	deployment.Spec.Template.Spec = podSpec

	log.Printf("un-injecting deployment %v", deployment.Name)
	return action.Update(&deployment)
}

func deleteContainer(containers []v1.Container, name string) ([]v1.Container, bool) {
	var modified bool
	for i, c := range containers {
		if c.Name == name {
			containers = append(containers[:i], containers[i+1:]...)
			modified = true
			break
		}
	}
	return containers, modified
}

func deleteVolume(volumes []v1.Volume, name string) ([]v1.Volume, bool) {
	var modified bool
	for i, c := range volumes {
		if c.Name == name {
			volumes = append(volumes[:i], volumes[i+1:]...)
			modified = true
			break
		}
	}
	return volumes, modified
}

func createInjectableResources(envoy *api.Envoy) (*v1.Container, *v1.Container, []v1.Volume, error) {
	baseDeployment, err := desiredDeployment(envoy)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "creating deployment from envoy api object")
	}
	podSpec := baseDeployment.Spec.Template.Spec
	if len(podSpec.InitContainers) != 1 {
		return nil, nil, nil, errors.Wrap(err, "missing init container")
	}
	if len(podSpec.Containers) != 1 {
		return nil, nil, nil, errors.Wrap(err, "missing envoy container")
	}
	if len(podSpec.Volumes) != 2 {
		return nil, nil, nil, errors.Wrap(err, "missing envoy volumes")
	}
	return &podSpec.InitContainers[0], &podSpec.Containers[0], podSpec.Volumes, nil
}

func updateContainers(theirContainers []v1.Container, ourContainer v1.Container) ([]v1.Container, bool) {
	// check if our init container is already there
	for i, c := range theirContainers {
		if hashContainer(c) == hashContainer(ourContainer) {
			// found ours and it matches desired
			return theirContainers, false
		}
		// update if the container already exists, but doesn't match ours
		if c.Name == ourContainer.Name {
			// replace our old container with our new one
			return append(append(theirContainers[:i], ourContainer), theirContainers[i+1:]...), true
		}
	}
	// our init container isn't there yet, append it
	return append(theirContainers, ourContainer), true
}

func hashContainer(c v1.Container) uint64 {
	env := make(map[string]string)
	for _, e := range c.Env {
		if e.ValueFrom != nil && e.ValueFrom.FieldRef != nil {
			env[e.Name] = e.ValueFrom.FieldRef.FieldPath
			continue
		}
		env[e.Name] = e.Value
	}
	var vmounts []struct {
		Name      string
		MountPath string
	}
	for _, vm := range c.VolumeMounts {
		vmounts = append(vmounts, struct {
			Name      string
			MountPath string
		}{
			Name:      vm.Name,
			MountPath: vm.MountPath,
		})
	}
	data := struct {
		Name  string
		Image string
		Args  []string
		Env   map[string]string
		VolumeMounts []struct {
			Name      string
			MountPath string
		}
	}{
		Name:         c.Name,
		Image:        c.Image,
		Args:         c.Args,
		Env:          env,
		VolumeMounts: vmounts,
	}
	hash, err := hashstructure.Hash(data, nil)
	if err != nil {
		panic(err)
	}
	return hash
}

func updateVolumes(theirVolumes, ourVolumes []v1.Volume) ([]v1.Volume, bool) {
	var volumesUpdated bool
	for _, vol := range ourVolumes {
		vols, updated := reconcileVolume(theirVolumes, vol)
		theirVolumes = vols
		volumesUpdated = volumesUpdated || updated
	}
	return theirVolumes, volumesUpdated
}

func reconcileVolume(theirVolumes []v1.Volume, ourVolume v1.Volume) ([]v1.Volume, bool) {
	// check if our init container is already there
	for i, v := range theirVolumes {
		if hashVolume(v) == hashVolume(ourVolume) {
			// found ours and it matches desired
			return theirVolumes, false
		}
		// update if the volume already exists, but doesn't match ours
		if v.Name == ourVolume.Name {
			// replace our old container with our new one
			return append(append(theirVolumes[:i], ourVolume), theirVolumes[i+1:]...), true
		}
	}
	// our init container isn't there yet, append it
	return append(theirVolumes, ourVolume), true
}

func hashVolume(v v1.Volume) uint64 {
	data := struct {
		Name         string
		VolumeSource string
	}{
		Name:         v.Name,
		VolumeSource: volumeSourceName(v.VolumeSource),
	}
	hash, err := hashstructure.Hash(data, nil)
	if err != nil {
		panic(err)
	}
	return hash
}

func volumeSourceName(vs v1.VolumeSource) string {
	switch {
	case vs.ConfigMap != nil:
		return fmt.Sprintf("configmap-%s", vs.ConfigMap.Name)
	case vs.EmptyDir != nil:
		return fmt.Sprintf("emptydir")
	case vs.Secret != nil:
		return fmt.Sprintf("secret-%s", vs.Secret.SecretName)
	}
	return "ignored"
}
