package reducer

import "straw/types"

var Thresholds = map[string]float64{
	"cpu_used_pct": 80.0,
	"net_rx_bytes": 500_000_000,
	"net_tx_bytes": 500_000_000,
}

func FilterMetrics(samples []types.MetricSample) []types.MetricSample {
	type key struct {
		entity string
		metric string
	}
	peaks := make(map[key]types.MetricSample)
	var order []key

	for _, s := range samples {
		threshold, known := Thresholds[s.Metric]
		if known && s.Value < threshold {
			continue
		}
		k := key{s.Entity, s.Metric}
		if _, exists := peaks[k]; !exists {
			order = append(order, k)
			peaks[k] = s
		} else if s.Value > peaks[k].Value {
			peaks[k] = s
		}
	}

	filtered := make([]types.MetricSample, 0, len(order))
	for _, k := range order {
		filtered = append(filtered, peaks[k])
	}

	return filtered
}
