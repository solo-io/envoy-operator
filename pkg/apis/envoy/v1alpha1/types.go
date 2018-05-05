package v1alpha1

// go:generate operator-sdk generate k8s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type EnvoyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Envoy `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Envoy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              EnvoySpec   `json:"spec"`
	Status            EnvoyStatus `json:"status,omitempty"`
}

type EnvoySpec struct {
	// Fill me

	ADSServer string

	Image string

	// ca, and potentiallyu client cert
	ClientTLS string

	AdminPort int32

	ClusterIdTemplate string

	NodeIdTemplate string

	StatsdSink string

	OpenTracing string

	Deployment EnvoyDeploymentSpec
	Injection  InjectionSpec
}

type EnvoyDeploymentSpec struct {
	Replicas uint32
}

type InjectionSpec struct {
	// 	Namespace string
}

type EnvoyStatus struct {
	// Fill me
}
