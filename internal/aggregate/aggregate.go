package aggregate

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
	"github.com/vutratenko/shturval-alert-wall/internal/mood"
	"github.com/vutratenko/shturval-alert-wall/internal/source"
)

type ClusterFetch struct {
	Name      string
	Source    string
	Reachable bool
	Alerts    []model.Alert
	Silenced  []model.SilencedAlert
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
		sourceName := f.Source
		if sourceName == "" {
			sourceName = f.Name
		}
		clusters = append(clusters, model.ClusterSummary{
			Name: f.Name, Source: sourceName, Reachable: f.Reachable, Counts: counts, Alerts: alerts, Silenced: f.Silenced,
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

func indexAlerts(alerts []model.Alert) map[string]model.Alert {
	out := make(map[string]model.Alert, len(alerts))
	for _, a := range alerts {
		out[a.Fingerprint] = a
	}
	return out
}

func indexSilenced(items []model.SilencedAlert) map[string]model.Alert {
	out := make(map[string]model.Alert, len(items))
	for _, s := range items {
		out[s.Fingerprint] = s.Alert
	}
	return out
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
		prevActive := indexAlerts(pc.Alerts)
		prevSilenced := indexSilenced(pc.Silenced)
		nextActive := indexAlerts(nc.Alerts)
		nextSilenced := indexSilenced(nc.Silenced)
		for fp, a := range nextActive {
			_, wasActive := prevActive[fp]
			_, wasSilenced := prevSilenced[fp]
			aa := a
			if wasSilenced {
				out = append(out, Event{Type: "alert.unsilenced", Cluster: name, Alert: &aa, At: now})
				out = append(out, Event{Type: "alert.firing", Cluster: name, Alert: &aa, At: now})
				continue
			}
			if !wasActive {
				out = append(out, Event{Type: "alert.firing", Cluster: name, Alert: &aa, At: now})
			}
		}
		for fp, a := range prevActive {
			if _, still := nextActive[fp]; still {
				continue
			}
			aa := a
			if silenced, nowSilenced := nextSilenced[fp]; nowSilenced {
				s := silenced
				out = append(out, Event{Type: "alert.silenced", Cluster: name, Alert: &s, At: now})
				continue
			}
			out = append(out, Event{Type: "alert.resolved", Cluster: name, Alert: &aa, At: now})
		}
		for fp, a := range prevSilenced {
			_, stillActive := nextActive[fp]
			_, stillSilenced := nextSilenced[fp]
			if stillActive || stillSilenced {
				continue
			}
			aa := a
			out = append(out, Event{Type: "alert.resolved", Cluster: name, Alert: &aa, At: now})
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
			batch, err := src.Fetch(cctx)
			cf := ClusterFetch{
				Name: src.Cluster(), Source: src.Cluster(),
				Alerts: batch.Alerts, Silenced: batch.Silenced,
				Reachable: err == nil, Err: err,
			}
			out[i] = cf
		}()
	}
	wg.Wait()
	return out
}

func (p *Poller) Snapshot(ctx context.Context) model.Snapshot {
	return Build(p.Poll(ctx), time.Now().UTC(), p.Th)
}
