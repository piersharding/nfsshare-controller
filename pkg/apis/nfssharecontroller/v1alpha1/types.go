/*
Copyright 2017 The Kubernetes Authors.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Nfsshare is a specification for a Nfsshare resource
type Nfsshare struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NfsshareSpec   `json:"spec"`
	Status NfsshareStatus `json:"status"`
}

// NfsshareSpec is the spec for a Nfsshare resource
type NfsshareSpec struct {
	ShareName       string `json:"shareName"`
    Image           string `json:"image"`
    Replicas        *int32 `json:"replicas,omitempty"`
    SharedDirectory string `json:"sharedDirectory"`
    Size            string `json:"size"`
    StorageClass    string `json:"storageClass"`
}

// NfsshareStatus is the status for a Nfsshare resource
type NfsshareStatus struct {
	AvailableReplicas int32 `json:"availableReplicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NfsshareList is a list of Nfsshare resources
type NfsshareList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Nfsshare `json:"items"`
}
