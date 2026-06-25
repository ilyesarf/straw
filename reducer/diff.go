package reducer

import "github.com/ilyesrf/straw/types"

func Diff(base, compare types.ReducedSnapshot) types.SnapshotDiff {
	diff := types.SnapshotDiff{
		BaseTimestamp:    base.Timestamp,
		CompareTimestamp: compare.Timestamp,
	}

	//topology diff
	for k, v := range compare.Topology {
		if _, exists := base.Topology[k]; !exists {
			diff.AddedEdges = append(diff.AddedEdges, v)
		}
	}
	for k, v := range base.Topology {
		if _, exists := compare.Topology[k]; !exists {
			diff.RemovedEdges = append(diff.RemovedEdges, v)
		}
	}

	//metrics diff
	type metricKey struct {
		entity string
		metric string
	}
	baseMetrics := make(map[metricKey]bool)
	for _, m := range base.Metrics {
		baseMetrics[metricKey{m.Entity, m.Metric}] = true
	}
	compareMetrics := make(map[metricKey]bool)
	for _, m := range compare.Metrics {
		compareMetrics[metricKey{m.Entity, m.Metric}] = true
	}

	for _, m := range compare.Metrics {
		if !baseMetrics[metricKey{m.Entity, m.Metric}] {
			diff.AddedMetrics = append(diff.AddedMetrics, m)
		}
	}
	for _, m := range base.Metrics {
		if !compareMetrics[metricKey{m.Entity, m.Metric}] {
			diff.ResolvedMetrics = append(diff.ResolvedMetrics, m)
		}
	}

	//log clusters diff
	baseLogs := make(map[string]bool)
	for _, l := range base.LogClusters {
		baseLogs[l.Pattern] = true
	}

	for _, l := range compare.LogClusters {
		if !baseLogs[l.Pattern] {
			diff.NewLogClusters = append(diff.NewLogClusters, l)
		}
	}

	return diff
}
