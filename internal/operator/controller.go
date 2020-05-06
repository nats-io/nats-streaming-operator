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
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	stanv1alpha1 "github.com/nats-io/nats-streaming-operator/pkg/apis/streaming/v1alpha1"
	stancrdclient "github.com/nats-io/nats-streaming-operator/pkg/client/v1alpha1"
	log "github.com/sirupsen/logrus"
	k8scorev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfields "k8s.io/apimachinery/pkg/fields"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	k8sclient "k8s.io/client-go/kubernetes"
	k8srestapi "k8s.io/client-go/rest"
	k8scache "k8s.io/client-go/tools/cache"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
)

// Options for the operator.
type Options struct {
	// Namespace where the NATS Streaming Operator will
	// be managing the clusters.
	Namespace string

	// NoSignals marks whether to enable the signal handler.
	NoSignals bool
}

// Controller manages NATS Clusters running in Kubernetes.
type Controller struct {
	mu sync.Mutex

	// Client to interact with Kubernetes resources.
	kc k8sclient.Interface

	// Client to interact with NATS Streaming Operator Kubernetes resources.
	ncr stancrdclient.Interface

	// clusters that the Operator is controlling.
	clusters map[k8stypes.UID]*stanv1alpha1.NatsStreamingCluster

	// opts is the set of options.
	opts *Options

	// quit stops the controller.
	quit func()
}

func NewController(opts *Options) *Controller {
	if opts == nil {
		opts = &Options{}
	}
	return &Controller{
		opts:     opts,
		clusters: make(map[k8stypes.UID]*stanv1alpha1.NatsStreamingCluster),
	}
}

// SetupClients takes the configuration and prepares the rest
// clients that will be used to interact with the cluster objects.
func (c *Controller) SetupClients(cfg *k8srestapi.Config) error {
	kc, err := k8sclient.NewForConfig(cfg)
	if err != nil {
		return err
	}
	c.kc = kc

	ncr, err := stancrdclient.NewForConfig(cfg)
	if err != nil {
		return err
	}
	c.ncr = ncr
	return nil
}

// SetupSignalHandler enables handling process signals.
func (c *Controller) SetupSignalHandler(ctx context.Context) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for sig := range sigCh {
		log.Debugf("Trapped '%v' signal", sig)

		// If main context already done, then just skip
		select {
		case <-ctx.Done():
			continue
		default:
		}

		switch sig {
		case syscall.SIGINT:
			log.Infof("Exiting...")
			os.Exit(0)
			return
		case syscall.SIGTERM:
			// Gracefully shutdown the operator.  This blocks
			// until all controllers have stopped running.
			c.Shutdown()
			return
		}
	}
}

// NewInformer takes a controller and a set of resource handlers and
// returns an indexer and a controller that are subscribed to changes
// to the state of a NATS cluster resource.
func NewInformer(
	c *Controller,
	resourceFuncs k8scache.ResourceEventHandlerFuncs,
	interval time.Duration,
) (k8scache.Indexer, k8scache.Controller) {
	listWatcher := k8scache.NewListWatchFromClient(
		c.ncr.StreamingV1alpha1().RESTClient(),

		// Plural name of the CRD
		"natsstreamingclusters",

		// Namespace where the clusters will be created.
		c.opts.Namespace,
		k8sfields.Everything(),
	)
	return k8scache.NewIndexerInformer(
		listWatcher,
		&stanv1alpha1.NatsStreamingCluster{},

		// How often it will poll for the state
		// of the resources.
		interval,

		// Handlers
		resourceFuncs,
		k8scache.Indexers{},
	)
}

// Run starts the NATS Streaming operator controller loop.
func (c *Controller) Run(ctx context.Context) error {
	if !c.opts.NoSignals {
		go c.SetupSignalHandler(ctx)
	}

	// Setup configuration for when operator runs inside/outside
	// the cluster and the API client for making requests.
	var err error
	var cfg *k8srestapi.Config
	if kubeconfig := os.Getenv("KUBERNETES_CONFIG_FILE"); kubeconfig != "" {
		cfg, err = k8sclientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		cfg, err = k8srestapi.InClusterConfig()
	}
	if err != nil {
		return err
	}
	if err := c.SetupClients(cfg); err != nil {
		return err
	}

	// Resolve namespace from environment if not set explicitly.
	if c.opts.Namespace == "" {
		ns := os.Getenv("MY_POD_NAMESPACE")
		if len(ns) != 0 {
			c.opts.Namespace = ns
		}
	}

	// Set up cancellation context for the main loop.
	ctx, cancelFn := context.WithCancel(ctx)

	// Subscribe to changes on NatsStreamingCluster resources.
	_, informer := NewInformer(c, k8scache.ResourceEventHandlerFuncs{
		AddFunc: func(o interface{}) {
			err := c.processAdd(ctx, o)
			if err != nil {
				log.Errorf("Error on add: %v", err)
			}
		},
		UpdateFunc: func(o interface{}, n interface{}) {
			err := c.processUpdate(ctx, o, n)
			if err != nil {
				log.Errorf("Error on update: %v", err)
			}
		},
		DeleteFunc: func(o interface{}) {
			err := c.processDelete(ctx, o)
			if err != nil {
				log.Errorf("Error on delete: %v", err)
			}
		},
	}, ResyncPeriod)

	c.quit = func() {
		// Signal cancellation of the main context.
		cancelFn()
	}

	// Stops running until the context is canceled,
	// which should only happen when Shutdown is called.
	informer.Run(ctx.Done())

	return ctx.Err()
}

// Shutdown stops the operator controller.
func (c *Controller) Shutdown() {
	c.quit()
	log.Infof("Bye")
	return
}

func (c *Controller) processAdd(ctx context.Context, v interface{}) error {
	o := v.(*stanv1alpha1.NatsStreamingCluster)
	log.Infof("Adding cluster '%s/%s' (uid=%s)", o.Namespace, o.Name, o.UID)

	if o.DeletionTimestamp != nil {
		// Throwaway cluster and let garbage collection remove
		// the pods via ownership cascade delete.
		log.Debugf("Removing %v cluster", o.Name)
		c.mu.Lock()
		delete(c.clusters, o.UID)
		c.mu.Unlock()
		return nil
	}

	// Collect metadata from latest perceived version.
	defer func() {
		c.mu.Lock()
		c.clusters[o.UID] = o
		c.mu.Unlock()
	}()

	return c.reconcile(o)
}

func (c *Controller) processUpdate(ctx context.Context, o interface{}, n interface{}) error {
	newc := n.(*stanv1alpha1.NatsStreamingCluster)
	log.Debugf("Syncing cluster '%s/%s' (uid=%s)", newc.Namespace, newc.Name, newc.UID)

	if newc.DeletionTimestamp != nil {
		_, ok := c.clusters[newc.UID]
		if !ok {
			return nil
		}
		c.mu.Lock()
		delete(c.clusters, newc.UID)
		c.mu.Unlock()
		log.Debugf("Deleting '%s/%s' cluster (uid=%s)", newc.Namespace, newc.Name, newc.UID)
		return nil
	}

	defer func() {
		c.mu.Lock()
		c.clusters[newc.UID] = newc
		c.mu.Unlock()
	}()

	return c.reconcile(newc)
}

func (c *Controller) processDelete(ctx context.Context, v interface{}) error {
	o := v.(*stanv1alpha1.NatsStreamingCluster)
	log.Infof("Deleted '%s/%s' cluster (uid=%s)", o.Namespace, o.Name, o.UID)

	return nil
}

func (c *Controller) reconcile(o *stanv1alpha1.NatsStreamingCluster) error {
	pods, err := c.findRunningPods(o.Name, o.Namespace)
	if err != nil {
		return err
	}
	if o.Spec.StoreType == "SQL" || o.Spec.Size < 1 {
		o.Spec.Size = 1
	}

	n := len(pods) - int(o.Spec.Size)
	if n == 0 {
		log.Debugf("Reconciled '%s/%s' cluster (size=%d/%d)", o.Namespace, o.Name, o.Spec.Size, o.Spec.Size)
		return nil
	} else if n > 0 {
		log.Infof("Too many pods for '%s/%s' cluster (size=%d/%d), removing %d pods...", o.Namespace, o.Name, len(pods), o.Spec.Size, n)
		return c.shrinkCluster(pods, n)
	} else if n < 0 {
		log.Infof("Missing pods for '%s/%s' cluster (size=%d/%d), creating %d pods...", o.Namespace, o.Name, len(pods), o.Spec.Size, n*-1)

		if o.Spec.StoreType == "SQL" || (o.Spec.Config != nil && o.Spec.Config.FTGroup != "") {
			return c.createMissingPods(o, n*-1)
		}

		// If this is the first time we have perceived the pod,
		// then try to create the first node to bootstrap the
		// cluster.
		c.mu.Lock()
		if _, ok := c.clusters[o.UID]; !ok {
			c.mu.Unlock()
			return c.createBootstrapPod(o)
		}
		c.mu.Unlock()

		// If no other nodes are available, then create one with the bootstrap
		// flag so that it can become the leader.
		if n == 0 {
			return c.createBootstrapPod(o)
		}
		return c.createMissingPods(o, n*-1)
	}

	return nil
}

func (c *Controller) createBootstrapPod(o *stanv1alpha1.NatsStreamingCluster) error {
	pod := newStanPod(o)
	pod.Name = fmt.Sprintf("%s-1", o.Name)

	container := stanContainer(o, pod)
	container.Command = stanContainerBootstrapCmd(o, pod)

	if len(pod.Spec.Containers) >= 1 {
		pod.Spec.Containers[0] = container
	} else {
		pod.Spec.Containers = []k8scorev1.Container{container}
	}

	log.Infof("Creating bootstrap pod '%s/%s'", o.Namespace, pod.Name)
	_, err := c.kc.CoreV1().Pods(o.Namespace).Create(pod)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		log.Errorf("Failed to create bootstrap Pod: %v", err)
		return err
	}

	return nil
}

func stanContainer(o *stanv1alpha1.NatsStreamingCluster, pod *k8scorev1.Pod) k8scorev1.Container {
	// Get the first container in case present and use it
	// as the container for NATS Streaming.
	var container k8scorev1.Container
	if len(pod.Spec.Containers) >= 1 {
		container = pod.Spec.Containers[0]
	} else {
		container = k8scorev1.Container{}
	}

	if o.Spec.Image != "" {
		container.Image = o.Spec.Image
	} else if container.Image == "" {
		container.Image = DefaultNATSStreamingImage
	}
	container.Name = "stan"

	return container
}

func stanContainerCmd(o *stanv1alpha1.NatsStreamingCluster, pod *k8scorev1.Pod) []string {
	args := []string{
		"/nats-streaming-server",
		"-cluster_id", o.Name,
		"-nats_server", fmt.Sprintf("nats://%s:4222", o.Spec.NatsService),
		"-m", fmt.Sprintf("%d", MonitoringPort),
	}

	var storeArgs []string
	if o.Spec.StoreType == "SQL" {
		storeArgs = []string{
			"-store", "SQL",
		}
	} else if o.Spec.StoreType == "MEMORY" {
		storeArgs = []string{
			"-store", "MEMORY",
		}
	} else {
		storeArgs = []string{
			"-store", "file",
		}

		ftModeEnabled := o.Spec.Config != nil && o.Spec.Config.FTGroup != ""
		isClustered := o.Spec.Config != nil && (o.Spec.Size > 1 || o.Spec.Config.Clustered)

		// Disable clustering if using single instance or FT mode.
		if isClustered && !ftModeEnabled {
			storeArgs = append(storeArgs, "-clustered")
			storeArgs = append(storeArgs, fmt.Sprintf("--cluster_node_id=%q", pod.Name))
		}

		// Allow using a custom mount path which could be a persistent volume.
		if o.Spec.Config != nil && o.Spec.Config.StoreDir != "" {
			if ftModeEnabled {
				// In case of FT mode then use the name of the first pod
				// as the storage directory in order to make it possible
				// to switch from clustered mode to fault tolerance mode.
				name := fmt.Sprintf("%s-1", o.Name)
				storeArgs = append(storeArgs, "-dir", o.Spec.Config.StoreDir+"/"+name)
				storeArgs = append(storeArgs, fmt.Sprintf("--ft_group=%s", o.Spec.Config.FTGroup))
			} else {
				// Using clustering.
				storeArgs = append(storeArgs, "-dir", o.Spec.Config.StoreDir+"/"+pod.Name)
				storeArgs = append(storeArgs, "--cluster_log_path", o.Spec.Config.StoreDir+"/raft/"+pod.Name)
			}
		} else {
			// Use local filesystem if no explicit directory was set.
			storeArgs = append(storeArgs, "-dir", "store")
		}
	}
	args = append(args, storeArgs...)

	// Debugging params
	if o.Spec.Config != nil {
		if o.Spec.Config.Debug {
			args = append(args, "-SD")
		}
		if o.Spec.Config.Trace {
			args = append(args, "-SV")
		}
		if o.Spec.Config.RaftLogging {
			args = append(args, "--cluster_raft_logging")
		}
	}

	if o.Spec.ConfigFile != "" {
		args = append(args, "-sc", o.Spec.ConfigFile)
	}

	return args
}

func stanContainerBootstrapCmd(o *stanv1alpha1.NatsStreamingCluster, pod *k8scorev1.Pod) []string {
	cmd := stanContainerCmd(o, pod)

	if o.Spec.Size == 1 {
		return cmd
	}

	cmd = append(cmd, "-cluster_bootstrap")
	return cmd
}

func (c *Controller) createMissingPods(o *stanv1alpha1.NatsStreamingCluster, n int) error {
	pods := make([]*k8scorev1.Pod, 0)
	for i := int(o.Spec.Size); len(pods) < n && i > 0; i-- {
		// Check whether the node has been created already,
		// otherwise skip it.
		name := fmt.Sprintf("%s-%d", o.Name, i)
		_, err := c.kc.CoreV1().Pods(o.Namespace).Get(name, k8smetav1.GetOptions{})
		if err == nil {
			continue
		}
		pod := newStanPod(o)
		pod.Name = name

		container := stanContainer(o, pod)
		container.Command = stanContainerCmd(o, pod)

		if len(pod.Spec.Containers) >= 1 {
			pod.Spec.Containers[0] = container
		} else {
			pod.Spec.Containers = []k8scorev1.Container{container}
		}
		pods = append(pods, pod)
	}

	for _, pod := range pods {
		log.Infof("Creating pod '%s/%s'", o.Namespace, pod.Name)
		_, err := c.kc.CoreV1().Pods(o.Namespace).Create(pod)
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			log.Errorf("Failed to create replica Pod: %v", err)
			continue
		}
	}
	return nil
}

func (c *Controller) shrinkCluster(pods []*k8scorev1.Pod, delta int) error {
	var err error
	var deleted int
	for i := len(pods) - 1; i > 0; i-- {
		if deleted == delta {
			return nil
		}

		pod := pods[i]
		err := c.kc.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &k8smetav1.DeleteOptions{})
		if err != nil {
			continue
		}
		deleted++
	}

	return err
}

func newStanPod(o *stanv1alpha1.NatsStreamingCluster) *k8scorev1.Pod {
	pod := &k8scorev1.Pod{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}

	// Base from Pod Spec from template if present.
	template := o.Spec.PodTemplate
	if template != nil {
		pod.ObjectMeta = template.ObjectMeta
		pod.Spec = template.Spec
	}

	pod.Namespace = o.Namespace
	pod.OwnerReferences = []k8smetav1.OwnerReference{
		*k8smetav1.NewControllerRef(o, k8sschema.GroupVersionKind{
			Group:   stanv1alpha1.SchemeGroupVersion.Group,
			Version: stanv1alpha1.SchemeGroupVersion.Version,
			Kind:    "NatsStreamingCluster",
		}),
	}

	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels["app"] = "nats-streaming"
	pod.Labels["stan_cluster"] = o.Name

	if pod.Spec.RestartPolicy == "" {
		pod.Spec.RestartPolicy = k8scorev1.RestartPolicyOnFailure
	}
	return pod.DeepCopy()
}

func (c *Controller) findPods(name string, namespace string) (*k8scorev1.PodList, error) {
	opts := k8smetav1.ListOptions{
		LabelSelector: k8slabels.SelectorFromSet(map[string]string{
			"app":          "nats-streaming",
			"stan_cluster": name,
		}).String(),
	}

	return c.kc.CoreV1().Pods(namespace).List(opts)
}

func (c *Controller) findRunningPods(name string, namespace string) ([]*k8scorev1.Pod, error) {
	pods, err := c.findPods(name, namespace)
	if err != nil {
		return nil, err
	}
	runningPods := make([]*k8scorev1.Pod, 0)
	for _, pod := range pods.Items {
		if pod.DeletionTimestamp != nil {
			continue
		}
		runningPods = append(runningPods, pod.DeepCopy())
	}
	return runningPods, nil
}
