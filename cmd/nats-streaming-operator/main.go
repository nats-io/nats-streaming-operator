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
	if os.Getenv("DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
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
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Fatalf("Failed to get watch namespace: %v", err)
	}

	// TODO: Move to constants
	resyncPeriod := 5
	log.Infof("Watching %s, %s, %s, %d", resource, kind, namespace, resyncPeriod)

	// Look for updates on the NatsStreamingClusters made on this namespace.
	sdk.Watch(resource, kind, namespace, resyncPeriod)
	sdk.Handle(operator.NewHandler())
	sdk.Run(context.Background())
}
