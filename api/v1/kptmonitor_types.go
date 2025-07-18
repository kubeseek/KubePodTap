/*
Copyright 2023 The Kubernetes Authors.

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

// KptMonitorSpec defines the desired state of KptMonitor
type KptMonitorSpec struct {
	// Namespace specifies the namespace where target pods are located
	Namespace string `json:"namespace"`

	// TargetPods specifies the pods to be monitored
	TargetPods string `json:"targetPods"`

	// TapDuration specifies the duration for monitoring
	TapDuration string `json:"tapDuration"`
}

// KptMonitorStatus defines the observed state of KptMonitor
type KptMonitorStatus struct {
	// Status indicates the current status of the monitoring
	// Valid values: "Created", "Monitoring", "Failed"
	Status string `json:"status,omitempty"`

	// Message provides detailed information about the current status
	Message string `json:"message,omitempty"`

	// LastUpdated is the timestamp when the status was last updated
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.namespace`
//+kubebuilder:printcolumn:name="TargetPods",type=string,JSONPath=`.spec.targetPods`
//+kubebuilder:printcolumn:name="TapDuration",type=string,JSONPath=`.spec.tapDuration`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// KptMonitor is the Schema for the kptmonitors API
type KptMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KptMonitorSpec   `json:"spec,omitempty"`
	Status KptMonitorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KptMonitorList contains a list of KptMonitor
type KptMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KptMonitor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KptMonitor{}, &KptMonitorList{})
}
