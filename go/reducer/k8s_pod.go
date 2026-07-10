package reducer

import "github.com/ilyesarf/straw/types"

const restartThreshold = 3

func ReducePods(pods []types.K8sPod) types.K8sPodSummary {
	summary := types.K8sPodSummary{Total: len(pods)}

	for _, p := range pods {
		if p.Phase == "Running" {
			summary.Running++
		}
		if p.Phase != "Running" || p.RestartCount >= restartThreshold {
			summary.Unhealthy = append(summary.Unhealthy, types.K8sPodEntry{
				Name:         p.Name,
				Namespace:    p.Namespace,
				Node:         p.Node,
				Phase:        p.Phase,
				RestartCount: p.RestartCount,
				MemLimitB:    p.MemLimitB,
			})
		}
	}

	sortPodEntries(summary.Unhealthy)
	return summary
}

func sortPodEntries(entries []types.K8sPodEntry) {
	for i := 1; i < len(entries); i++ {
		key := entries[i]
		j := i - 1
		for j >= 0 && entries[j].RestartCount < key.RestartCount {
			entries[j+1] = entries[j]
			j--
		}
		entries[j+1] = key
	}
}
