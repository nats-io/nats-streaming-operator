#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

echo "--- Updating NatsStreamingCluster CRD Deepcopy..."
deepcopy-gen \
             --input-dirs="github.com/nats-io/nats-streaming-operator/pkg/apis/streaming/v1alpha1" \
	     --output-file-base zz_generated.deepcopy  \
	     --bounding-dirs "github.com/nats-io/nats-operator/pkg/apis" \
             --go-header-file hack/boilerplate.txt

TYPED_CLIENT_VERSION=v1alpha1

echo "--- Updating NatsStreamingCluster Typed Client..."
client-gen \
     --input="github.com/nats-io/nats-streaming-operator/pkg/apis/streaming/v1alpha1" \
     --output-package="github.com/nats-io/nats-streaming-operator/pkg/client/" \
     --clientset-name $TYPED_CLIENT_VERSION \
     --input-base ""  \
     --go-header-file hack/boilerplate.txt
