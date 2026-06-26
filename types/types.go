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

type K8sEvent struct {
	Timestamp    time.Time
	Namespace    string
	Type         string // "Normal" or "Warning"
	Reason       string
	Message      string
	InvolvedKind string
	InvolvedName string
}

type K8sEventCluster struct {
	Type   string
	Reason string
	Count  int
	Sample string
}

type RawSnapshot struct {
	Timestamp time.Time
	Logs      []LogLine
	Metrics   []MetricSample
	Topology  []TopologyEdge
	K8sEvents []K8sEvent
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
	Timestamp        time.Time
	LogClusters      []LogCluster
	Topology         map[FlowKey]AggregatedEdge
	Metrics          []MetricSample
	K8sEventClusters []K8sEventCluster
}

type SnapshotDiff struct {
	BaseTimestamp    time.Time
	CompareTimestamp time.Time

	NewLogClusters []LogCluster

	AddedEdges   []AggregatedEdge
	RemovedEdges []AggregatedEdge

	AddedMetrics    []MetricSample
	ResolvedMetrics []MetricSample

	NewK8sEventClusters []K8sEventCluster
}
