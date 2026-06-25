package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyesrf/straw/reducer"
	"github.com/ilyesrf/straw/reducer/parser"
	"github.com/ilyesrf/straw/renderer"
	"github.com/ilyesrf/straw/types"
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

	// split data into two halves to simulate two time windows
	midLogs := len(snap.Logs) / 2
	midMetrics := len(snap.Metrics) / 2
	midTopo := len(snap.Topology) / 2

	snap1 := &types.RawSnapshot{
		Timestamp: snap.Timestamp,
		Logs:      snap.Logs[:midLogs],
		Metrics:   snap.Metrics[:midMetrics],
		Topology:  snap.Topology[:midTopo],
	}

	snap2 := &types.RawSnapshot{
		Timestamp: snap.Timestamp.Add(1 * time.Minute),
		Logs:      snap.Logs[midLogs:],
		Metrics:   snap.Metrics[midMetrics:],
		Topology:  snap.Topology[midTopo:],
	}

	thresholds := map[string]float64{
		"cpu_used_pct": 80.0,
		"net_rx_bytes": 500_000_000,
		"net_tx_bytes": 500_000_000,
	}

	reduced1 := reducer.Reduce(snap1, thresholds)
	reduced2 := reducer.Reduce(snap2, thresholds)

	diff := reducer.Diff(reduced1, reduced2)

	fmt.Print(renderer.RenderDiff(diff))
}
