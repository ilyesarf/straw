package reducer

import "github.com/ilyesarf/straw/types"

func AggregateMetrics(rows []types.MetricRow) types.MetricSummary {
	type key struct{ src, dst, proto string }

	type totals struct {
		reqs     uint64
		errs     uint64
		latSumNs int64
		latMaxNs int64
	}

	aggs := make(map[key]*totals)
	order := []key{}

	for _, r := range rows {
		k := key{r.Src, r.Dst, r.Protocol}
		a, ok := aggs[k]
		if !ok {
			a = &totals{}
			aggs[k] = a
			order = append(order, k)
		}
		a.reqs += r.ReqCount
		a.errs += r.ErrCount
		a.latSumNs += r.LatSumNs
		if r.LatMaxNs > a.latMaxNs {
			a.latMaxNs = r.LatMaxNs
		}
	}

	summary := types.MetricSummary{Entries: make([]types.MetricEntry, 0, len(order))}
	for _, k := range order {
		a := aggs[k]
		entry := types.MetricEntry{
			Src:      k.src,
			Dst:      k.dst,
			Protocol: k.proto,
			Reqs:     a.reqs,
			Errs:     a.errs,
		}
		if a.reqs > 0 {
			entry.ErrPct = float64(a.errs) / float64(a.reqs) * 100
			entry.LatAvgMs = float64(a.latSumNs) / float64(a.reqs) / 1e6
		}
		entry.LatMaxMs = float64(a.latMaxNs) / 1e6
		summary.Entries = append(summary.Entries, entry)
	}
	return summary
}
