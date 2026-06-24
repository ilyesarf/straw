package reducer

import (
	"strconv"

	"github.com/ilyesrf/straw/types"
)

func AggregateTopology(edges []types.TopologyEdge) map[types.FlowKey]types.AggregatedEdge {
	result := make(map[types.FlowKey]types.AggregatedEdge)

	for _, raw := range edges {
		key := types.FlowKey{
			Src:      raw.Source,
			Dst:      raw.Dest,
			Protocol: raw.Protocol,
		}

		agg, exists := result[key]
		if !exists {
			agg = types.AggregatedEdge{
				Key:        key,
				Operations: make(map[string]int),
			}
		}

		count := 1
		if c, ok := raw.Metadata["count"]; ok {
			if parsed, err := strconv.Atoi(c); err == nil {
				count = parsed
			}
		}
		agg.TotalCount += count

		if op, ok := raw.Metadata["operation"]; ok && op != "" {
			agg.Operations[op] += count
		}

		result[key] = agg
	}

	return result
}
