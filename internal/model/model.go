package model

import (
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
	Fingerprint string    `json:"fingerprint"`
	Alertname   string    `json:"alertname"`
	Severity    Bucket    `json:"severity"`
	Namespace   string    `json:"namespace,omitempty"`
	Cluster     string    `json:"cluster,omitempty"`
	Summary     string    `json:"summary,omitempty"`
	Description string    `json:"description,omitempty"`
	RunbookURL  string    `json:"runbookUrl,omitempty"`
	StartsAt    time.Time `json:"startsAt"`
}

type Counts struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}

func (c Counts) Total() int { return c.Critical + c.Warning + c.Info }

type ClusterSummary struct {
	Name      string  `json:"name"`
	Reachable bool    `json:"reachable"`
	Counts    Counts  `json:"counts"`
	Alerts    []Alert `json:"alerts"`
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
	Clusters    []ClusterSummary `json:"clusters"`
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