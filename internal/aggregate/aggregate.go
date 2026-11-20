package aggregate

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/mood"
	"github.com/vutratenko/shturval-alert-wall/internal/model"
	"github.com/vutratenko/shturval-alert-wall/internal/source"
)

type ClusterFetch struct {
	Name      string
	Reachable bool
	Alerts    []model.Alert
	Err       error
}

type Event struct {
	Type    string          `json:"type"`
	Cluster string          `json:"cluster,omitempty"`
	Alert   *model.Alert    `json:"alert,omitempty"`
	At      time.Time       `json:"at"`
	Snap    *model.Snapshot `json:"snapshot,omitempty"`
}

func Build(fetches []ClusterFetch, at time.Time, th mood.Thresholds) model.Snapshot {
	clusters := make([]model.ClusterSummary, 0, len(fetches))
	var total model.Counts
	unreachable := 0
	for _, f := range fetches {
		alerts := f.Alerts
		if alerts == nil {
			alerts = []model.Alert{}
		}
		counts := model.CountAlerts(alerts)
		if !f.Reachable {
			unreachable++
		}
		total.Critical += counts.Critical
		total.Warning += counts.Warning
		total.Info += counts.Info
		clusters = append(clusters, model.ClusterSummary{
			Name: f.Name, Reachable: f.Reachable, Counts: counts, Alerts: alerts,
		})
	}
	sortClusters(clusters)
	return model.Snapshot{
		GeneratedAt: at,
		Mood:        mood.Compute(total, unreachable, th),
		Clusters:    clusters,
	}
}

func sortClusters(clusters []model.ClusterSummary) {
	sort.Slice(clusters, func(i, j int) bool {
		a, b := clusters[i], clusters[j]
		if a.Reachable != b.Reachable {
			return b.Reachable
		}
		sa := severityScore(a.Counts)
		sb := severityScore(b.Counts)
		if sa != sb {
			return sa > sb
		}
		if a.Counts.Total() != b.Counts.Total() {
			return a.Counts.Total() > b.Counts.Total()
		}
		return a.Name < b.Name
	})
}

func severityScore(c model.Counts) int {
	return c.Critical*100 + c.Warning*10 + c.Info
}

func Diff(prev, next model.Snapshot) []Event {
	now := next.GeneratedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var out []Event
	prevClusters := map[string]model.ClusterSummary{}
	nextClusters := map[string]model.ClusterSummary{}
	for _, c := range prev.Clusters {
		prevClusters[c.Name] = c
	}
	for _, c := range next.Clusters {
		nextClusters[c.Name] = c
	}
	for name, nc := range nextClusters {
		pc, ok := prevClusters[name]
		if !ok {
			if !nc.Reachable {
				out = append(out, Event{Type: "cluster.down", Cluster: name, At: now})
			}
			for i := range nc.Alerts {
				a := nc.Alerts[i]
				out = append(out, Event{Type: "alert.firing", Cluster: name, Alert: &a, At: now})
			}
			continue
		}
		if pc.Reachable && !nc.Reachable {
			out = append(out, Event{Type: "cluster.down", Cluster: name, At: now})
		}
		if !pc.Reachable && nc.Reachable {
			out = append(out, Event{Type: "cluster.up", Cluster: name, At: now})
		}
		prevFP := map[string]model.Alert{}
		for _, a := range pc.Alerts {
			prevFP[a.Fingerprint] = a
		}
		nextFP := map[string]model.Alert{}
		for _, a := range nc.Alerts {
			nextFP[a.Fingerprint] = a
		}
		for fp, a := range nextFP {
			if _, exists := prevFP[fp]; !exists {
				aa := a
				out = append(out, Event{Type: "alert.firing", Cluster: name, Alert: &aa, At: now})
			}
		}
		for fp, a := range prevFP {
			if _, exists := nextFP[fp]; !exists {
				aa := a
				out = append(out, Event{Type: "alert.resolved", Cluster: name, Alert: &aa, At: now})
			}
		}
	}
	for name, pc := range prevClusters {
		if _, ok := nextClusters[name]; !ok {
			if pc.Reachable {
				out = append(out, Event{Type: "cluster.down", Cluster: name, At: now})
			}
		}
	}
	return out
}

type Poller struct {
	Sources []source.Source
	Timeout time.Duration
	Th      mood.Thresholds
}

func (p *Poller) Poll(ctx context.Context) []ClusterFetch {
	if p.Timeout <= 0 {
		p.Timeout = 5 * time.Second
	}
	out := make([]ClusterFetch, len(p.Sources))
	var wg sync.WaitGroup
	wg.Add(len(p.Sources))
	for i, src := range p.Sources {
		i, src := i, src
		go func() {
			defer wg.Done()
			cctx, cancel := context.WithTimeout(ctx, p.Timeout)
			defer cancel()
			alerts, err := src.Fetch(cctx)
			cf := ClusterFetch{Name: src.Cluster(), Alerts: alerts, Reachable: err == nil, Err: err}
			out[i] = cf
		}()
	}
	wg.Wait()
	return out
}

func (p *Poller) Snapshot(ctx context.Context) model.Snapshot {
	return Build(p.Poll(ctx), time.Now().UTC(), p.Th)
}