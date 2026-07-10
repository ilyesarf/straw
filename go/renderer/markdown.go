package renderer

import (
	"fmt"
	"strings"

	"github.com/ilyesarf/straw/types"
)

func Render(snap types.ReducedSnapshot) string {
	var sb strings.Builder

	if len(snap.Topology) > 0 {
		sb.WriteString("=== TRAFFIC TOPOLOGY ===\n")
		for _, edge := range snap.Topology {
			sb.WriteString(fmt.Sprintf("- %s -> %s [%s, reqs:%d]\n",
				edge.Key.Src, edge.Key.Dst, edge.Key.Protocol, edge.TotalCount))
		}
		sb.WriteString("\n")
	}

	if len(snap.Metrics) > 0 {
		sb.WriteString("=== RESOURCE SATURATION ===\n")
		for _, m := range snap.Metrics {
			sb.WriteString(fmt.Sprintf("- %s | %s: %.2f [ELEVATED]\n",
				m.Entity, m.Metric, m.Value))
		}
		sb.WriteString("\n")
	}

	if len(snap.LogClusters) > 0 {
		sb.WriteString("=== LOG SIGNATURES ===\n")
		for _, c := range snap.LogClusters {
			sb.WriteString(fmt.Sprintf("- count:%d | %s\n",
				c.Count, c.Pattern))
		}
		sb.WriteString("\n")
	}

	if len(snap.K8sEventClusters) > 0 {
		sb.WriteString("=== K8S EVENTS ===\n")
		for _, c := range snap.K8sEventClusters {
			sb.WriteString(fmt.Sprintf("- count:%d | [%s] %s\n",
				c.Count, c.Type, c.Reason))
		}
		sb.WriteString("\n")
	}

	if snap.PodSummary != nil {
		sb.WriteString(RenderPods(*snap.PodSummary))
		sb.WriteString("\n")
	}

	if len(snap.Nodes) > 0 {
		sb.WriteString("=== K8S NODES ===\n")
		for _, n := range snap.Nodes {
			status := "Ready"
			if !n.Ready {
				status = "NotReady"
			}
			flags := ""
			if n.MemPressure {
				flags += " MemPressure"
			}
			if n.DiskPressure {
				flags += " DiskPressure"
			}
			diskStr := ""
			if n.DiskUsedPct > 0 {
				diskStr = fmt.Sprintf(" disk:%.1f%%", n.DiskUsedPct)
			}
			sb.WriteString(fmt.Sprintf("- %s [%s%s] cpu:%.1f%% mem:%.1f%%%s (%dMi total)\n",
				n.Name, status, flags,
				n.CPUUsedPct, n.MemUsedPct, diskStr,
				n.MemCapacityB/(1024*1024)))
		}
	}

	if sb.Len() == 0 {
		return "No data."
	}

	return strings.TrimRight(sb.String(), "\n") + "\n"
}

func RenderMetrics(summary types.MetricSummary) string {
	var sb strings.Builder
	sb.WriteString("=== METRICS ===\n")
	for _, e := range summary.Entries {
		sb.WriteString(fmt.Sprintf("- %s -> %s [%s] reqs:%d errs:%d err%%:%.2f lat_avg:%.2fms lat_max:%.2fms\n",
			e.Src, e.Dst, e.Protocol, e.Reqs, e.Errs, e.ErrPct, e.LatAvgMs, e.LatMaxMs))
	}
	return sb.String()
}

func RenderPods(summary types.K8sPodSummary) string {
	var sb strings.Builder

	sb.WriteString("=== K8S PODS ===\n")
	sb.WriteString(fmt.Sprintf("total:%d running:%d unhealthy:%d\n",
		summary.Total, summary.Running, len(summary.Unhealthy)))

	for _, p := range summary.Unhealthy {
		memMi := p.MemLimitB / (1024 * 1024)
		if memMi > 0 {
			sb.WriteString(fmt.Sprintf("- %s [%s/%s] phase:%s restarts:%d memlimit:%dMi\n",
				p.Name, p.Namespace, p.Node, p.Phase, p.RestartCount, memMi))
		} else {
			sb.WriteString(fmt.Sprintf("- %s [%s/%s] phase:%s restarts:%d\n",
				p.Name, p.Namespace, p.Node, p.Phase, p.RestartCount))
		}
	}

	return sb.String()
}

func RenderDiff(diff types.SnapshotDiff) string {
	var sb strings.Builder

	sb.WriteString("=== SNAPSHOT DIFF ===\n")
	sb.WriteString(fmt.Sprintf("New Log Clusters: %d\n", len(diff.NewLogClusters)))
	for _, l := range diff.NewLogClusters {
		sb.WriteString(fmt.Sprintf("  + %s\n", l.Pattern))
	}

	sb.WriteString("\nTopology Changes:\n")
	sb.WriteString(fmt.Sprintf("  Added Edges: %d\n", len(diff.AddedEdges)))
	for _, e := range diff.AddedEdges {
		sb.WriteString(fmt.Sprintf("    + %s -> %s [%s]\n", e.Key.Src, e.Key.Dst, e.Key.Protocol))
	}
	sb.WriteString(fmt.Sprintf("  Removed Edges: %d\n", len(diff.RemovedEdges)))
	for _, e := range diff.RemovedEdges {
		sb.WriteString(fmt.Sprintf("    - %s -> %s [%s]\n", e.Key.Src, e.Key.Dst, e.Key.Protocol))
	}

	sb.WriteString("\nMetric Saturation Changes:\n")
	sb.WriteString(fmt.Sprintf("  Newly Elevated: %d\n", len(diff.AddedMetrics)))
	for _, m := range diff.AddedMetrics {
		sb.WriteString(fmt.Sprintf("    + %s | %s: %.2f\n", m.Entity, m.Metric, m.Value))
	}
	sb.WriteString(fmt.Sprintf("  Resolved: %d\n", len(diff.ResolvedMetrics)))
	for _, m := range diff.ResolvedMetrics {
		sb.WriteString(fmt.Sprintf("    - %s | %s: %.2f\n", m.Entity, m.Metric, m.Value))
	}

	if len(diff.NewK8sEventClusters) > 0 {
		sb.WriteString(fmt.Sprintf("\nNew K8s Event Patterns: %d\n", len(diff.NewK8sEventClusters)))
		for _, c := range diff.NewK8sEventClusters {
			sb.WriteString(fmt.Sprintf("  + count:%d | [%s] %s\n", c.Count, c.Type, c.Reason))
		}
	}

	if len(diff.PodDiffs) > 0 {
		sb.WriteString(fmt.Sprintf("\nPod Changes: %d\n", len(diff.PodDiffs)))
		for _, p := range diff.PodDiffs {
			switch p.Change {
			case "added":
				sb.WriteString(fmt.Sprintf("  + %s/%s [%s] restarts:%d\n", p.Namespace, p.Name, p.NewPhase, p.NewRestarts))
			case "removed":
				sb.WriteString(fmt.Sprintf("  - %s/%s (was %s)\n", p.Namespace, p.Name, p.OldPhase))
			case "phase_changed":
				sb.WriteString(fmt.Sprintf("  ~ %s/%s phase:%s->%s\n", p.Namespace, p.Name, p.OldPhase, p.NewPhase))
			case "restarts_increased":
				sb.WriteString(fmt.Sprintf("  ~ %s/%s restarts:%d->%d\n", p.Namespace, p.Name, p.OldRestarts, p.NewRestarts))
			}
		}
	}

	if len(diff.NodeDiffs) > 0 {
		sb.WriteString(fmt.Sprintf("\nNode Changes: %d\n", len(diff.NodeDiffs)))
		for _, n := range diff.NodeDiffs {
			switch n.Change {
			case "added":
				ready := "Ready"
				if !n.NewReady {
					ready = "NotReady"
				}
				sb.WriteString(fmt.Sprintf("  + %s [%s]\n", n.Name, ready))
			case "removed":
				sb.WriteString(fmt.Sprintf("  - %s\n", n.Name))
			case "ready_changed":
				from, to := "Ready", "NotReady"
				if n.NewReady {
					from, to = "NotReady", "Ready"
				}
				sb.WriteString(fmt.Sprintf("  ~ %s %s->%s\n", n.Name, from, to))
			case "pressure_changed":
				sb.WriteString(fmt.Sprintf("  ~ %s %s\n", n.Name, n.Detail))
			}
		}
	}

	return sb.String()
}
