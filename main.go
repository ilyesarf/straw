package main

import (
	"fmt"
	"os"
	"strings"

	"straw/reducer"
	"straw/reducer/parser"
	"straw/types"
)

func PrintRawSnapshot(snap *types.RawSnapshot) {
	fmt.Printf("=== Raw Snapshot @ %s ===\n\n", snap.Timestamp.Format("15:04:05"))

	fmt.Printf("Logs: %d entries\n", len(snap.Logs))
	if len(snap.Logs) > 0 {
		shown := min(len(snap.Logs), 5)
		for i, l := range snap.Logs[:shown] {
			msg := l.Message
			if len(msg) > 100 {
				msg = msg[:100] + "..."
			}
			fmt.Printf("  [%d] component=%s  msg=%s\n", i, l.Component, msg)
		}
		if len(snap.Logs) > shown {
			fmt.Printf("  ... and %d more\n", len(snap.Logs)-shown)
		}
	}

	fmt.Printf("\nMetrics: %d samples\n", len(snap.Metrics))
	if len(snap.Metrics) > 0 {
		shown := min(len(snap.Metrics), 10)
		for i, m := range snap.Metrics[:shown] {
			fmt.Printf("  [%d] entity=%-30s  metric=%-15s  value=%.2f\n", i, m.Entity, m.Metric, m.Value)
		}
		if len(snap.Metrics) > shown {
			fmt.Printf("  ... and %d more\n", len(snap.Metrics)-shown)
		}
	}

	fmt.Printf("\nTopology: %d edges\n", len(snap.Topology))
	if len(snap.Topology) > 0 {
		shown := min(len(snap.Topology), 10)
		for i, e := range snap.Topology[:shown] {
			var meta []string
			for k, v := range e.Metadata {
				meta = append(meta, fmt.Sprintf("%s=%s", k, v))
			}
			fmt.Printf("  [%d] %s --%s--> %s  {%s}\n", i, e.Source, e.Protocol, e.Dest, strings.Join(meta, ", "))
		}
		if len(snap.Topology) > shown {
			fmt.Printf("  ... and %d more\n", len(snap.Topology)-shown)
		}
	}

	fmt.Println()
}

func PrintReducedSnapshot(snap *types.RawSnapshot) {
	fmt.Println("=== Reduced Snapshot ===\n")

	// 1. log clusters
	clusters := reducer.ClusterLogs(snap.Logs)
	fmt.Printf("Log Clusters: %d unique patterns (from %d raw lines)\n", len(clusters), len(snap.Logs))
	shown := min(len(clusters), 15)
	for i, c := range clusters[:shown] {
		pattern := c.Pattern
		if len(pattern) > 90 {
			pattern = pattern[:90] + "..."
		}
		fmt.Printf("  [%d] count=%-5d  pattern=%s\n", i, c.Count, pattern)
	}
	if len(clusters) > shown {
		fmt.Printf("  ... and %d more patterns\n", len(clusters)-shown)
	}

	// 2. aggregated topology
	topo := reducer.AggregateTopology(snap.Topology)
	fmt.Printf("\nTopology: %d unique flows (from %d raw edges)\n", len(topo), len(snap.Topology))
	i := 0
	for _, edge := range topo {
		if i >= 15 {
			fmt.Printf("  ... and %d more flows\n", len(topo)-15)
			break
		}
		fmt.Printf("  %s --%s--> %s  total_reqs=%d\n",
			edge.Key.Src, edge.Key.Protocol, edge.Key.Dst, edge.TotalCount)
		i++
	}

	// 3. filtered metrics
	filtered := reducer.FilterMetrics(snap.Metrics)
	fmt.Printf("\nMetrics: %d elevated (from %d raw samples)\n", len(filtered), len(snap.Metrics))
	for i, m := range filtered {
		if i >= 15 {
			fmt.Printf("  ... and %d more\n", len(filtered)-15)
			break
		}
		fmt.Printf("  entity=%-30s  metric=%-15s  value=%.2f\n", m.Entity, m.Metric, m.Value)
	}
	if len(filtered) == 0 {
		fmt.Println("  (all metrics within normal thresholds)")
	}

	fmt.Println()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <stream-file>\n", os.Args[0])
		os.Exit(1)
	}

	snap, err := parser.ParseStream(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	PrintRawSnapshot(snap)
	PrintReducedSnapshot(snap)
}
