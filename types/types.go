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
	//Threshold float64
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
