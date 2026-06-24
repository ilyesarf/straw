package reducer

import "straw/types"

var Thresholds = map[string]float64{
	"cpu_used_pct": 80.0,
	"net_rx_bytes": 500_000_000,
	"net_tx_bytes": 500_000_000,
}

func FilterMetrics(samples []types.MetricSample) []types.MetricSample {
	filtered := make([]types.MetricSample, 0)

	for _, s := range samples {
		threshold, known := Thresholds[s.Metric]
		if !known {
			filtered = append(filtered, s)
			continue
		}
		if s.Value >= threshold {
			filtered = append(filtered, s)
		}
	}

	return filtered
}
