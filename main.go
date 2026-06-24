package main

import (
	"fmt"
	"os"
	"strings"

	"straw/reducer/parser"
)

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

	fmt.Printf("=== Snapshot @ %s ===\n\n", snap.Timestamp.Format("15:04:05"))

	// --- Logs ---
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

	// --- Metrics ---
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

	// --- Topology ---
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
