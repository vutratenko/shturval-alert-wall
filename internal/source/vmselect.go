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

// VMSelect queries VictoriaMetrics for ALERTS time series in firing state.
type VMSelect struct {
	SelectURL string
	Client    *http.Client
}

func (v *VMSelect) Cluster() string { return "all" }

type vmQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []any             `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func (v *VMSelect) Fetch(ctx context.Context) ([]model.Alert, error) {
	client := v.Client
	if client == nil {
		client = http.DefaultClient
	}
	u, err := url.Parse(strings.TrimRight(v.SelectURL, "/"))
	if err != nil {
		return nil, err
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1/query"
	q := u.Query()
	q.Set("query", `ALERTS{alertstate="firing"}`)
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
		return nil, fmt.Errorf("vmselect: %s", resp.Status)
	}
	var raw vmQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if raw.Status != "success" {
		return nil, fmt.Errorf("vmselect: status %s", raw.Status)
	}
	out := make([]model.Alert, 0, len(raw.Data.Result))
	for _, r := range raw.Data.Result {
		m := r.Metric
		cluster := m["cluster"]
		if cluster == "" {
			cluster = "unknown"
		}
		fp := fingerprintFromMetric(m)
		starts := time.Now().UTC()
		if len(r.Value) >= 1 {
			if s, ok := r.Value[0].(string); ok {
				if sec, err := parseFloat(s); err == nil {
					starts = time.Unix(int64(sec), 0).UTC()
				}
			}
		}
		out = append(out, model.Alert{
			Fingerprint: fp,
			Alertname:   m["alertname"],
			Severity:    model.NormalizeSeverity(m["severity"]),
			Namespace:   m["namespace"],
			Cluster:     cluster,
			Summary:     m["summary"],
			Description: m["description"],
			RunbookURL:  m["runbook_url"],
			StartsAt:    starts,
		})
	}
	return out, nil
}

func fingerprintFromMetric(m map[string]string) string {
	parts := make([]string, 0, len(m))
	for k, v := range m {
		parts = append(parts, k+"="+v)
	}
	// stable enough for diffing within a process; Alertmanager FP preferred when available
	return strings.Join(parts, "|")
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// SplitByCluster turns a flat vmselect result into per-cluster fetches (used by orchestrator).
func SplitByCluster(alerts []model.Alert) map[string][]model.Alert {
	out := map[string][]model.Alert{}
	for _, a := range alerts {
		c := a.Cluster
		if c == "" {
			c = "unknown"
		}
		out[c] = append(out[c], a)
	}
	return out
}