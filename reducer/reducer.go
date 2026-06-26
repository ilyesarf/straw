package reducer

import "github.com/ilyesarf/straw/types"

func Reduce(raw *types.RawSnapshot, thresholds map[string]float64) types.ReducedSnapshot {
	return types.ReducedSnapshot{
		Timestamp:        raw.Timestamp,
		LogClusters:      ClusterLogs(raw.Logs),
		Topology:         AggregateTopology(raw.Topology),
		Metrics:          FilterMetrics(raw.Metrics, thresholds),
		K8sEventClusters: ClusterK8sEvents(raw.K8sEvents),
	}
}
