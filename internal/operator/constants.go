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
