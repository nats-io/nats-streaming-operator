// Copyright 2018 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NatsStreamingClusterList
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NatsStreamingClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []NatsStreamingCluster `json:"items"`
}

// NatsStreamingCluster
//
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NatsStreamingCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              NatsStreamingClusterSpec   `json:"spec"`
	Status            NatsStreamingClusterStatus `json:"status,omitempty"`
}

type NatsStreamingClusterSpec struct {
	// Size is the number of nodes in the NATS Streaming cluster.
	Size int32 `json:"size"`

	// Version is the version of NATS Streaming that is being used.
	// By default it will be the latest version.
	Image string `json:"image"`

	// NatsService is the Kubernetes service to which the NATS
	// Streaming nodes will connect. The service has to be in the
	// same namespace as the NATS Operator.
	NatsService  string               `json:"natsSvc"`
	Volume       []corev1.Volume      `json:"volume,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name" protobuf:"bytes,1,rep,name=volumes"`
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts"`
}

type NatsStreamingClusterStatus struct {
}
