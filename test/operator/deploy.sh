#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Wait once again for cluster to be ready
JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'
until kubectl get nodes -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1; done

# Bootstrap own user policy
kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --serviceaccount=kube-system:default

# Confirm delete the CRD in case present right now
kubectl get crd | grep natscluster && {
    kubectl delete crd natsclusters.nats.io
}

# Deploy the NATS cluster manifest with RBAC enabled
kubectl apply -f https://github.com/nats-io/nats-operator/releases/download/v0.4.2/00-prereqs.yaml
kubectl apply -f https://github.com/nats-io/nats-operator/releases/download/v0.4.2/10-deployment.yaml

# Wait until the CRD is ready
attempts=0
until kubectl get crd natsclusters.nats.io -o yaml | grep InitialNamesAccepted; do
    if [[ attempts -eq 60 ]]; then
        echo "Gave up waiting for CRD to be ready..."
        kubectl -n nats-io logs deployment/nats-operator
        exit 1
    fi
    ((++attempts))

    echo "Waiting for CRD... ($attempts attempts)"
    sleep 1
done

# Deploy the NATS Streaming CRD with RBAC enabled
kubectl apply -f deploy/default-rbac.yaml

kubectl apply -f deploy/deployment.yaml

# Wait until the CRD is ready
attempts=0
until kubectl get crd natsstreamingclusters.streaming.nats.io -o yaml | grep InitialNamesAccepted; do
    if [[ attempts -eq 60 ]]; then
        echo "Gave up waiting for CRD to be ready..."
        kubectl -n nats-io logs deployment/nats-streaming-operator
        exit 1
    fi
    ((++attempts))

    echo "Waiting for CRD... ($attempts attempts)"
    sleep 1
done

# Deploy an example manifest and wait for pods to appear
kubectl apply -f deploy/examples/example-nats-cluster.yaml

# Wait until 3 pods appear
attempts=0
until kubectl get pods | grep -v operator | grep nats | grep Running | wc -l | grep 3; do
    if [[ attempts -eq 60 ]]; then
        echo "Gave up waiting for NatsCluster to be ready..."
        kubectl logs deployment/nats-operator
        kubectl logs -l nats_cluster=example-nats
        exit 1
    fi

    echo "Waiting for pods to appear ($attempts attempts)..."
    ((++attempts))
    sleep 1
done

# Show output to confirm.
kubectl logs -l nats_cluster=example-nats

# Next, deploy the NATS Streaming Cluster
kubectl apply -f deploy/examples/example-stan-cluster.yaml

# Wait until 3 pods appear
attempts=0
until kubectl get pods | grep -v operator | grep stan | wc -l | grep 3; do
    if [[ attempts -eq 120 ]]; then
        echo "Gave up waiting for NatsStreamingCluster to be ready..."
        kubectl get pods
        kubectl logs deployment/nats-streaming-operator
        kubectl logs -l stan_cluster=example-stan
        exit 1
    fi

    echo "Waiting for pods to appear ($attempts attempts)..."
    ((++attempts))
    sleep 1
done

kubectl delete deployment/nats-streaming-operator
kubectl delete stanclusters example-stan
