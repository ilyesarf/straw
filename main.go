package main

import (
	"fmt"
	"os"

	"straw/reducer"
	"straw/reducer/parser"
	"straw/renderer"
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

	topo := reducer.AggregateTopology(snap.Topology)
	metrics := reducer.FilterMetrics(snap.Metrics)
	logs := reducer.ClusterLogs(snap.Logs)

	md := renderer.Render(topo, metrics, logs)
	fmt.Println(md)
}
