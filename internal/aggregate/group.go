package aggregate

import (
	"sort"
	"strings"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

var workloadKinds = []string{"statefulset", "deployment", "daemonset", "job"}

// Group folds firing alerts of one cluster into episodes. A non-positive window returns no groups.
func Group(cluster string, alerts []model.Alert, window time.Duration) []model.Group {
	if window <= 0 || len(alerts) == 0 {
		return nil
	}
	type bucket struct {
		kind, value string
		alerts      []model.Alert
	}
	order := make([]string, 0)
	buckets := map[string]*bucket{}
	for _, alert := range alerts {
		kind, value := resourceKey(alert)
		id := kind + "\x00" + value
		b, ok := buckets[id]
		if !ok {
			b = &bucket{kind: kind, value: value}
			buckets[id] = b
			order = append(order, id)
		}
		b.alerts = append(b.alerts, alert)
	}
	groups := make([]model.Group, 0, len(alerts))
	for _, id := range order {
		b := buckets[id]
		groups = append(groups, episodes(cluster, b.kind, b.value, b.alerts, window)...)
	}
	sortGroups(groups)
	return groups
}

func resourceKey(alert model.Alert) (kind, value string) {
	if node := label(alert, "node"); node != "" {
		return "node", node
	}
	if pod := label(alert, "pod"); pod != "" {
		return "pod", namespaceOf(alert) + "/" + pod
	}
	for _, kind := range workloadKinds {
		if name := label(alert, kind); name != "" {
			return "workload", namespaceOf(alert) + "/" + kind + "/" + name
		}
	}
	return "alert", alert.Fingerprint
}

func episodes(cluster, kind, value string, alerts []model.Alert, window time.Duration) []model.Group {
	sorted := append([]model.Alert(nil), alerts...)
	sort.Slice(sorted, func(i, j int) bool {
		if !sorted[i].StartsAt.Equal(sorted[j].StartsAt) {
			return sorted[i].StartsAt.Before(sorted[j].StartsAt)
		}
		return sorted[i].Fingerprint < sorted[j].Fingerprint
	})
	var chunks [][]model.Alert
	for _, alert := range sorted {
		placed := false
		if kind != "alert" {
			for i := range chunks {
				anchor := chunks[i][0]
				if !alert.StartsAt.After(anchor.StartsAt.Add(window)) {
					chunks[i] = append(chunks[i], alert)
					placed = true
					break
				}
			}
		}
		if !placed {
			chunks = append(chunks, []model.Alert{alert})
		}
	}
	out := make([]model.Group, 0, len(chunks))
	for _, chunk := range chunks {
		anchor := chunk[0]
		members := append([]model.Alert(nil), chunk...)
		sortMembers(members)
		root := members[0]
		out = append(out, model.Group{
			ID:       cluster + "\x00" + kind + "\x00" + value + "\x00" + anchor.Fingerprint,
			KeyKind:  kind,
			KeyValue: value,
			Severity: root.Severity,
			StartsAt: anchor.StartsAt,
			Root:     root,
			Members:  members,
		})
	}
	return out
}

func sortGroups(groups []model.Group) {
	sort.Slice(groups, func(i, j int) bool {
		ri, rj := severityRank(groups[i].Severity), severityRank(groups[j].Severity)
		if ri != rj {
			return ri < rj
		}
		if !groups[i].StartsAt.Equal(groups[j].StartsAt) {
			return groups[i].StartsAt.Before(groups[j].StartsAt)
		}
		return groups[i].ID < groups[j].ID
	})
}

func sortMembers(alerts []model.Alert) {
	sort.Slice(alerts, func(i, j int) bool {
		ri, rj := severityRank(alerts[i].Severity), severityRank(alerts[j].Severity)
		if ri != rj {
			return ri < rj
		}
		if !alerts[i].StartsAt.Equal(alerts[j].StartsAt) {
			return alerts[i].StartsAt.Before(alerts[j].StartsAt)
		}
		return alerts[i].Fingerprint < alerts[j].Fingerprint
	})
}

func severityRank(bucket model.Bucket) int {
	switch bucket {
	case model.BucketCritical:
		return 0
	case model.BucketWarning:
		return 1
	default:
		return 2
	}
}

func countGroups(groups []model.Group) model.Counts {
	var counts model.Counts
	for _, group := range groups {
		switch group.Severity {
		case model.BucketCritical:
			counts.Critical++
		case model.BucketWarning:
			counts.Warning++
		default:
			counts.Info++
		}
	}
	return counts
}

func label(alert model.Alert, key string) string {
	return strings.TrimSpace(alert.Labels[key])
}

func namespaceOf(alert model.Alert) string {
	if ns := strings.TrimSpace(alert.Namespace); ns != "" {
		return ns
	}
	return label(alert, "namespace")
}
