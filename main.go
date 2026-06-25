package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyesrf/straw/reducer"
	"github.com/ilyesrf/straw/reducer/parser"
	"github.com/ilyesrf/straw/renderer"
	"github.com/ilyesrf/straw/types"
)

func main() {
	savePath := flag.String("save", "", "save markdown output to file")
	target := flag.String("target", "", "filter snapshot to specific component or node")
	diffMode := flag.Bool("diff", false, "run in diff mode (splits snapshot in half)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s [--save <file>] [--target <name>] <stream-file>\n", os.Args[0])
		os.Exit(1)
	}

	snap, err := parser.ParseStream(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	snap = reducer.FilterSnapshot(snap, *target)

	thresholds := map[string]float64{
		"cpu_used_pct": 80.0,
		"net_rx_bytes": 500_000_000,
		"net_tx_bytes": 500_000_000,
	}

	var md string
	if *diffMode {
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

		reduced1 := reducer.Reduce(snap1, thresholds)
		reduced2 := reducer.Reduce(snap2, thresholds)
		diff := reducer.Diff(reduced1, reduced2)
		md = renderer.RenderDiff(diff)
	} else {
		reduced := reducer.Reduce(snap, thresholds)
		md = renderer.Render(reduced)
	}

	if *savePath != "" {
		if err := os.WriteFile(*savePath, []byte(md), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "failed to save file: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Print(md)
}
