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
)

const (
	initContainerImage = "soloio/envoy-operator-init:0.1"

	downwardVolName = "downward-api-volume"
	downwardVolPath = "/etc/podinfo/"

	envoyConfigVolName = "envoy-config"
	envoyConfigPath    = "/etc/tmp-envoy/"

	envoyConfigTmpVolName = "envoy-tmp-config"
	envoyConfigTmpPath    = "/etc/envoy/"

	envoyConfigFilePath       = "/etc/envoy/envoy.json"
	envoySourceConfigFilePath = "/etc/tmp-envoy/envoy.json"
)

func deployEnvoy(e *api.Envoy) error {

	whatsNeeded := downward.TetsNeededDownwardAPI()
	interpolate := downward.NewInterpolator()
	_, err := interpolate.InterpolateString(e.Spec.NodeIdTemplate, whatsNeeded)
	if err != nil {
		return err
	}
	_, err = interpolate.InterpolateString(e.Spec.ClusterIdTemplate, whatsNeeded)
	if err != nil {
		return err
	}

	volumes := []v1.Volume{{
		Name: envoyConfigVolName,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: configMapNameForEnvoy(e),
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
	selector := labelsForEnvoy(e.GetName())

	podTempl := v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.GetName(),
			Namespace: e.GetNamespace(),
			Labels:    selector,
		},
		Spec: v1.PodSpec{
			InitContainers: []v1.Container{configInitContainer(e, env, volumes, downwardVolNeeded)},
			Containers:     []v1.Container{envoyContainer(e)},
			Volumes:        volumes,
		},
	}

	var reps int32
	reps = int32(e.Spec.Deployment.Replicas)

	d := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.GetName(),
			Namespace: e.GetNamespace(),
			Labels:    selector,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &reps,
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

	addOwnerRefToObject(d, asOwner(&e.ObjectMeta))
	err = action.Create(d)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func labelsForEnvoy(name string) map[string]string {
	return map[string]string{"app": "envoy", "envoy_cluster": name}
}

func addEnv(name, ref string) v1.EnvVar {
	return v1.EnvVar{
		Name: name,
		ValueFrom: &v1.EnvVarSource{
			FieldRef: &v1.ObjectFieldSelector{
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

func envoyContainer(e *api.Envoy) v1.Container {

	vmounts := []v1.VolumeMount{{
		Name:      envoyConfigVolName,
		MountPath: filepath.Dir(envoyConfigPath),
	}}

	var ports []v1.ContainerPort
	if e.Spec.AdminPort != 0 {
		ports = append(ports, v1.ContainerPort{
			ContainerPort: e.Spec.AdminPort,
			Name:          "admin",
		})
	}

	return v1.Container{
		Name:  "envoy",
		Image: e.Spec.Image,
		// TODO: figure out the args needed if dumb init is used.
		Args: []string{
			"-c", envoyConfigFilePath, "--v2-config-only",
		},
		VolumeMounts: vmounts,
		Ports:        ports,
	}
}

func configInitContainer(v *api.Envoy, env []v1.EnvVar, volumes []v1.Volume, downwardvol bool) v1.Container {

	vmounts := []v1.VolumeMount{{
		Name:      envoyConfigVolName,
		MountPath: filepath.Dir(envoyConfigPath),
	}, {
		Name:      envoyConfigTmpVolName,
		MountPath: filepath.Dir(envoyConfigTmpPath),
	}}

	if downwardvol {
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
		Env:          env,
		VolumeMounts: vmounts,
	}
}

func configMapNameForEnvoy(e *api.Envoy) string { return e.Name }

func addOwnerRefToObject(o metav1.Object, r metav1.OwnerReference) {
	o.SetOwnerReferences(append(o.GetOwnerReferences(), r))
}

func asOwner(e *metav1.ObjectMeta) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: api.SchemeGroupVersion.String(),
		Kind:       api.EnvoyServiceKind,
		Name:       e.Name,
		UID:        e.UID,
		Controller: &trueVar,
	}
}
