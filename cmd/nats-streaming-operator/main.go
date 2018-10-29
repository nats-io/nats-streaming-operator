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

	"github.com/nats-io/nats-streaming-operator/internal/operator"
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

	controller := operator.NewController()
	log.Infof("Starting NATS Streaming Operator v%s", operator.Version)
	log.Infof("Go Version: %s", runtime.Version())

	err := controller.Run(context.Background())
	if err != nil && err != context.Canceled {
		log.Errorf(err.Error())
		os.Exit(1)
	}
}
