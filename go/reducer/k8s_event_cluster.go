package reducer

import "github.com/ilyesarf/straw/types"

func ClusterK8sEvents(events []types.K8sEvent) []types.K8sEventCluster {
	type key struct{ typ, reason string }

	counts := make(map[key]int)
	samples := make(map[key]string)

	for _, e := range events {
		k := key{e.Type, e.Reason}
		counts[k]++
		if _, exists := samples[k]; !exists {
			samples[k] = e.Message
		}
	}

	clusters := make([]types.K8sEventCluster, 0, len(counts))
	for k, count := range counts {
		clusters = append(clusters, types.K8sEventCluster{
			Type:   k.typ,
			Reason: k.reason,
			Count:  count,
			Sample: samples[k],
		})
	}

	sortK8sClusters(clusters)
	return clusters
}

func sortK8sClusters(clusters []types.K8sEventCluster) {
	for i := 1; i < len(clusters); i++ {
		key := clusters[i]
		j := i - 1
		for j >= 0 && clusters[j].Count < key.Count {
			clusters[j+1] = clusters[j]
			j--
		}
		clusters[j+1] = key
	}
}
