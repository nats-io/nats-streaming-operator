# NATS Streaming Operator

[![License Apache 2.0](https://img.shields.io/badge/License-Apache2-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![Build Status](https://travis-ci.org/nats-io/nats-streaming-operator.svg?branch=master)](https://travis-ci.org/nats-io/nats-streaming-operator)
[![Version](https://d25lcipzij17d.cloudfront.net/badge.svg?id=go&type=5&v=0.3.0)](https://github.com/nats-io/nats-streaming-operator/releases/tag/v0.3.0)

Operator for managing NATS Streaming clusters running on [Kubernetes](http://kubernetes.io).

You can find more info about running NATS on Kubernetes in the [docs](https://docs.nats.io/nats-on-kubernetes/nats-kubernetes) as well as a more minimal setup using `StatefulSets` only without using the operator to get started [here](https://docs.nats.io/nats-on-kubernetes/minimal-setup).

## Requirements

- Kubernetes v1.8+
- [NATS Operator](https://github.com/nats-io/nats-operator) v0.2.0+

## Getting Started

The NATS Streaming Operator makes available a `NatsStreamingCluster` [Custom Resources Definition](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/) 
that can be used to assemble a NATS Streaming cluster on top of a Kubernetes cluster.

To install the NATS Streaming Operator on your cluster:

```sh
# Install NATS Operator on default namespace
$ kubectl apply -f https://github.com/nats-io/nats-operator/releases/download/v0.5.0/00-prereqs.yaml
$ kubectl apply -f https://github.com/nats-io/nats-operator/releases/download/v0.5.0/10-deployment.yaml

# Install NATS Streaming Operator on default namespace
$ kubectl apply -f https://raw.githubusercontent.com/nats-io/nats-streaming-operator/master/deploy/default-rbac.yaml

$ kubectl apply -f https://raw.githubusercontent.com/nats-io/nats-streaming-operator/master/deploy/deployment.yaml
```

You will then be able to confirm that there is a new `natsstreamingclusters.streaming.nats.io` CRD registered:

```
$ kubectl get crd

NAME                                      CREATED AT
natsclusters.nats.io                      2018-07-20T07:59:42Z
natsserviceroles.nats.io                  2018-07-20T07:59:46Z
natsstreamingclusters.streaming.nats.io   2018-07-23T00:12:13Z

```

### Deploying a NATS Streaming cluster

An operated NATS Streaming cluster requires being able to connect to a NATS cluster service.
As an example, let's create a NATS cluster named `example-nats` with the NATS Operator on
the `default` namespace:

```sh
echo '
---
apiVersion: "nats.io/v1alpha2"
kind: "NatsCluster"
metadata:
  name: "example-nats"
spec:
  size: 3
' | kubectl apply -f -

$ kubectl get pods
NAME                                       READY     STATUS    RESTARTS   AGE
example-nats-1                             1/1       Running   0          18s
example-nats-2                             1/1       Running   0          8s
example-nats-3                             1/1       Running   0          2s

```

Next, let's deploy a `NatsStreamingCluster` named `example-stan` on the
same namespace that will connect to the `example-nats` service:

```sh
echo '
---
apiVersion: "streaming.nats.io/v1alpha1"
kind: "NatsStreamingCluster"
metadata:
  name: "example-stan"
spec:
  size: 3
  natsSvc: "example-nats"
' | kubectl apply -f -
```

Below you can find an example of a client connecting to the NATS service
and then borrowing that NATS connection to use the NATS Streaming cluster 
named `example-stan` for publishing a few messages, then creating a
subscription to consume the published messages from the beginning.

```go
package main

import (
	"log"

	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
)

func main() {
	nc, err := nats.Connect("nats://example-nats:4222")
	if err != nil {
		log.Fatal(err)
	}
	sc, err := stan.Connect("example-stan", "client-123", stan.NatsConn(nc))
	if err != nil {
		log.Fatal(err)
	}
	sc.Publish("hello", []byte("one"))
	sc.Publish("hello", []byte("two"))
	sc.Publish("hello", []byte("three"))

	sc.Subscribe("hello", func(m *stan.Msg) {
		log.Printf("[Received] %+v", m)
	}, stan.DeliverAllAvailable())

	select {}
}
```

You can list all the NATS Streaming clusters available in a namespace as follows:

```sh
$ kubectl get natsstreamingcluster.streaming.nats.io

NAME                   AGE
example-stan-cluster   1s
```

Or also listing via the `stanclusters` shorter version:

```sh
$ kubectl get stanclusters

NAME                   AGE
example-stan-cluster   1s
```

In this deployment, the first NATS Streaming node would have started as the 
bootstrap node, so it becomes the leader as confirmed by the logs:

```sh
$ kubectl get pods
NAME                                       READY     STATUS    RESTARTS   AGE
example-nats-1                             1/1       Running   0          2m
example-nats-2                             1/1       Running   0          2m
example-nats-3                             1/1       Running   0          1m
example-stan-1                             1/1       Running   0          10s
example-stan-2                             1/1       Running   0          6s
example-stan-3                             1/1       Running   0          6s

$ kubectl logs example-stan-1
[1] 2018/07/26 16:23:08.828388 [INF] STREAM: Starting nats-streaming-server[example-stan] version 0.10.2
[1] 2018/07/26 16:23:08.828430 [INF] STREAM: ServerID: XY20nYFAFI5JctzTDbrpAP
[1] 2018/07/26 16:23:08.828435 [INF] STREAM: Go version: go1.10.3
[1] 2018/07/26 16:23:08.852193 [INF] STREAM: Recovering the state...
[1] 2018/07/26 16:23:08.852348 [INF] STREAM: No recovered state
[1] 2018/07/26 16:23:08.852424 [INF] STREAM: Cluster Node ID : XY20nYFAFI5JctzTDbrpDO
[1] 2018/07/26 16:23:08.852431 [INF] STREAM: Cluster Log Path: example-stan/XY20nYFAFI5JctzTDbrpDO
[1] 2018/07/26 16:23:08.860361 [INF] STREAM: Message store is RAFT_FILE
[1] 2018/07/26 16:23:08.860377 [INF] STREAM: Store location: store
[1] 2018/07/26 16:23:08.860413 [INF] STREAM: ---------- Store Limits ----------
[1] 2018/07/26 16:23:08.860418 [INF] STREAM: Channels:                  100 *
[1] 2018/07/26 16:23:08.860421 [INF] STREAM: --------- Channels Limits --------
[1] 2018/07/26 16:23:08.860427 [INF] STREAM:   Subscriptions:          1000 *
[1] 2018/07/26 16:23:08.860430 [INF] STREAM:   Messages     :       1000000 *
[1] 2018/07/26 16:23:08.860433 [INF] STREAM:   Bytes        :     976.56 MB *
[1] 2018/07/26 16:23:08.860437 [INF] STREAM:   Age          :     unlimited *
[1] 2018/07/26 16:23:08.860440 [INF] STREAM:   Inactivity   :     unlimited *
[1] 2018/07/26 16:23:08.860443 [INF] STREAM: ----------------------------------
[1] 2018/07/26 16:23:10.551835 [INF] STREAM: server became leader, performing leader promotion actions
[1] 2018/07/26 16:23:10.553500 [INF] STREAM: finished leader promotion actions
```

In case of failure, then any of the remaining nodes will then takeover
and missing pods will be replaced.

### Using a DB store

The NATS Streaming Operator can also be used to manage instances
backed by a SQL store.  In this mode, only a single replica will be
created. In order to use DB store support it is needed to include
the DB credentials within the NATS Streaming configuration and mount
it as a volume (using a secret for example):

```sh
echo '
streaming: {
  sql: {
    driver: "postgres"
    source: "postgres://exampleuser:notasecret@example.com/stan?sslmode=disable"
  }
}
' > /tmp/secret.conf

kubectl create secret generic stan-secret --from-file /tmp/secret.conf

echo '
---
apiVersion: "streaming.nats.io/v1alpha1"
kind: "NatsStreamingCluster"
metadata:
  name: "example-stan-db"
spec:
  natsSvc: "example-nats"

  # Explicitly set that the managed NATS Streaming instance
  # will be using an SQL storage, to ensure that only a single
  # instance is available.
  store: SQL
  configFile: "/etc/stan/config/secret.conf"

  # Customize using a Pod Spec template
  template:
    spec:
      volumes:
      - name: stan-secret
        secret:
          secretName: stan-secret
      containers:
        - name: nats-streaming
          volumeMounts:
          - mountPath: /etc/stan/config
            name: stan-secret
            readOnly: true
' | kubectl apply -f -
```

### Using a custom store dir for a Persistent Volume

In order to use a persistent volume, it is needed to customize the location 
of the storage directory so that it is on the mounted filesystem. The volumes
to be mounted by the NATS Streaming pod by definining them in the `template`
with the spec for the Pod.

```yaml
---
apiVersion: "streaming.nats.io/v1alpha1"
kind: "NatsStreamingCluster"
metadata:
  name: "example-stan-pv"
spec:
  natsSvc: "example-nats"

  config:
    storeDir: "/pv/stan"

  # Define mounts in the Pod Spec
  template:
    spec:
      volumes:
      - name: stan-store-dir
        persistentVolumeClaim:
          claimName: streaming-pvc
      containers:
        - name: nats-streaming
          volumeMounts:
          - mountPath: /pv
            name: stan-store-dir
```

### Using a custom store dir and fault tolerance mode

```yaml
---
apiVersion: "streaming.nats.io/v1alpha1"
kind: "NatsStreamingCluster"
metadata:
  name: "example-stan-pv"
spec:
  natsSvc: "example-nats"

  config:
    storeDir: "/pv/stan"
    ftGroup: "stan"

  # Define mounts in the Pod Spec
  template:
    spec:
      volumes:
      - name: stan-store-dir
        persistentVolumeClaim:
          claimName: streaming-pvc
      containers:
        - name: nats-streaming
          volumeMounts:
          - mountPath: /pv
            name: stan-store-dir
```

## Development

### Building the Docker Image

To build the `nats-streaming-operator` Docker image:

```sh
$ docker build . -f docker/operator/Dockerfile -t <image>:<tag> .
```

You'll need Docker `17.06.0-ce` or higher.

### Running outside the cluster for development

For development purposes, the operator can be started locally as follows (against `minikube` for example):

```sh
KUBERNETES_CONFIG_FILE=$HOME/.kube/config DEBUG=true go run cmd/nats-streaming-operator/main.go
```
