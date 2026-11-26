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
	Group   *model.Group    `json:"group,omitempty"`
	At      time.Time       `json:"at"`
	Snap    *model.Snapshot `json:"snapshot,omitempty"`
}

func Build(fetches []ClusterFetch, at time.Time, th mood.Thresholds, window time.Duration) model.Snapshot {
	clusters := make([]model.ClusterSummary, 0, len(fetches))
	var total model.Counts
	unreachable := 0
	for _, f := range fetches {
		alerts := f.Alerts
		if alerts == nil {
			alerts = []model.Alert{}
		}
		var groups []model.Group
		counts := model.CountAlerts(alerts)
		if window > 0 {
			groups = Group(f.Name, alerts, window)
			counts = countGroups(groups)
		}
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
			Name: f.Name, Source: sourceName, Reachable: f.Reachable, Counts: counts, Alerts: alerts, Groups: groups, Silenced: f.Silenced,
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
			if len(nc.Groups) > 0 {
				out = append(out, openGroups(name, nc.Groups, now)...)
			} else {
				for i := range nc.Alerts {
					a := nc.Alerts[i]
					out = append(out, Event{Type: "alert.firing", Cluster: name, Alert: &a, At: now})
				}
			}
			continue
		}
		if pc.Reachable && !nc.Reachable {
			out = append(out, Event{Type: "cluster.down", Cluster: name, At: now})
		}
		if !pc.Reachable && nc.Reachable {
			out = append(out, Event{Type: "cluster.up", Cluster: name, At: now})
		}
		grouped := len(pc.Groups) > 0 || len(nc.Groups) > 0
		prevActive := indexAlerts(pc.Alerts)
		prevSilenced := indexSilenced(pc.Silenced)
		nextActive := indexAlerts(nc.Alerts)
		nextSilenced := indexSilenced(nc.Silenced)
		if grouped {
			out = append(out, diffGroups(name, pc, nc, prevActive, nextActive, nextSilenced, now)...)
		}
		out = append(out, diffAlerts(name, prevActive, prevSilenced, nextActive, nextSilenced, now, grouped)...)
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

func openGroups(cluster string, groups []model.Group, at time.Time) []Event {
	out := make([]Event, 0, len(groups))
	for i := range groups {
		g := groups[i]
		root := g.Root
		out = append(out, Event{Type: "group.opened", Cluster: cluster, Alert: &root, Group: &g, At: at})
	}
	return out
}

func diffGroups(cluster string, prev, next model.ClusterSummary, prevActive, nextActive, nextSilenced map[string]model.Alert, at time.Time) []Event {
	var out []Event
	for i := range next.Groups {
		g := next.Groups[i]
		if !overlaps(g.Members, prevActive) {
			root := g.Root
			out = append(out, Event{Type: "group.opened", Cluster: cluster, Alert: &root, Group: &g, At: at})
			continue
		}
		if prevSev, ok := overlapSeverity(prev, g.Members); ok && severityRank(g.Severity) < severityRank(prevSev) {
			root := g.Root
			out = append(out, Event{Type: "group.promoted", Cluster: cluster, Alert: &root, Group: &g, At: at})
		}
	}
	for i := range prev.Groups {
		g := prev.Groups[i]
		if overlaps(g.Members, nextActive) || overlaps(g.Members, nextSilenced) {
			continue
		}
		root := g.Root
		out = append(out, Event{Type: "group.resolved", Cluster: cluster, Alert: &root, Group: &g, At: at})
	}
	return out
}

func overlaps(members []model.Alert, fps map[string]model.Alert) bool {
	for _, member := range members {
		if _, ok := fps[member.Fingerprint]; ok {
			return true
		}
	}
	return false
}

func overlapSeverity(prev model.ClusterSummary, members []model.Alert) (model.Bucket, bool) {
	want := map[string]struct{}{}
	for _, member := range members {
		want[member.Fingerprint] = struct{}{}
	}
	best := model.BucketInfo
	found := false
	consider := func(sev model.Bucket) {
		if !found || severityRank(sev) < severityRank(best) {
			best = sev
			found = true
		}
	}
	if len(prev.Groups) > 0 {
		for _, group := range prev.Groups {
			for _, member := range group.Members {
				if _, ok := want[member.Fingerprint]; ok {
					consider(group.Severity)
					break
				}
			}
		}
		return best, found
	}
	for _, alert := range prev.Alerts {
		if _, ok := want[alert.Fingerprint]; ok {
			consider(alert.Severity)
		}
	}
	return best, found
}

func diffAlerts(cluster string, prevActive, prevSilenced, nextActive, nextSilenced map[string]model.Alert, at time.Time, grouped bool) []Event {
	var out []Event
	for fp, alert := range nextActive {
		_, wasActive := prevActive[fp]
		_, wasSilenced := prevSilenced[fp]
		aa := alert
		if wasSilenced {
			out = append(out, Event{Type: "alert.unsilenced", Cluster: cluster, Alert: &aa, At: at})
			if !grouped {
				out = append(out, Event{Type: "alert.firing", Cluster: cluster, Alert: &aa, At: at})
			}
			continue
		}
		if !wasActive && !grouped {
			out = append(out, Event{Type: "alert.firing", Cluster: cluster, Alert: &aa, At: at})
		}
	}
	for fp, alert := range prevActive {
		if _, still := nextActive[fp]; still {
			continue
		}
		if silenced, nowSilenced := nextSilenced[fp]; nowSilenced {
			s := silenced
			out = append(out, Event{Type: "alert.silenced", Cluster: cluster, Alert: &s, At: at})
			continue
		}
		if !grouped {
			aa := alert
			out = append(out, Event{Type: "alert.resolved", Cluster: cluster, Alert: &aa, At: at})
		}
	}
	if grouped {
		return out
	}
	for fp, alert := range prevSilenced {
		_, stillActive := nextActive[fp]
		_, stillSilenced := nextSilenced[fp]
		if stillActive || stillSilenced {
			continue
		}
		aa := alert
		out = append(out, Event{Type: "alert.resolved", Cluster: cluster, Alert: &aa, At: at})
	}
	return out
}

type Poller struct {
	Sources     []source.Source
	Timeout     time.Duration
	Th          mood.Thresholds
	GroupWindow time.Duration
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
	return Build(p.Poll(ctx), time.Now().UTC(), p.Th, p.GroupWindow)
}
