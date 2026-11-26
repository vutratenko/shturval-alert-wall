package aggregate

import "github.com/vutratenko/shturval-alert-wall/internal/source"

// ExpandByClusterLabel splits a single reachable fetch into one row per alert label `cluster`
// (central Alertmanager on the management cluster). Multiple sources stay one row each.
func ExpandByClusterLabel(fetches []ClusterFetch) []ClusterFetch {
	if len(fetches) != 1 {
		return fetches
	}
	f := fetches[0]
	if len(f.Alerts) == 0 {
		return fetches
	}
	byCluster := source.SplitByCluster(f.Alerts)
	if len(byCluster) <= 1 {
		return fetches
	}
	out := make([]ClusterFetch, 0, len(byCluster))
	for name, alerts := range byCluster {
		out = append(out, ClusterFetch{Name: name, Reachable: f.Reachable, Alerts: alerts, Err: f.Err})
	}
	return out
}