/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ServiceSpec struct {
	// +optional
	// +kubebuilder:default:="ClusterIp"
	// +kubebuilder:validation:enum=ClusterIp,NodePort,LoadBalancer
	Type corev1.ServiceType `json:"type,omitempty"`
}

// NacosStandaloneSpec defines the desired state of NacosStandalone
type NacosStandaloneSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	Image string `json:"image,omitempty"`
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty" patchStrategy:"merge" patchMergeKey:"name"`
	// +optional
	Service ServiceSpec `json:"service,omitempty"`
	// +optional
	Pvc *corev1.PersistentVolumeClaimSpec `json:"pvc,omitempty"`
}

// NacosStandaloneStatus defines the observed state of NacosStandalone
type NacosStandaloneStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +optional
	Nodes []string `json:"nodes,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NacosStandalone is the Schema for the nacosstandalones API
type NacosStandalone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NacosStandaloneSpec   `json:"spec,omitempty"`
	Status NacosStandaloneStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NacosStandaloneList contains a list of NacosStandalone
type NacosStandaloneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NacosStandalone `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NacosStandalone{}, &NacosStandaloneList{})
}
