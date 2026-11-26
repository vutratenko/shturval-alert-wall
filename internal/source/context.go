package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

const (
	contextBefore = 15 * time.Minute
	contextAfter  = 5 * time.Minute
	contextStep   = time.Minute
)

type ContextSample struct {
	Name   string `json:"name"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

type Context struct {
	Available bool            `json:"available"`
	Samples   []ContextSample `json:"samples,omitempty"`
	Error     string          `json:"error,omitempty"`
}

type namedQuery struct {
	Name  string
	Query string
}

// ContextQueries returns the fixed metric queries for an alert. Label values are quoted.
// A cluster matcher is added only when the alert itself carries a cluster label.
func ContextQueries(alert model.Alert) []namedQuery {
	var out []namedQuery
	node := strings.TrimSpace(alert.Labels["node"])
	if node != "" {
		out = append(out, namedQuery{
			Name:  "нода Ready",
			Query: "kube_node_status_condition{" + selector(alert, "node", node, "condition", "Ready", "status", "true") + "}",
		})
	}
	ns := strings.TrimSpace(alert.Namespace)
	if ns == "" {
		ns = strings.TrimSpace(alert.Labels["namespace"])
	}
	pod := strings.TrimSpace(alert.Labels["pod"])
	if ns != "" && pod != "" {
		out = append(out, namedQuery{
			Name:  "рестарты пода",
			Query: "sum(kube_pod_container_status_restarts_total{" + selector(alert, "namespace", ns, "pod", pod) + "})",
		})
	}
	return out
}

func selector(alert model.Alert, pairs ...string) string {
	parts := make([]string, 0, len(pairs)/2+1)
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, pairs[i]+"="+promQuote(pairs[i+1]))
	}
	if cluster := strings.TrimSpace(alert.Labels["cluster"]); cluster != "" {
		parts = append(parts, "cluster="+promQuote(cluster))
	}
	return strings.Join(parts, ",")
}

func promQuote(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// QueryContext loads the first and last sample of each fixed query. An empty query list
// does not call VictoriaMetrics.
func QueryContext(ctx context.Context, selectURL string, client *http.Client, alert model.Alert, now time.Time) (Context, error) {
	queries := ContextQueries(alert)
	if len(queries) == 0 {
		return Context{Available: true}, nil
	}
	if client == nil {
		client = http.DefaultClient
	}
	start, end := contextWindow(alert.StartsAt, now)
	samples := make([]ContextSample, 0, len(queries))
	for _, query := range queries {
		points, err := queryRange(ctx, client, selectURL, query.Query, start, end)
		if err != nil {
			return Context{}, err
		}
		if len(points) == 0 {
			continue
		}
		sample := ContextSample{Name: query.Name, After: formatSample(points[len(points)-1])}
		if len(points) > 1 {
			sample.Before = formatSample(points[0])
		}
		samples = append(samples, sample)
	}
	return Context{Available: true, Samples: samples}, nil
}

func contextWindow(starts, now time.Time) (time.Time, time.Time) {
	if starts.IsZero() {
		return now.Add(-contextBefore), now
	}
	start := starts.Add(-contextBefore)
	end := starts.Add(contextAfter)
	if end.After(now) {
		end = now
	}
	if !end.After(start) {
		end = start.Add(contextStep)
	}
	return start, end
}

func queryRange(ctx context.Context, client *http.Client, selectURL, query string, start, end time.Time) ([]float64, error) {
	u, err := url.Parse(strings.TrimRight(selectURL, "/"))
	if err != nil {
		return nil, err
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1/query_range"
	q := u.Query()
	q.Set("query", query)
	q.Set("start", strconv.FormatInt(start.Unix(), 10))
	q.Set("end", strconv.FormatInt(end.Unix(), 10))
	q.Set("step", strconv.FormatInt(int64(contextStep.Seconds()), 10))
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
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vmselect: %s", resp.Status)
	}
	var raw struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Values [][]any `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if raw.Status != "success" {
		return nil, fmt.Errorf("vmselect: status %s", raw.Status)
	}
	if len(raw.Data.Result) == 0 || len(raw.Data.Result[0].Values) == 0 {
		return nil, nil
	}
	points := make([]float64, 0, len(raw.Data.Result[0].Values))
	for _, pair := range raw.Data.Result[0].Values {
		if len(pair) < 2 {
			continue
		}
		text, ok := pair[1].(string)
		if !ok {
			continue
		}
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			continue
		}
		points = append(points, value)
	}
	return points, nil
}

func formatSample(value float64) string {
	if value == 0 || value == 1 || value == -1 {
		return strconv.FormatInt(int64(value), 10)
	}
	rounded := math.Round(value*10) / 10
	return strconv.FormatFloat(rounded, 'f', 1, 64)
}
