package reducer

import (
	"strings"

	"github.com/ilyesarf/straw/types"
)

func FilterSnapshot(snap *types.RawSnapshot, target string) *types.RawSnapshot {
	if target == "" {
		return snap
	}

	filtered := &types.RawSnapshot{
		Timestamp: snap.Timestamp,
	}

	for _, l := range snap.Logs {
		if strings.Contains(l.Component, target) {
			filtered.Logs = append(filtered.Logs, l)
		}
	}

	for _, m := range snap.Metrics {
		if strings.Contains(m.Entity, target) {
			filtered.Metrics = append(filtered.Metrics, m)
		}
	}

	for _, e := range snap.Topology {
		if strings.Contains(e.Source, target) || strings.Contains(e.Dest, target) {
			filtered.Topology = append(filtered.Topology, e)
		}
	}

	return filtered
}
