package main

import (
	"flag"
	"fmt"
	"os"

	"straw/reducer"
	"straw/reducer/parser"
	"straw/renderer"
)

func main() {
	savePath := flag.String("save", "", "save markdown output to file")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "usage: %s [--save <file>] <stream-file>\n", os.Args[0])
		os.Exit(1)
	}

	snap, err := parser.ParseStream(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	topo := reducer.AggregateTopology(snap.Topology)
	metrics := reducer.FilterMetrics(snap.Metrics)
	logs := reducer.ClusterLogs(snap.Logs)

	md := renderer.Render(topo, metrics, logs)

	if *savePath != "" {
		if err := os.WriteFile(*savePath, []byte(md), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "failed to save file: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Print(md)
}
