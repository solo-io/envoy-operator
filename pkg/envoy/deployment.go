package envoy

import (
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/sdk/action"
	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"

	"github.com/solo-io/envoy-operator/pkg/downward"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"github.com/pkg/errors"
	"github.com/operator-framework/operator-sdk/pkg/sdk/query"
	"fmt"
	"reflect"
	"log"
	"time"
)

const (
	initContainerImage = "soloio/envoy-operator-init:0.1"

	downwardVolName = "downward-api-volume"
	downwardVolPath = "/etc/podinfo/"

	envoyConfigVolName = "envoy-config"
	envoyConfigPath    = "/etc/tmp-envoy/"

	envoyConfigTmpVolName = "envoy-tmp-config"
	envoyConfigTmpPath    = "/etc/envoy/"

	// Config map mounts are readonly, so we have to move the transformed config to a different place...
	envoyConfigFilePath       = "/etc/envoy/envoy.json"
	envoySourceConfigFilePath = "/etc/tmp-envoy/envoy.json"

	envoyTLSVolName = "tls-certs"
)

// restart pods in the event the configmap changed buy the deployment hasn't
func reconcileEnvoyDeployment(restartPods bool, envoy *api.Envoy) error {
	desired, err := desiredDeployment(envoy)
	if err != nil {
		return err
	}

	existing := &appsv1.Deployment{
		TypeMeta:   desired.TypeMeta,
		ObjectMeta: desired.ObjectMeta,
	}
	if err := query.Get(existing); err == nil {
		if deploymentEqualForOurPurposes(*desired, *existing) {
			if restartPods {
				reps := desired.Spec.Replicas
				intVar := int32(0)
				desired.Spec.Replicas = &intVar
				//TODO: support hot restarts
				log.Printf("restarting pod because configmap changed")
				desired.ResourceVersion = existing.ResourceVersion
				if err := action.Update(desired); err != nil {
					return errors.Wrap(err, "setting replicas to 0")
				}
				time.Sleep(time.Second)
				if err := query.Get(existing, query.WithGetOptions(&metav1.GetOptions{
					TypeMeta: existing.TypeMeta,
					ResourceVersion: existing.ResourceVersion,
				})); err != nil {
					return errors.Wrap(err, "getting updated resource version")
				}
				desired.ResourceVersion = existing.ResourceVersion
				desired.Spec.Replicas = reps
				if err := action.Update(desired); err != nil {
					return errors.Wrap(err, "restoring replicas")
				}
			}
			return nil
		}
		if !ownedBy(existing, envoy.ObjectMeta) {
			log.Printf("Warning: an identical deployment exists that is not owned by this crd")
			return nil
		}
		log.Printf("updating deployment %v", desired.Name)
		desired.ResourceVersion = existing.ResourceVersion
		return action.Update(desired)
	}

	log.Printf("creating deployment %v", desired.Name)
	if err := action.Create(desired); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("prepare envoy config error: create new deployment (%s) failed: %v", desired.Name, err)
	}
	return nil
}

func desiredDeployment(envoy *api.Envoy) (*appsv1.Deployment, error) {
	volumes := []v1.Volume{{
		Name: envoyConfigVolName,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: configMapNameForEnvoy(envoy),
				},
			},
		},
	}, {
		Name: envoyConfigTmpVolName,
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{},
		},
	},
	}

	if envoy.Spec.TLSSecretName != "" {
		volumes = append(volumes, v1.Volume{
			Name: envoyTLSVolName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: envoy.Spec.TLSSecretName,
				},
			},
		})

	}

	downwardApiVolumes, env, err := createDownwardApiConfig(envoy)
	if err != nil {
		return nil, errors.Wrap(err, "generating downward api configuration")
	}
	volumes = append(volumes, downwardApiVolumes...)
	downwardVolNeeded := len(downwardApiVolumes) != 0

	selector := labelsForEnvoy(envoy)

	podTempl := v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      envoy.GetName(),
			Namespace: envoy.GetNamespace(),
			Labels:    selector,
		},
		Spec: v1.PodSpec{
			InitContainers: []v1.Container{configInitContainer(env, downwardVolNeeded)},
			Containers:     []v1.Container{envoyContainer(envoy)},
			Volumes:        volumes,
		},
	}

	replicas := int32(envoy.Spec.Deployment.Replicas)

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      envoy.GetName(),
			Namespace: envoy.GetNamespace(),
			Labels:    selector,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: podTempl,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: func(a intstr.IntOrString) *intstr.IntOrString { return &a }(intstr.FromInt(1)),
					MaxSurge:       func(a intstr.IntOrString) *intstr.IntOrString { return &a }(intstr.FromInt(1)),
				},
			},
		},
	}

	addOwnerRefToObject(deployment, ownerRef(envoy.ObjectMeta))

	return deployment, nil
}

func labelsForEnvoy(envoy *api.Envoy) map[string]string {
	return map[string]string{"app": "envoy", "envoy_cluster": envoy.Name}
}

func addEnv(name, ref string) v1.EnvVar {
	return v1.EnvVar{
		Name: name,
		ValueFrom: &v1.EnvVarSource{
			FieldRef: &v1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath: ref,
			},
		},
	}
}

func addVolumes(isPodLabels, isPodAnnotations bool) v1.Volume {
	var items []v1.DownwardAPIVolumeFile
	if isPodLabels {
		items = append(items, v1.DownwardAPIVolumeFile{
			Path: "labels",
			FieldRef: &v1.ObjectFieldSelector{
				FieldPath: "metadata.labels",
			},
		})
	}
	if isPodAnnotations {
		items = append(items, v1.DownwardAPIVolumeFile{
			Path: "annotations",
			FieldRef: &v1.ObjectFieldSelector{
				FieldPath: "metadata.annotations",
			},
		})
	}

	return v1.Volume{
		Name: downwardVolName,
		VolumeSource: v1.VolumeSource{
			DownwardAPI: &v1.DownwardAPIVolumeSource{
				Items: items,
			},
		},
	}
}

func createDownwardApiConfig(envoy *api.Envoy) ([]v1.Volume, []v1.EnvVar, error) {
	whatsNeeded := downward.TetsNeededDownwardAPI()
	interpolate := downward.NewInterpolator()
	_, err := interpolate.InterpolateString(envoy.Spec.NodeIdTemplate, whatsNeeded)
	if err != nil {
		return nil, nil, errors.Wrap(err, "interpolating node id")
	}
	_, err = interpolate.InterpolateString(envoy.Spec.ClusterIdTemplate, whatsNeeded)
	if err != nil {
		return nil, nil, errors.Wrap(err, "interpolating cluster id")
	}

	var volumes []v1.Volume
	downwardVolNeeded := whatsNeeded.IsPodAnnotations || whatsNeeded.IsPodLabels
	if downwardVolNeeded {
		volumes = append(volumes, addVolumes(whatsNeeded.IsPodLabels, whatsNeeded.IsPodAnnotations))
	}
	var env []v1.EnvVar
	if whatsNeeded.IsPodName {
		env = append(env, addEnv("POD_NAME", "metadata.name"))
	}
	if whatsNeeded.IsPodNamespace {
		env = append(env, addEnv("POD_NAMESPACE", "metadata.namespace"))
	}
	if whatsNeeded.IsPodIp {
		env = append(env, addEnv("POD_IP", "status.podIp"))
	}
	if whatsNeeded.IsPodSvcAccount {
		env = append(env, addEnv("POD_SVCACCNT", "spec.serviceAccountName"))
	}
	if whatsNeeded.IsPodNamespace {
		env = append(env, addEnv("POD_UID", "metadata.uid"))
	}
	if whatsNeeded.IsNodeName {
		env = append(env, addEnv("NODE_NAME", "spec.nodeName"))
	}
	if whatsNeeded.IsNodeIp {
		env = append(env, addEnv("NODE_IP", "status.hostIP"))
	}
	return volumes, env, nil
}

func envoyContainer(envoy *api.Envoy) v1.Container {
	vmounts := []v1.VolumeMount{{
		Name:      envoyConfigTmpVolName,
		MountPath: filepath.Dir(envoyConfigTmpPath),
	}}

	var ports []v1.ContainerPort
	if envoy.Spec.AdminPort != 0 {
		ports = append(ports, v1.ContainerPort{
			ContainerPort: envoy.Spec.AdminPort,
			Name:          "admin",
			Protocol:      v1.ProtocolTCP,
		})
	}

	if envoy.Spec.TLSSecretName != "" {
		vmounts = append(vmounts, v1.VolumeMount{
			Name:      envoyTLSVolName,
			MountPath: filepath.Dir(api.EnvoyTLSVolPath),
		})
	}

	return v1.Container{
		Name:    "envoy",
		Image:   envoy.Spec.Image,
		Command: envoy.Spec.ImageCommand,
		Args: []string{
			"-c", envoyConfigFilePath, "--v2-config-only",
		},
		VolumeMounts:             vmounts,
		Ports:                    ports,
		ImagePullPolicy:          "IfNotPresent",
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: v1.TerminationMessageReadFile,
	}
}

func configInitContainer(env []v1.EnvVar, downwardVol bool) v1.Container {
	vmounts := []v1.VolumeMount{{
		Name:      envoyConfigVolName,
		MountPath: filepath.Dir(envoyConfigPath),
	}, {
		Name:      envoyConfigTmpVolName,
		MountPath: filepath.Dir(envoyConfigTmpPath),
	}}

	if downwardVol {
		vmounts = append(vmounts, v1.VolumeMount{
			Name:      downwardVolName,
			MountPath: filepath.Dir(downwardVolPath),
		})
	}

	return v1.Container{
		Name:  "envoy-init",
		Image: initContainerImage,
		Args: []string{
			"-input",
			envoySourceConfigFilePath,
			"-output",
			envoyConfigFilePath,
		},
		Env:                      env,
		VolumeMounts:             vmounts,
		ImagePullPolicy:          "IfNotPresent",
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: v1.TerminationMessageReadFile,
	}
}

func deploymentEqualForOurPurposes(dep1, dep2 appsv1.Deployment) bool {
	if !reflect.DeepEqual(dep1.Labels, dep2.Labels) {
		return false
	}
	if !reflect.DeepEqual(dep1.Spec.Selector, dep2.Spec.Selector) {
		return false
	}
	vols1 := dep1.Spec.Template.Spec.Volumes
	vols2 := dep2.Spec.Template.Spec.Volumes
	if len(vols1) != len(vols2) {
		return false
	}
	for i := range vols1 {
		vol1 := vols1[i]
		vol2 := vols2[i]
		if vol1.Name != vol2.Name {
			return false
		}
		if vol1.VolumeSource.DownwardAPI != nil {
			if vol2.VolumeSource.DownwardAPI == nil {
				return false
			}
			downItems1 := vol1.VolumeSource.DownwardAPI.Items
			downItems2 := vol2.VolumeSource.DownwardAPI.Items
			if len(downItems1) != len(downItems2) {
				return false
			}
			for j := range downItems1 {
				item1 := downItems1[j]
				item2 := downItems2[j]
				if item1.Path != item2.Path {
					return false
				}
				if item1.FieldRef.FieldPath != item2.FieldRef.FieldPath {
					return false
				}
			}
		}
		if vol1.VolumeSource.ConfigMap != nil {
			if vol2.VolumeSource.ConfigMap == nil {
				return false
			}
			if vol1.VolumeSource.ConfigMap.LocalObjectReference.Name != vol2.VolumeSource.ConfigMap.LocalObjectReference.Name {
				return false
			}
		}
		if vol1.VolumeSource.Secret != nil {
			if vol2.VolumeSource.Secret == nil {
				return false
			}
			if vol1.VolumeSource.Secret.SecretName != vol2.VolumeSource.Secret.SecretName {
				return false
			}
		}
	}
	if !reflect.DeepEqual(dep1.Spec.Template.Spec.InitContainers, dep2.Spec.Template.Spec.InitContainers) {
		return false
	}
	if !reflect.DeepEqual(dep1.Spec.Template.Spec.Containers, dep2.Spec.Template.Spec.Containers) {
		return false
	}
	return true
}
