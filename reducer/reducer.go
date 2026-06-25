package reducer

import "github.com/ilyesrf/straw/types"

func Reduce(raw *types.RawSnapshot) types.ReducedSnapshot {
	return types.ReducedSnapshot{
		Timestamp:   raw.Timestamp,
		LogClusters: ClusterLogs(raw.Logs),
		Topology:    AggregateTopology(raw.Topology),
		Metrics:     FilterMetrics(raw.Metrics),
	}
}
