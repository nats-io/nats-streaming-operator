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

const (
	// DefaultNATSStreamingImage is the default image
	// of NATS Streaming that will be used, meant to be
	// the latest release available.
	DefaultNATSStreamingImage = "nats-streaming:0.10.2"

	// DefaultNATSStreamingClusterSize is the default size
	// for the cluster.  Clustering is done via Raft so
	// an odd number of pods is recommended.
	DefaultNATSStreamingClusterSize = 3
)
