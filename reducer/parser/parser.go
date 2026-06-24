package parser

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
)

type streamLogs struct {
	Entries []struct {
		TsUnixNs string `json:"tsUnixNs"`
		Service  string `json:"service"`
		Message  string `json:"message"`
		Stream   string `json:"stream"`
	} `json:"entries"`
}

type streamMetrics struct {
	ContainerResources []struct {
		Container  string `json:"container"`
		Node       string `json:"node"`
		TsUnix     string `json:"tsUnix"`
		NetRxBytes string `json:"netRxBytes"`
		NetTxBytes string `json:"netTxBytes"`
	} `json:"containerResources"`
	HostMetrics []struct {
		Node       string  `json:"node"`
		TsUnix     string  `json:"tsUnix"`
		CpuUsedPct float64 `json:"cpuUsedPct"`
	} `json:"hostMetrics"`
}

type streamEvents struct {
	Events []struct {
		SrcService string `json:"srcService"`
		DstService string `json:"dstService"`
		Protocol   string `json:"protocol"`
		Operation  string `json:"operation"`
		Status     int    `json:"status"`
		Count      int    `json:"count"`
	} `json:"events"`
	Errors []struct {
		SrcService string `json:"srcService"`
		DstService string `json:"dstService"`
		Method     string `json:"method"`
		Status     int    `json:"status"`
	} `json:"errors"`
}

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

func ParseStream(filepath string) (*RawSnapshot, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	snapshot := &RawSnapshot{
		Timestamp: time.Now(),
		Logs:      []LogLine{},
		Metrics:   []MetricSample{},
		Topology:  []TopologyEdge{},
	}

	scanner := bufio.NewScanner(file)
	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			dataRaw := strings.TrimPrefix(line, "data: ")

			switch currentEvent {
			case "logs":
				var l streamLogs
				if err := json.Unmarshal([]byte(dataRaw), &l); err == nil {
					for _, entry := range l.Entries {
						snapshot.Logs = append(snapshot.Logs, LogLine{
							Component: entry.Service,
							Message:   entry.Message,
						})
					}
				}
			case "metrics":
				var m streamMetrics
				if err := json.Unmarshal([]byte(dataRaw), &m); err == nil {
					for _, hm := range m.HostMetrics {
						snapshot.Metrics = append(snapshot.Metrics, MetricSample{
							Entity: hm.Node,
							Metric: "cpu_used_pct",
							Value:  hm.CpuUsedPct,
						})
					}
					for _, cr := range m.ContainerResources {
						rx, _ := strconv.ParseFloat(cr.NetRxBytes, 64)
						snapshot.Metrics = append(snapshot.Metrics, MetricSample{
							Entity: cr.Container,
							Metric: "net_rx_bytes",
							Value:  rx,
						})
					}
				}
			case "events":
				var e streamEvents
				if err := json.Unmarshal([]byte(dataRaw), &e); err == nil {
					for _, ev := range e.Events {
						dest := ev.DstService
						if dest == "" {
							dest = "external" // catch missing destinations
						}
						snapshot.Topology = append(snapshot.Topology, TopologyEdge{
							Source:   ev.SrcService,
							Dest:     dest,
							Protocol: ev.Protocol,
							Metadata: map[string]string{
								"operation": ev.Operation,
								"count":     strconv.Itoa(ev.Count),
							},
						})
					}
				}
			}
		}
	}

	return snapshot, scanner.Err()
}
