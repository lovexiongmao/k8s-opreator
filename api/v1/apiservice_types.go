/*
Copyright 2025.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApiserviceSpec defines the desired state of Apiservice
type ApiserviceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of Apiservice. Edit apiservice_types.go to remove/update
	// +optional
	Foo *string `json:"foo,omitempty"`

	// 副本数
	Replicas int32 `json:"replicas,omitempty"`

	// 容器镜像
	Image string `json:"image"`

	// 端口
	Port int32 `json:"port"`

	// 环境变量
	Env []EnvVar `json:"env,omitempty"`

	// Test Field
	TestSpec string `json:"testSpec,omitempty"`

	// 资源限制
	Resources ResourceRequirements `json:"resources,omitempty"`
}

// ApiserviceStatus defines the observed state of Apiservice.
type ApiserviceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the Apiservice resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// 可用副本数
	AvailableReplicas int32 `json:"availableReplicas"`

	// 服务端点
	ServiceEndpoint string `json:"serviceEndpoint,omitempty"`

	// Test Status
	TestStatus string `json:"testStatus,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ResourceRequirements struct {
	Requests ResourceList `json:"requests,omitempty"`
	Limits   ResourceList `json:"limits,omitempty"`
}

type ResourceList struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Apiservice is the Schema for the apiservices API
type Apiservice struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Apiservice
	// +required
	Spec ApiserviceSpec `json:"spec"`

	// status defines the observed state of Apiservice
	// +optional
	Status ApiserviceStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ApiserviceList contains a list of Apiservice
type ApiserviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Apiservice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Apiservice{}, &ApiserviceList{})
}
