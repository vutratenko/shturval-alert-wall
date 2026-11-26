package aggregate

import "github.com/vutratenko/shturval-alert-wall/internal/model"

// ExpandByClusterLabel names tiles from the alert label `cluster` for a single
// central Alertmanager, including when every alert belongs to one cluster.
// Multiple sources stay one row each. Source stays the Alertmanager key.
func ExpandByClusterLabel(fetches []ClusterFetch) []ClusterFetch {
	if len(fetches) != 1 {
		return fetches
	}
	f := fetches[0]
	if len(f.Alerts) == 0 && len(f.Silenced) == 0 {
		return fetches
	}
	type bucket struct {
		alerts   []model.Alert
		silenced []model.SilencedAlert
	}
	byCluster := map[string]*bucket{}
	ensure := func(name string) *bucket {
		if name == "" {
			name = "unknown"
		}
		b, ok := byCluster[name]
		if !ok {
			b = &bucket{}
			byCluster[name] = b
		}
		return b
	}
	for _, a := range f.Alerts {
		b := ensure(a.Cluster)
		b.alerts = append(b.alerts, a)
	}
	for _, a := range f.Silenced {
		b := ensure(a.Cluster)
		b.silenced = append(b.silenced, a)
	}
	sourceName := f.Source
	if sourceName == "" {
		sourceName = f.Name
	}
	out := make([]ClusterFetch, 0, len(byCluster))
	for name, b := range byCluster {
		out = append(out, ClusterFetch{
			Name: name, Source: sourceName, Reachable: f.Reachable,
			Alerts: b.alerts, Silenced: b.silenced, Err: f.Err,
		})
	}
	return out
}
