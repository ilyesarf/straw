package renderer

import (
	"fmt"
	"strings"

	"github.com/ilyesarf/straw/types"
)

func Render(snap types.ReducedSnapshot) string {
	var sb strings.Builder

	sb.WriteString("=== TRAFFIC TOPOLOGY ===\n")
	for _, edge := range snap.Topology {
		sb.WriteString(fmt.Sprintf("- %s -> %s [%s, reqs:%d]\n",
			edge.Key.Src, edge.Key.Dst, edge.Key.Protocol, edge.TotalCount))
	}
	sb.WriteString("\n")

	sb.WriteString("=== RESOURCE SATURATION ===\n")
	for _, m := range snap.Metrics {
		sb.WriteString(fmt.Sprintf("- %s | %s: %.2f [ELEVATED]\n",
			m.Entity, m.Metric, m.Value))
	}
	sb.WriteString("\n")

	sb.WriteString("=== LOG SIGNATURES ===\n")
	for _, c := range snap.LogClusters {
		sb.WriteString(fmt.Sprintf("- count:%d | %s\n",
			c.Count, c.Pattern))
	}

	if len(snap.K8sEventClusters) > 0 {
		sb.WriteString("\n=== K8S EVENTS ===\n")
		for _, c := range snap.K8sEventClusters {
			sb.WriteString(fmt.Sprintf("- count:%d | [%s] %s\n",
				c.Count, c.Type, c.Reason))
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

	return sb.String()
}
