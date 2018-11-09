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

package main

import (
	"context"
	"os"
	"runtime"

	operator "github.com/nats-io/nats-streaming-operator/internal/operator"
	. "github.com/nats-io/nats-streaming-operator/version"
	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	k8sutil "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	log "github.com/sirupsen/logrus"
)

func main() {

	var err error
	var namespace = ""

	if os.Getenv("DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
	}


	// Get namespace if empty set empty string
        // we will then watch all namespaces
        if os.Getenv("WATCH_NAMESPACE") != "" {
		namespace, err = k8sutil.GetWatchNamespace()
		if err != nil {
			log.Fatalf("Failed to get watch namespace: %v", err)
		}
	} else {
		log.Infof("No namespace provided, watching all namespaces")
	}

	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)

	log.Infof("Starting NATS Streaming Operator v%s", Version)
	log.Infof("Go Version: %s", runtime.Version())
	log.Infof("operator-sdk Version: %v", sdkVersion.Version)

	resource := "streaming.nats.io/v1alpha1"
	kind := "NatsStreamingCluster"

	// TODO: Move to constants
	resyncPeriod := 5
	if namespace == "" {
		log.Infof("Watching %s, %s, all namespace, %d", resource, kind, resyncPeriod)
	} else {
		log.Infof("Watching %s, %s, %s, %d", resource, kind, namespace, resyncPeriod)
	}

	// Look for updates on the NatsStreamingClusters made on this namespace.
	sdk.Watch(resource, kind, namespace, resyncPeriod)
	sdk.Handle(operator.NewHandler())
	sdk.Run(context.Background())
}
