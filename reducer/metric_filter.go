package reducer

import "github.com/ilyesrf/straw/types"

func FilterMetrics(samples []types.MetricSample, thresholds map[string]float64) []types.MetricSample {
	type key struct {
		entity string
		metric string
	}
	peaks := make(map[key]types.MetricSample)
	var order []key

	for _, s := range samples {
		threshold, known := thresholds[s.Metric]
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
