package reducer

import "github.com/ilyesarf/straw/types"

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

	//k8s event clusters diff
	baseK8s := make(map[string]bool)
	for _, c := range base.K8sEventClusters {
		baseK8s[c.Type+"|"+c.Reason] = true
	}
	for _, c := range compare.K8sEventClusters {
		if !baseK8s[c.Type+"|"+c.Reason] {
			diff.NewK8sEventClusters = append(diff.NewK8sEventClusters, c)
		}
	}

	diff.PodDiffs = diffPods(base.PodSummary, compare.PodSummary)
	diff.NodeDiffs = diffNodes(base.Nodes, compare.Nodes)

	return diff
}

type podKey struct {
	name      string
	namespace string
}

func diffPods(base, compare *types.K8sPodSummary) []types.PodDiff {
	if base == nil && compare == nil {
		return nil
	}

	basePods := make(map[podKey]types.K8sPodEntry)
	if base != nil {
		for _, p := range base.Unhealthy {
			basePods[podKey{p.Name, p.Namespace}] = p
		}
	}
	comparePods := make(map[podKey]types.K8sPodEntry)
	if compare != nil {
		for _, p := range compare.Unhealthy {
			comparePods[podKey{p.Name, p.Namespace}] = p
		}
	}

	var diffs []types.PodDiff

	for k, cp := range comparePods {
		bp, existed := basePods[k]
		if !existed {
			diffs = append(diffs, types.PodDiff{
				Name: cp.Name, Namespace: cp.Namespace,
				Change: "added", NewPhase: cp.Phase, NewRestarts: cp.RestartCount,
			})
			continue
		}
		if bp.Phase != cp.Phase {
			diffs = append(diffs, types.PodDiff{
				Name: cp.Name, Namespace: cp.Namespace,
				Change: "phase_changed", OldPhase: bp.Phase, NewPhase: cp.Phase,
			})
		}
		if cp.RestartCount > bp.RestartCount {
			diffs = append(diffs, types.PodDiff{
				Name: cp.Name, Namespace: cp.Namespace,
				Change: "restarts_increased", OldRestarts: bp.RestartCount, NewRestarts: cp.RestartCount,
			})
		}
	}

	for k, bp := range basePods {
		if _, exists := comparePods[k]; !exists {
			diffs = append(diffs, types.PodDiff{
				Name: bp.Name, Namespace: bp.Namespace,
				Change: "removed", OldPhase: bp.Phase, OldRestarts: bp.RestartCount,
			})
		}
	}

	return diffs
}

func diffNodes(base, compare []types.K8sNodeEntry) []types.NodeDiff {
	baseNodes := make(map[string]types.K8sNodeEntry, len(base))
	for _, n := range base {
		baseNodes[n.Name] = n
	}
	compareNodes := make(map[string]types.K8sNodeEntry, len(compare))
	for _, n := range compare {
		compareNodes[n.Name] = n
	}

	var diffs []types.NodeDiff

	for name, cn := range compareNodes {
		bn, existed := baseNodes[name]
		if !existed {
			diffs = append(diffs, types.NodeDiff{
				Name: name, Change: "added", NewReady: cn.Ready,
			})
			continue
		}
		if bn.Ready != cn.Ready {
			diffs = append(diffs, types.NodeDiff{
				Name: name, Change: "ready_changed", OldReady: bn.Ready, NewReady: cn.Ready,
			})
		}
		if bn.MemPressure != cn.MemPressure || bn.DiskPressure != cn.DiskPressure {
			var parts []string
			if bn.MemPressure != cn.MemPressure {
				if cn.MemPressure {
					parts = append(parts, "MemPressure onset")
				} else {
					parts = append(parts, "MemPressure resolved")
				}
			}
			if bn.DiskPressure != cn.DiskPressure {
				if cn.DiskPressure {
					parts = append(parts, "DiskPressure onset")
				} else {
					parts = append(parts, "DiskPressure resolved")
				}
			}
			detail := ""
			for i, p := range parts {
				if i > 0 {
					detail += ", "
				}
				detail += p
			}
			diffs = append(diffs, types.NodeDiff{
				Name: name, Change: "pressure_changed", Detail: detail,
			})
		}
	}

	for name, bn := range baseNodes {
		if _, exists := compareNodes[name]; !exists {
			diffs = append(diffs, types.NodeDiff{
				Name: name, Change: "removed", OldReady: bn.Ready,
			})
		}
	}

	return diffs
}
