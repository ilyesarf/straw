package types

import "time"

type LogLine struct {
	Timestamp time.Time
	Component string
	Message   string
}

type MetricSample struct {
	Entity string
	Metric string
	Value  float64
}

type TopologyEdge struct {
	Source   string
	Dest     string
	Protocol string
	Metadata map[string]string
}

type RawSnapshot struct {
	Timestamp time.Time
	Logs      []LogLine
	Metrics   []MetricSample
	Topology  []TopologyEdge
}

type LogCluster struct {
	Pattern string
	Count   int
	Sample  string
}

type FlowKey struct {
	Src      string
	Dst      string
	Protocol string
}

type AggregatedEdge struct {
	Key        FlowKey
	TotalCount int
	Operations map[string]int
}

type ReducedSnapshot struct {
	Timestamp   time.Time
	LogClusters []LogCluster
	Topology    map[FlowKey]AggregatedEdge
	Metrics     []MetricSample
}

type SnapshotDiff struct {
	BaseTimestamp    time.Time
	CompareTimestamp time.Time

	NewLogClusters []LogCluster

	AddedEdges   []AggregatedEdge
	RemovedEdges []AggregatedEdge

	AddedMetrics    []MetricSample
	ResolvedMetrics []MetricSample
}
