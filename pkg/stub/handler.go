package stub

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
		pods, err := findPods(o.Name, o.Namespace)
		if err != nil {
			return err
		}

		// Check cluster health.
		n := int(o.Spec.Size) - len(pods.Items)
		if n == 0 {
			log.Infof("Enough pods for cluster(name=%q, delta=%d, uid=%v)", o.Name, n, o.UID)
			return nil
		} else if n < 0 {
			log.Infof("Too many pods for cluster(name=%q, delta=%d, uid=%v)", o.Name, n, o.UID)
			return nil
		} else if n > 0 {
			log.Infof("Not enough pods for cluster(name=%q, delta=%d, uid=%v)", o.Name, n, o.UID)

			// If this is the first time we have perceived the pod,
			// then try to create the first node to bootstrap the
			// cluster.
			if _, ok := h.clusters[o.UID]; !ok {
				err := createBootstrapPod(o)
				if err != nil {
					log.Errorf("Could not create bootstrapping node: %s", err)
				}
				return nil
			}

			return createMissingPods(o, n)
		}
	}
	return nil
}

func createBootstrapPod(o *v1alpha1.NatsStreamingCluster) error {
	pod := newStanPod(o)
	pod.Name = fmt.Sprintf("%s-1", o.Name)

	// Fill in the containers information for the bootstrap node.
	container := corev1.Container{
		Name:  "stan",
		Image: o.Spec.Image,
		Command: []string{"/nats-streaming-server",
			"-store", "file", "-dir", "store",
			"-clustered", "-cluster_id", o.Name,
			"-cluster_bootstrap", // Bootstrap flag!
			"-nats_server", fmt.Sprintf("nats://%s:4222", o.Spec.NatsService),
		},
	}
	pod.Spec.Containers = []corev1.Container{container}

	log.Debugf("Creating pod: %v", pod)
	err := sdk.Create(pod)
	if err != nil && !errors.IsAlreadyExists(err) {
		log.Errorf("Failed to create bootstrap Pod: %v", err)
		return err
	}

	return nil
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
		container := corev1.Container{
			Name:  "stan",
			Image: o.Spec.Image,
			Command: []string{"/nats-streaming-server",
				"-store", "file", "-dir", "store",
				"-clustered", "-cluster_id", o.Name,
				"-nats_server", fmt.Sprintf("nats://%s:4222", o.Spec.NatsService),
			},
		}
		pod.Spec.Containers = []corev1.Container{container}
		pods = append(pods, pod)
	}

	for _, pod := range pods {
		log.Infof("Creating pod: %v", pod)
		err := sdk.Create(pod)
		if err != nil && !errors.IsAlreadyExists(err) {
			log.Errorf("Failed to create replica Pod: %v", err)
			continue
		}
	}

	return nil
}

func newStanPod(cr *v1alpha1.NatsStreamingCluster) *corev1.Pod {
	labels := map[string]string{
		"app":          "nats-streaming",
		"stan_cluster": cr.Name,
	}
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cr.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    "NatsStreamingCluster",
				}),
			},
			Labels: labels,
		},
		Spec: corev1.PodSpec{},
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
