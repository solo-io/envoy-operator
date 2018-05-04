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
}

type EnvoyStatus struct {
	// Fill me
}
