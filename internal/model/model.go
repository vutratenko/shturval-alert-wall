package model

import (
	"sort"
	"strings"
	"time"
)

type Bucket string

const (
	BucketCritical Bucket = "critical"
	BucketWarning  Bucket = "warning"
	BucketInfo     Bucket = "info"
)

type Alert struct {
	Fingerprint  string            `json:"fingerprint"`
	Alertname    string            `json:"alertname"`
	Severity     Bucket            `json:"severity"`
	Namespace    string            `json:"namespace,omitempty"`
	Cluster      string            `json:"cluster,omitempty"`
	Summary      string            `json:"summary,omitempty"`
	Description  string            `json:"description,omitempty"`
	RunbookURL   string            `json:"runbookUrl,omitempty"`
	GeneratorURL string            `json:"generatorUrl,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	StartsAt     time.Time         `json:"startsAt"`
}

type Matcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"isRegex"`
	IsEqual bool   `json:"isEqual"`
}

type SilenceRef struct {
	ID        string    `json:"id"`
	EndsAt    time.Time `json:"endsAt"`
	CreatedBy string    `json:"createdBy,omitempty"`
	Comment   string    `json:"comment,omitempty"`
	Matchers  []Matcher `json:"matchers,omitempty"`
}

type SilencedAlert struct {
	Alert
	Silences []SilenceRef `json:"silences"`
}

// SilenceDraft is a silence the wall asks Alertmanager to create.
type SilenceDraft struct {
	Matchers  []Matcher
	StartsAt  time.Time
	EndsAt    time.Time
	CreatedBy string
	Comment   string
}

type Counts struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}

func (c Counts) Total() int { return c.Critical + c.Warning + c.Info }

// Group is one firing episode inside a cluster. Members share a node, pod, or workload
// and started within the group window. Counts on the wall use groups, not raw alerts.
type Group struct {
	ID       string    `json:"id"`
	KeyKind  string    `json:"keyKind"`
	KeyValue string    `json:"keyValue"`
	Severity Bucket    `json:"severity"`
	StartsAt time.Time `json:"startsAt"`
	Root     Alert     `json:"root"`
	Members  []Alert   `json:"members"`
}

type ClusterSummary struct {
	Name      string          `json:"name"`
	Source    string          `json:"source,omitempty"`
	Reachable bool            `json:"reachable"`
	Counts    Counts          `json:"counts"`
	Alerts    []Alert         `json:"alerts"`
	Groups    []Group         `json:"groups,omitempty"`
	Silenced  []SilencedAlert `json:"silenced,omitempty"`
}

type Mood string

const (
	MoodCalm    Mood = "calm"
	MoodWorried Mood = "worried"
	MoodPanic   Mood = "panic"
)

type Snapshot struct {
	GeneratedAt time.Time        `json:"generatedAt"`
	Mood        Mood             `json:"mood"`
	Silence     bool             `json:"silence"`
	Clusters    []ClusterSummary `json:"clusters"`
}

// MatchersFromLabels builds equality matchers for every non-empty label.
func MatchersFromLabels(labels map[string]string) []Matcher {
	if len(labels) == 0 {
		return nil
	}
	keys := make([]string, 0, len(labels))
	for k, v := range labels {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Matcher, 0, len(keys))
	for _, k := range keys {
		out = append(out, Matcher{Name: k, Value: labels[k], IsEqual: true})
	}
	return out
}

// SameMatchers reports whether both sets describe the same equality rules.
func SameMatchers(a, b []Matcher) bool {
	if len(a) == 0 || len(a) != len(b) {
		return false
	}
	left := matcherKeys(a)
	right := matcherKeys(b)
	if len(left) != len(a) || len(right) != len(b) {
		return false
	}
	for k := range left {
		if _, ok := right[k]; !ok {
			return false
		}
	}
	return true
}

func matcherKeys(ms []Matcher) map[string]struct{} {
	out := make(map[string]struct{}, len(ms))
	for _, m := range ms {
		eq, re := "0", "0"
		if m.IsEqual {
			eq = "1"
		}
		if m.IsRegex {
			re = "1"
		}
		out[m.Name+"\x00"+m.Value+"\x00"+eq+re] = struct{}{}
	}
	return out
}

// NormalizeSeverity maps Shturval's eight severity labels to three display buckets.
func NormalizeSeverity(raw string) Bucket {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "emergency", "alert", "critical":
		return BucketCritical
	case "error", "warning":
		return BucketWarning
	default:
		return BucketInfo
	}
}

func CountAlerts(alerts []Alert) Counts {
	var c Counts
	for _, a := range alerts {
		switch a.Severity {
		case BucketCritical:
			c.Critical++
		case BucketWarning:
			c.Warning++
		default:
			c.Info++
		}
	}
	return c
}
