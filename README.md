# NATS Streaming Operator

[![License Apache 2.0](https://img.shields.io/badge/License-Apache2-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![Version](https://d25lcipzij17d.cloudfront.net/badge.svg?id=go&type=5&v=0.1.0)](https://github.com/nats-io/nats-operator/releases/tag/v0.1.0)

Operator for managing NATS Streaming clusters running on [Kubernetes](http://kubernetes.io).

## Requirements

- Kubernetes v1.8+
- [NATS Operator](https://github.com/nats-io/nats-operator) v0.2.0+

## Getting Started

The NATS Streaming Operator makes available a `NatsStreamingCluster` [Custom Resources Definition](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/) 
that can be used to quickly assemble a NATS Streaming cluster on top of a Kubernetes clusters.

In order to create a

The current version of the operator creates a `NatsCluster` 
(CRD) under the `nats.io` API group, to which you can make requests to
create NATS clusters.

To add the `NatsStreamingCluster` CRD and running NATS Streaming Operator to your cluster you can run:

```sh
# Installing the NatsStreamingCluster CRD
$ kubectl apply -f https://raw.githubusercontent.com/nats-io/nats-streaming-operator/master/deploy/crd.yaml

# Installing the NATS Streaming Operator on the default namespace
$ kubectl apply -f https://raw.githubusercontent.com/nats-io/nats-streaming-operator/master/deploy/operator.yaml
```

You will then be able to confirm that there is a new CRD registered
in the cluster:

```
$ kubectl get crd

NAME                                      CREATED AT
natsclusters.nats.io                      2018-07-20T07:59:42Z
natsserviceroles.nats.io                  2018-07-20T07:59:46Z
natsstreamingclusters.streaming.nats.io   2018-07-23T00:12:13Z
```

An example of creating a 3 node NATS Streaming cluster that connects
to a 3 node NATS cluster is below.  The NATS Streaming operator will
be responsible of assembling the cluster and replacing pods in case of
failures.

```
echo '
---
apiVersion: "nats.io/v1alpha2"
kind: "NatsCluster"
metadata:
  name: "example-nats"
spec:
  size: 3
---
apiVersion: "streaming.nats.io/v1alpha1"
kind: "NatsStreamingCluster"
metadata:
  name: "example-stan"
spec:
  # Number of nodes in the cluster
  size: 3

  # NATS Service available in the same namespace
  natsSvc: "example-nats"
' | kubectl apply -f -
```

To list all the NATS Streaming clusters:

```sh
$ kubectl get natsstreamingcluster.streaming.nats.io

NAME                   AGE
example-stan-cluster   1s
```

Or also via the `stanclusters` version:

```sh
$ kubectl get stanclusters

NAME                   AGE
example-stan-cluster   1s
```

## Development

### Building the Docker Image

To build the `nats-streaming-operator` Docker image:

```sh
$ docker build -f docker/operator/Dockerfile -t <image>:<tag> .
```

You'll need Docker `17.06.0-ce` or higher.

### Running outside the cluster for debugging

The NATS Streaming Operator was built using the [Operator Framework](https://github.com/operator-framework/operator-sdk) so
it is required to be installed for development purposes ([quick start](https://github.com/operator-framework/operator-sdk#quick-start)).
Using the Operator SDK, the operator can be started as follows:

```
WATCH_NAMESPACE=default OPERATOR_NAME=nats-streaming-operator operator-sdk up local
```

