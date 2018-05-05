package envoy

import (
	api "github.com/solo-io/envoy-operator/pkg/apis/envoy/v1alpha1"
	"github.com/solo-io/envoy-operator/pkg/downward"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const initContainerImage = "soloio/init-envoy:0.1"

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

	podTempl := v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.GetName(),
			Namespace: e.GetNamespace(),
			// Labels:    selector,
		},
		Spec: v1.PodSpec{
			InitContainers: []v1.Container{configInitContainer(e, env, downwardVolNeeded)},
			// Containers:     []v1.Container{vaultContainer(v), statsdExporterContainer()},
			Volumes: volumes,
			SecurityContext: &v1.PodSecurityContext{
				RunAsUser:    func(i int64) *int64 { return &i }(9000),
				RunAsNonRoot: func(b bool) *bool { return &b }(true),
				FSGroup:      func(i int64) *int64 { return &i }(9000),
			},
		},
	}
	// TODO continue this
	podTempl = podTempl

	return nil
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
		Name: "downward-api-volume",
		VolumeSource: v1.VolumeSource{
			DownwardAPI: &v1.DownwardAPIVolumeSource{
				Items: items,
			},
		},
	}
}

func configInitContainer(v *api.Envoy, env []v1.EnvVar, downwardvol bool) v1.Container {
	// ARGS = infile in the configmap
	// and outfile in the config map for envoy

	/*
	   map the downward api to /etc/podinfo
	   map the env vars to env vars

	   run: initialize -infile /tmp/envoy-config -outfile /etc/envoy/config.yaml

	*/

	panic("TODO")
}
