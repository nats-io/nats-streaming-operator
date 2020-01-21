# NATS Streaming Operator Helm Chart

NATS Streaming is an extremely performant, lightweight reliable streaming
platform built on NATS.

## TL;DR

```bash
$ helm install .
```

## Introduction

NATS Streaming Operator manages [NATS
Streaming](https://github.com/nats-io/nats-streaming-server) clusters atop
[Kubernetes](http://kubernetes.io), automating their creation and
administration. With the NATS Operator you can benefits from the flexibility
brought by the Kubernetes operator pattern. It means less juggling between
manifests and a few handy features like automatic configuration reload.

If you want to manage NATS entirely by yourself and have more control over your
NATS cluster, you can always use the classic [NATS
Streaming](https://github.com/nats-io/nats-streaming-operator/releases)
deployment.

## Prerequisites

- Kubernetes 1.8+

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
$ helm install --name my-release .
```

The command deploys NATS and the NATS Operator on the Kubernetes cluster in the
default configuration. The [configuration](#configuration) section lists the
parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and
deletes the release.

## Configuration

The following table lists the configurable parameters of the NATS chart and
their default values.

| Parameter                            | Description                                                                                  | Default                                         |
| ------------------------------------ | -------------------------------------------------------------------------------------------- | ----------------------------------------------- |
| `rbacEnabled`                        | Switch to enable/disable RBAC for this chart                                                 | `true`                                          |
| `cluster.enabled`                    | Deploy a NATS Streaming Cluster with the operator                                            | `true`                                          |
| `cluster.name`                       | Name of the NATS Streaming Cluster                                                           | `nats-streaming-cluster`                        |
| `cluster.version`                    | Version of the NATS Streaming Cluster                                                        | `0.12.2`                                        |
| `cluster.size`                       | Number of nodes in the cluster                                                               | `3`                                             |
| `cluster.natsSvc`                    | NATS cluster service name                                                                    | `nats-cluster`                                  |
| `cluster.config.debug`               | Enable debug log                                                                             | `true`                                          |
| `cluster.config.trace`               | Enable tracing                                                                               | `true`                                          |
| `cluster.config.raftLogging`         | Enable Raft Logging                                                                          | `true`                                          |
| `cluster.metrics.enabled`            | Enable prometheus metrics exporter                                                           | `true`                                          |
| `cluster.metrics.image`              | Prometheus metrics exporter image name                                                       | `synadia/prometheus-nats-exporter`              |
| `cluster.metrics.version`            | Prometheus metrics exporter image tag                                                        | `0.2.0`                                         |
| `image.registry`                     | NATS Operator image registry                                                                 | `docker.io`                                     |
| `image.repository`                   | NATS Operator image name                                                                     | `connecteverything/nats-operator`               |
| `image.tag`                          | NATS Operator image tag                                                                      | `0.4.3-v1alpha2`                                |
| `image.pullPolicy`                   | Image pull policy                                                                            | `Always`                                        |
| `image.pullSecrets`                  | Specify image pull secrets                                                                   | `nil`                                           |
| `securityContext.enabled`            | Enable security context                                                                      | `true`                                          |
| `securityContext.fsGroup`            | Group ID for the container                                                                   | `1001`                                          |
| `securityContext.runAsUser`          | User ID for the container                                                                    | `1001`                                          |
| `nodeSelector`                       | Node labels for pod assignment                                                               | `nil`                                           |
| `tolerations`                        | Toleration labels for pod assignment                                                         | `nil`                                           |
| `schedulerName`                      | Name of an alternate scheduler                                                               | `nil`                                           |
| `antiAffinity`                       | Anti-affinity for pod assignment (values: soft or hard)                                      | `soft`                                          |
| `podAnnotations`                     | Annotations to be added to pods                                                              | `{}`                                            |
| `podLabels`                          | Additional labels to be added to pods                                                        | `{}`                                            |
| `updateStrategy`                     | Replicaset Update strategy                                                                   | `OnDelete`                                      |
| `rollingUpdatePartition`             | Partition for Rolling Update strategy                                                        | `nil`                                           |
| `resources`                          | CPU/Memory resource requests/limits                                                          | `{}`                                            |
| `livenessProbe.enabled`              | Enable liveness probe                                                                        | `true`                                          |
| `livenessProbe.initialDelaySeconds`  | Delay before liveness probe is initiated                                                     | `30`                                            |
| `livenessProbe.periodSeconds`        | How often to perform the probe                                                               | `10`                                            |
| `livenessProbe.timeoutSeconds`       | When the probe times out                                                                     | `5`                                             |
| `livenessProbe.failureThreshold`     | Minimum consecutive failures for the probe to be considered failed after having succeeded.   | `6`                                             |
| `livenessProbe.successThreshold`     | Minimum consecutive successes for the probe to be considered successful after having failed. | `1`                                             |
| `readinessProbe.enabled`             | Enable readiness probe                                                                       | `true`                                          |
| `readinessProbe.initialDelaySeconds` | Delay before readiness probe is initiated                                                    | `5`                                             |
| `readinessProbe.periodSeconds`       | How often to perform the probe                                                               | `10`                                            |
| `readinessProbe.timeoutSeconds`      | When the probe times out                                                                     | `5`                                             |
| `readinessProbe.failureThreshold`    | Minimum consecutive failures for the probe to be considered failed after having succeeded.   | `6`                                             |
| `readinessProbe.successThreshold`    | Minimum consecutive successes for the probe to be considered successful after having failed. | `1`                                             |
| `clusterScoped`                      | Enable cluster scoped installation (read carefully the warnings)                             | `false`                                         |

### Example

Here is an example of how to setup a NATS cluster with client authentication.

Specify each parameter using the `--set key=value[,key=value]` argument to `helm
install`.

```bash
$ helm install \
  --name my-release \
  --set cluster.natsSvc=my-nats-cluster \
```

You can consider editing the default values.yaml as it is easier to manage:

```yaml
...
cluster:
  size: 2
  natsSvc: my-nats-cluster
...
```

Alternatively, a YAML file that specifies the values for the parameters can be
provided while installing the chart. For example,

> **Tip**: You can use the default [values.yaml](values.yaml)

```bash
$ helm install --name my-release -f values.yaml .
```
