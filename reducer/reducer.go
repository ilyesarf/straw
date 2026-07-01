package reducer

import "github.com/ilyesarf/straw/types"

func Reduce(raw *types.RawSnapshot, thresholds map[string]float64) types.ReducedSnapshot {
	rs := types.ReducedSnapshot{
		Timestamp:        raw.Timestamp,
		LogClusters:      ClusterLogs(raw.Logs),
		Topology:         AggregateTopology(raw.Topology),
		Metrics:          FilterMetrics(raw.Metrics, thresholds),
		K8sEventClusters: ClusterK8sEvents(raw.K8sEvents),
	}

	if len(raw.Pods) > 0 {
		summary := ReducePods(raw.Pods)
		rs.PodSummary = &summary
	}

	for _, n := range raw.Nodes {
		var memUsedPct float64
		if n.MemTotalB > 0 {
			memUsedPct = float64(n.MemTotalB-n.MemAvailableB) / float64(n.MemTotalB) * 100
		}
		rs.Nodes = append(rs.Nodes, types.K8sNodeEntry{
			Name: n.Name, Ready: n.Ready,
			MemPressure: n.MemPressure, DiskPressure: n.DiskPressure,
			CPUCapacityM: n.CPUCapacityM, CPUAllocatableM: n.CPUAllocatableM,
			MemCapacityB: n.MemCapacityB, MemAllocatableB: n.MemAllocatableB,
			CPUUsedPct: n.CPUUsedPct, MemUsedPct: memUsedPct,
		})
	}

	return rs
}
