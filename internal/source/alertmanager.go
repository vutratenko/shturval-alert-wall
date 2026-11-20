package source

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

type Alertmanager struct {
	ClusterName string
	BaseURL     string
	Client      *http.Client
}

func (a *Alertmanager) Cluster() string { return a.ClusterName }

type amAlert struct {
	Fingerprint string            `json:"fingerprint"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
}

func (a *Alertmanager) Fetch(ctx context.Context) ([]model.Alert, error) {
	client := a.Client
	if client == nil {
		client = http.DefaultClient
	}
	u, err := url.Parse(strings.TrimRight(a.BaseURL, "/"))
	if err != nil {
		return nil, err
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v2/alerts"
	q := u.Query()
	q.Set("active", "true")
	q.Set("silenced", "false")
	q.Set("inhibited", "false")
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("alertmanager %s: %s", a.ClusterName, resp.Status)
	}
	var raw []amAlert
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]model.Alert, 0, len(raw))
	for _, r := range raw {
		out = append(out, mapAMAlert(a.ClusterName, r))
	}
	return out, nil
}

func mapAMAlert(cluster string, r amAlert) model.Alert {
	labels := r.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	ann := r.Annotations
	if ann == nil {
		ann = map[string]string{}
	}
	if c := labels["cluster"]; c != "" {
		cluster = c
	}
	return model.Alert{
		Fingerprint: r.Fingerprint,
		Alertname:   labels["alertname"],
		Severity:    model.NormalizeSeverity(labels["severity"]),
		Namespace:   labels["namespace"],
		Cluster:     cluster,
		Summary:     firstNonEmpty(ann["summary"], labels["summary"]),
		Description: firstNonEmpty(ann["description"], labels["description"]),
		RunbookURL:  firstNonEmpty(ann["runbook_url"], labels["runbook_url"]),
		StartsAt:    r.StartsAt,
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}