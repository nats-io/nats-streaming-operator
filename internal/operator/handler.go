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

package operator

import (
	"context"
	"fmt"
	"sync"

	"github.com/nats-io/nats-streaming-operator/pkg/apis/streaming/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func NewHandler() sdk.Handler {
	return &Handler{
		clusters: make(map[types.UID]*v1alpha1.NatsStreamingCluster),
	}
}

// Handler is the custom operator logic to handle NATS Streaming clusters.
type Handler struct {
	mu       sync.Mutex
	clusters map[types.UID]*v1alpha1.NatsStreamingCluster
}

// Handle is the reconcile loop from the operator when managing the
// nodes from a cluster.
func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	// Every sync seconds it will be looking for the related pods,
	// from each one of the NATS Streaming Clusters.
	switch o := event.Object.(type) {
	case *v1alpha1.NatsStreamingCluster:
		// In case the cluster has been deleted or not ready, then just skip the event.
		if o.DeletionTimestamp != nil {
			// Throwaway cluster and let garbage collection remove
			// the pods via ownership cascade delete.
			log.Debugf("Removing %v cluster", o.Name)
			h.mu.Lock()
			delete(h.clusters, o.UID)
			h.mu.Unlock()
			return nil
		}

		// Collect metadata from latest perceived version.
		defer func() {
			h.mu.Lock()
			h.clusters[o.UID] = o
			h.mu.Unlock()
		}()

		// Look up the related pods for this cluster.
		pods, err := findRunningPods(o.Name, o.Namespace)
		if err != nil {
			return err
		}

		// Check cluster health.
		n := len(pods) - int(o.Spec.Size)
		if n == 0 {
			log.Debugf("Cluster %q size reconciled (%d/%d)", o.Name, o.Spec.Size, o.Spec.Size)
			return nil
		} else if n > 0 {
			log.Infof("Too many pods for %q cluster (%d/%d), removing %d pods...", o.Name, len(pods), o.Spec.Size, n)
			return shrinkCluster(pods, n)
		} else if n < 0 {
			log.Infof("Missing pods for %q cluster (%d/%d), creating %d pods...", o.Name, len(pods), o.Spec.Size, n*-1)

			// If this is the first time we have perceived the pod,
			// then try to create the first node to bootstrap the
			// cluster.
			h.mu.Lock()
			if _, ok := h.clusters[o.UID]; !ok {
				h.mu.Unlock()
				return createBootstrapPod(o)
			}
			h.mu.Unlock()

			// If no other nodes are available, then create one with the bootstrap
			// flag so that it can become the leader.
			if n == 0 {
				return createBootstrapPod(o)
			}

			return createMissingPods(o, n*-1)
		}
	}
	return nil
}

func createBootstrapPod(o *v1alpha1.NatsStreamingCluster) error {
	pod := newStanPod(o)
	pod.Name = fmt.Sprintf("%s-1", o.Name)
	container := stanContainer(o)
	volume := stanVolume(o)
	container.Command = stanContainerBootstrapCmd(o)
	pod.Spec.Containers = []corev1.Container{container}
	pod.Spec.Volumes = []corev1.Volume(volume)
	log.Debugf("Creating pod %q", pod.Name)
	err := sdk.Create(pod)
	if err != nil && !errors.IsAlreadyExists(err) {
		log.Errorf("Failed to create bootstrap Pod: %v", err)
		return err
	}

	return nil
}
func stanVolume(o *v1alpha1.NatsStreamingCluster) []corev1.Volume {
	volume := o.Spec.Volume
	return volume
}
func stanContainer(o *v1alpha1.NatsStreamingCluster) corev1.Container {
	container := corev1.Container{
		Name:         "stan",
		VolumeMounts: o.Spec.VolumeMounts,
	}
	if o.Spec.Image != "" {
		container.Image = o.Spec.Image
	} else {
		container.Image = DefaultNATSStreamingImage
	}

	return container
}

func stanContainerCmd(o *v1alpha1.NatsStreamingCluster) []string {
	return []string{
		"/nats-streaming-server",
		"-store", "file", "-dir", "store",
		"-clustered", "-cluster_id", o.Name,
		"-nats_server", fmt.Sprintf("nats://%s:4222", o.Spec.NatsService),
	}
}

func stanContainerBootstrapCmd(o *v1alpha1.NatsStreamingCluster) []string {
	cmd := stanContainerCmd(o)
	cmd = append(cmd, "-cluster_bootstrap")
	return cmd
}

func createMissingPods(o *v1alpha1.NatsStreamingCluster, n int) error {
	pods := make([]*corev1.Pod, 0)
	for i := int(o.Spec.Size); len(pods) < n && i > 0; i-- {
		// Check whether the node has been created already,
		// otherwise skip it.
		pod := newStanPod(o)
		pod.Name = fmt.Sprintf("%s-%d", o.Name, i)
		err := sdk.Get(pod.DeepCopy())
		if err == nil {
			continue
		}
		// Fill in the containers information for the bootstrap node.
		container := stanContainer(o)
		container.Command = stanContainerCmd(o)
		pod.Spec.Containers = []corev1.Container{container}
		pods = append(pods, pod)
	}

	for _, pod := range pods {
		log.Debugf("Creating pod %q", pod.Name)
		err := sdk.Create(pod)
		if err != nil && !errors.IsAlreadyExists(err) {
			log.Errorf("Failed to create replica Pod: %v", err)
			continue
		}
	}
	return nil
}

func shrinkCluster(pods []*corev1.Pod, delta int) error {
	var err error
	var deleted int
	for i := len(pods) - 1; i > 0; i-- {
		if deleted == delta {
			return nil
		}

		pod := pods[i]
		err = sdk.Delete(pod.DeepCopy())
		if err != nil {
			continue
		}
		deleted++
	}

	return err
}

func newStanPod(o *v1alpha1.NatsStreamingCluster) *corev1.Pod {
	labels := map[string]string{
		"app":          "nats-streaming",
		"stan_cluster": o.Name,
	}
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(o, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    "NatsStreamingCluster",
				}),
			},
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
}

func findPods(name string, namespace string) (*corev1.PodList, error) {
	pods := &corev1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}
	err := sdk.List(namespace, pods, sdk.WithListOptions(
		&metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app":          "nats-streaming",
				"stan_cluster": name,
			}).String(),
		},
	))

	return pods, err
}

func findRunningPods(name string, namespace string) ([]*corev1.Pod, error) {
	pods, err := findPods(name, namespace)
	if err != nil {
		return nil, err
	}
	runningPods := make([]*corev1.Pod, 0)
	for _, pod := range pods.Items {
		if pod.DeletionTimestamp != nil {
			continue
		}
		runningPods = append(runningPods, pod.DeepCopy())
	}
	return runningPods, nil
}
