package v1alpha1

import (
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
	NatsStreamingImage string `json:"image"`

	// NatsService is the Kubernetes service to which the NATS Streaming nodes will connect.
	// The service has to be in the same namespace as the NATS Operator.
	NatsService string `json:"natsSvc"`
}

type NatsStreamingClusterStatus struct {
	// TODO
}
