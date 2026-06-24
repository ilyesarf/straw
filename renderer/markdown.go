package renderer

import (
	"fmt"
	"strings"

	"straw/types"
)

func Render(topo map[types.FlowKey]types.AggregatedEdge, metrics []types.MetricSample, logs []types.LogCluster) string {
	var sb strings.Builder

	sb.WriteString("=== TRAFFIC TOPOLOGY ===\n")
	for _, edge := range topo {
		sb.WriteString(fmt.Sprintf("- %s -> %s [%s, reqs:%d]\n",
			edge.Key.Src, edge.Key.Dst, edge.Key.Protocol, edge.TotalCount))
	}
	sb.WriteString("\n")

	sb.WriteString("=== RESOURCE SATURATION ===\n")
	for _, m := range metrics {
		sb.WriteString(fmt.Sprintf("- %s | %s: %.2f [ELEVATED]\n",
			m.Entity, m.Metric, m.Value))
	}
	sb.WriteString("\n")

	sb.WriteString("=== LOG SIGNATURES ===\n")
	for _, c := range logs {
		sb.WriteString(fmt.Sprintf("- count:%d | %s\n",
			c.Count, c.Pattern))
	}

	return sb.String()
}
