package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats-streaming-operator/internal/operator"
	stanv1alpha1 "github.com/nats-io/nats-streaming-operator/pkg/apis/streaming/v1alpha1"
	stancrdclient "github.com/nats-io/nats-streaming-operator/pkg/client/v1alpha1"
	k8scorev1 "k8s.io/api/core/v1"
	k8scrdclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	k8sclient "k8s.io/client-go/kubernetes/typed/core/v1"
	k8srestapi "k8s.io/client-go/rest"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
)

func TestCreateCluster(t *testing.T) {
	kc, err := newKubeClients()
	if err != nil {
		t.Fatal(err)
	}
	controller := operator.NewController(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	go controller.Run(ctx)

	cluster := &stanv1alpha1.NatsStreamingCluster{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       "NatsStreamingCluster",
			APIVersion: stanv1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      "stan-cluster-basic-test",
			Namespace: "default",
		},
		Spec: stanv1alpha1.NatsStreamingClusterSpec{
			Size:        3,
			NatsService: "example-nats",
		},
	}
	_, err = kc.stan.StreamingV1alpha1().NatsStreamingClusters("default").Create(cluster)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := kc.stan.StreamingV1alpha1().NatsStreamingClusters("default").Delete("stan-cluster-basic-test", &k8smetav1.DeleteOptions{})
		if err != nil {
			t.Error(err)
		}
	}()

	opts := k8smetav1.ListOptions{
		LabelSelector: k8slabels.SelectorFromSet(map[string]string{
			"app":          "nats-streaming",
			"stan_cluster": "stan-cluster-basic-test",
		}).String(),
	}

	err = waitFor(ctx, func() error {
		result, err := kc.core.Pods("default").List(opts)
		if err != nil {
			return err
		}
		got := len(result.Items)
		if got < 3 {
			return fmt.Errorf("Not enough pods, got: %v", got)
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCreateClusterWithDebugFlags(t *testing.T) {
	kc, err := newKubeClients()
	if err != nil {
		t.Fatal(err)
	}
	controller := operator.NewController(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	go controller.Run(ctx)

	name := "stan-cluster-debug-test"
	cluster := &stanv1alpha1.NatsStreamingCluster{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       "NatsStreamingCluster",
			APIVersion: stanv1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: stanv1alpha1.NatsStreamingClusterSpec{
			Size:        3,
			NatsService: "example-nats",
			Config: &stanv1alpha1.ServerConfig{
				Debug:       true,
				Trace:       true,
				RaftLogging: true,
			},
		},
	}
	_, err = kc.stan.StreamingV1alpha1().NatsStreamingClusters("default").Create(cluster)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := kc.stan.StreamingV1alpha1().NatsStreamingClusters("default").Delete(name, &k8smetav1.DeleteOptions{})
		if err != nil {
			t.Error(err)
		}
	}()

	opts := k8smetav1.ListOptions{
		LabelSelector: k8slabels.SelectorFromSet(map[string]string{
			"app":          "nats-streaming",
			"stan_cluster": name,
		}).String(),
	}

	err = waitFor(ctx, func() error {
		result, err := kc.core.Pods("default").List(opts)
		if err != nil {
			return err
		}
		for _, item := range result.Items {
			s := strings.Join(item.Spec.Containers[0].Command, " ")

			expectedFlag := "-SD"
			if !strings.Contains(s, expectedFlag) {
				return fmt.Errorf("Does not contain %s flag", expectedFlag)
			}

			expectedFlag = "-SV"
			if !strings.Contains(s, expectedFlag) {
				return fmt.Errorf("Does not contain %s flag", expectedFlag)
			}

			expectedFlag = "--cluster_raft_logging"
			if !strings.Contains(s, expectedFlag) {
				return fmt.Errorf("Does not contain %s flag", expectedFlag)
			}

			expectedFlag = "-m"
			if !strings.Contains(s, expectedFlag) {
				return fmt.Errorf("Does not contain %s flag", expectedFlag)
			}
		}

		got := len(result.Items)
		if got < 1 {
			return fmt.Errorf("Not enough pods, got: %v", got)
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCreateClusterWithCustomTemplate(t *testing.T) {
	kc, err := newKubeClients()
	if err != nil {
		t.Fatal(err)
	}
	controller := operator.NewController(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	go controller.Run(ctx)

	name := "stan-cluster-custom-test"
	cluster := &stanv1alpha1.NatsStreamingCluster{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       "NatsStreamingCluster",
			APIVersion: stanv1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: stanv1alpha1.NatsStreamingClusterSpec{
			Size:        3,
			NatsService: "example-nats",
			StoreType:   "SQL",
			ConfigFile:  "/etc/streaming/config/stan.conf",
			Config: &stanv1alpha1.ServerConfig{
				Debug:       true,
				Trace:       true,
				RaftLogging: true,
			},
			PodTemplate: &k8scorev1.PodTemplateSpec{
				Spec: k8scorev1.PodSpec{
					RestartPolicy: k8scorev1.RestartPolicyNever,
				},
			},
		},
	}
	_, err = kc.stan.StreamingV1alpha1().NatsStreamingClusters("default").Create(cluster)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := kc.stan.StreamingV1alpha1().NatsStreamingClusters("default").Delete(name, &k8smetav1.DeleteOptions{})
		if err != nil {
			t.Error(err)
		}
	}()

	opts := k8smetav1.ListOptions{
		LabelSelector: k8slabels.SelectorFromSet(map[string]string{
			"app":          "nats-streaming",
			"stan_cluster": name,
		}).String(),
	}

	err = waitFor(ctx, func() error {
		result, err := kc.core.Pods("default").List(opts)
		if err != nil {
			return err
		}
		for _, item := range result.Items {
			s := strings.Join(item.Spec.Containers[0].Command, " ")

			expectedFlag := "-sc"
			if !strings.Contains(s, expectedFlag) {
				return fmt.Errorf("Does not contain %s flag", expectedFlag)
			}

			expectedFlag = "-store"
			if !strings.Contains(s, expectedFlag) {
				return fmt.Errorf("Does not contain %s flag", expectedFlag)
			}

			if item.Spec.RestartPolicy != k8scorev1.RestartPolicyNever {
				return fmt.Errorf("Custom restart policy was not set, got %s", item.Spec.RestartPolicy)
			}
		}

		got := len(result.Items)
		if got < 1 {
			return fmt.Errorf("Not enough pods, got: %v", got)
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestCreateWithCustomStoreDirTemplate(t *testing.T) {
	kc, err := newKubeClients()
	if err != nil {
		t.Fatal(err)
	}
	controller := operator.NewController(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	go controller.Run(ctx)

	name := "stan-cluster-custom-store-dir-test"
	cluster := &stanv1alpha1.NatsStreamingCluster{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       "NatsStreamingCluster",
			APIVersion: stanv1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: stanv1alpha1.NatsStreamingClusterSpec{
			Size:        1,
			NatsService: "example-nats",
			Config: &stanv1alpha1.ServerConfig{
				StoreDir: "/my-store-dir",
			},
			PodTemplate: &k8scorev1.PodTemplateSpec{
				Spec: k8scorev1.PodSpec{
					RestartPolicy: k8scorev1.RestartPolicyNever,
				},
			},
		},
	}
	_, err = kc.stan.StreamingV1alpha1().NatsStreamingClusters("default").Create(cluster)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := kc.stan.StreamingV1alpha1().NatsStreamingClusters("default").Delete(name, &k8smetav1.DeleteOptions{})
		if err != nil {
			t.Error(err)
		}
	}()

	opts := k8smetav1.ListOptions{
		LabelSelector: k8slabels.SelectorFromSet(map[string]string{
			"app":          "nats-streaming",
			"stan_cluster": name,
		}).String(),
	}

	err = waitFor(ctx, func() error {
		result, err := kc.core.Pods("default").List(opts)
		if err != nil {
			return err
		}
		for _, item := range result.Items {
			got := strings.Join(item.Spec.Containers[0].Command, " ")
			expected := `/nats-streaming-server -cluster_id stan-cluster-custom-store-dir-test -nats_server nats://example-nats:4222 -m 8222 -store file --cluster_node_id="stan-cluster-custom-store-dir-test-1" -dir /my-store-dir/stan-cluster-custom-store-dir-test-1`
			if got != expected {
				return fmt.Errorf("Expected %s, got: %s", expected, got)
			}
		}

		got := len(result.Items)
		if got < 1 {
			return fmt.Errorf("Not enough pods, got: %v", got)
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func waitFor(ctx context.Context, cb func() error) error {
	for {
		var err error
		select {
		case <-ctx.Done():
			if ctx.Err() != nil && ctx.Err() != context.Canceled {
				if err == nil {
					err = ctx.Err()
				}
				return err
			}
		default:
		}

		err = cb()
		if err != nil {
			continue
		}

		return nil
	}
}

type clients struct {
	core k8sclient.CoreV1Interface
	crd  k8scrdclient.Interface
	stan stancrdclient.Interface
}

func newKubeClients() (*clients, error) {
	var err error
	var cfg *k8srestapi.Config
	if kubeconfig := os.Getenv("KUBERNETES_CONFIG_FILE"); kubeconfig != "" {
		cfg, err = k8sclientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("KUBERNETES_CONFIG_FILE env variable must be set")
	}
	kc, err := k8sclient.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	kcrdc, err := k8scrdclient.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	ncr, err := stancrdclient.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &clients{
		core: kc,
		crd:  kcrdc,
		stan: ncr,
	}, nil
}
