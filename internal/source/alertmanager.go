package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

type amStatus struct {
	State       string   `json:"state"`
	SilencedBy  []string `json:"silencedBy"`
	InhibitedBy []string `json:"inhibitedBy"`
}

type amAlert struct {
	Fingerprint  string            `json:"fingerprint"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Status       amStatus          `json:"status"`
}

type amSilence struct {
	ID     string `json:"id"`
	Status struct {
		State string `json:"state"`
	} `json:"status"`
	Comment   string          `json:"comment"`
	CreatedBy string          `json:"createdBy"`
	StartsAt  time.Time       `json:"startsAt"`
	EndsAt    time.Time       `json:"endsAt"`
	Matchers  []model.Matcher `json:"matchers"`
}

func (a *Alertmanager) Fetch(ctx context.Context) (Batch, error) {
	q := url.Values{}
	q.Set("active", "true")
	q.Set("silenced", "true")
	q.Set("inhibited", "false")
	body, err := a.get(ctx, "/api/v2/alerts", q.Encode())
	if err != nil {
		return Batch{}, err
	}
	var raw []amAlert
	if err := json.Unmarshal(body, &raw); err != nil {
		return Batch{}, err
	}
	var batch Batch
	type pending struct {
		alert model.Alert
		ids   []string
	}
	var silenced []pending
	for _, r := range raw {
		if len(r.Status.InhibitedBy) > 0 {
			continue
		}
		alert := mapAMAlert(a.ClusterName, r)
		if len(r.Status.SilencedBy) > 0 {
			silenced = append(silenced, pending{alert: alert, ids: r.Status.SilencedBy})
			continue
		}
		batch.Alerts = append(batch.Alerts, alert)
	}
	if len(silenced) == 0 {
		return batch, nil
	}
	known, err := a.listSilences(ctx)
	if err != nil {
		return Batch{}, err
	}
	byID := map[string]model.SilenceRef{}
	for _, s := range known {
		byID[s.ID] = silenceRef(s)
	}
	batch.Silenced = make([]model.SilencedAlert, 0, len(silenced))
	for _, p := range silenced {
		refs := make([]model.SilenceRef, 0, len(p.ids))
		for _, id := range p.ids {
			if ref, ok := byID[id]; ok {
				refs = append(refs, ref)
				continue
			}
			refs = append(refs, model.SilenceRef{ID: id})
		}
		batch.Silenced = append(batch.Silenced, model.SilencedAlert{Alert: p.alert, Silences: refs})
	}
	return batch, nil
}

func (a *Alertmanager) CreateSilence(ctx context.Context, draft model.SilenceDraft) (model.SilenceRef, error) {
	if strings.TrimSpace(draft.Comment) == "" || len(draft.Matchers) == 0 {
		return model.SilenceRef{}, fmt.Errorf("alertmanager %s: invalid silence", a.ClusterName)
	}
	known, err := a.listSilences(ctx)
	if err != nil {
		return model.SilenceRef{}, err
	}
	now := time.Now().UTC()
	for _, s := range known {
		if !silenceActive(s, now) || !model.SameMatchers(s.Matchers, draft.Matchers) {
			continue
		}
		return silenceRef(s), nil
	}
	payload := struct {
		Matchers  []model.Matcher `json:"matchers"`
		StartsAt  time.Time       `json:"startsAt"`
		EndsAt    time.Time       `json:"endsAt"`
		CreatedBy string          `json:"createdBy"`
		Comment   string          `json:"comment"`
	}{draft.Matchers, draft.StartsAt, draft.EndsAt, draft.CreatedBy, draft.Comment}
	raw, err := json.Marshal(payload)
	if err != nil {
		return model.SilenceRef{}, err
	}
	body, err := a.send(ctx, http.MethodPost, "/api/v2/silences", "", raw)
	if err != nil {
		return model.SilenceRef{}, err
	}
	var created struct {
		SilenceID string `json:"silenceID"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		return model.SilenceRef{}, err
	}
	if created.SilenceID == "" {
		return model.SilenceRef{}, fmt.Errorf("alertmanager %s: empty silence id", a.ClusterName)
	}
	return model.SilenceRef{
		ID:        created.SilenceID,
		EndsAt:    draft.EndsAt,
		CreatedBy: draft.CreatedBy,
		Comment:   draft.Comment,
		Matchers:  draft.Matchers,
	}, nil
}

func (a *Alertmanager) ExpireSilence(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("alertmanager %s: empty silence id", a.ClusterName)
	}
	_, err := a.send(ctx, http.MethodDelete, "/api/v2/silence/"+url.PathEscape(id), "", nil)
	return err
}

func (a *Alertmanager) listSilences(ctx context.Context) ([]amSilence, error) {
	body, err := a.get(ctx, "/api/v2/silences", "")
	if err != nil {
		return nil, err
	}
	var raw []amSilence
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func silenceActive(s amSilence, now time.Time) bool {
	switch s.Status.State {
	case "active", "pending":
		return true
	case "expired":
		return false
	default:
		return s.EndsAt.After(now)
	}
}

func silenceRef(s amSilence) model.SilenceRef {
	return model.SilenceRef{
		ID:        s.ID,
		EndsAt:    s.EndsAt,
		CreatedBy: s.CreatedBy,
		Comment:   s.Comment,
		Matchers:  s.Matchers,
	}
}

func (a *Alertmanager) get(ctx context.Context, path, rawQuery string) ([]byte, error) {
	return a.send(ctx, http.MethodGet, path, rawQuery, nil)
}

func (a *Alertmanager) send(ctx context.Context, method, path, rawQuery string, payload []byte) ([]byte, error) {
	u, err := a.endpoint(path, rawQuery)
	if err != nil {
		return nil, err
	}
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("alertmanager %s: %s", a.ClusterName, resp.Status)
	}
	return respBody, nil
}

func (a *Alertmanager) endpoint(path, rawQuery string) (string, error) {
	u, err := url.Parse(strings.TrimRight(a.BaseURL, "/"))
	if err != nil {
		return "", err
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	u.RawQuery = rawQuery
	return u.String(), nil
}

func (a *Alertmanager) httpClient() *http.Client {
	if a.Client != nil {
		return a.Client
	}
	return http.DefaultClient
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
		Fingerprint:  r.Fingerprint,
		Alertname:    labels["alertname"],
		Severity:     model.NormalizeSeverity(labels["severity"]),
		Namespace:    labels["namespace"],
		Cluster:      cluster,
		Summary:      firstNonEmpty(ann["summary"], labels["summary"]),
		Description:  firstNonEmpty(ann["description"], labels["description"]),
		RunbookURL:   firstNonEmpty(ann["runbook_url"], labels["runbook_url"]),
		GeneratorURL: r.GeneratorURL,
		Labels:       copyStringMap(labels),
		StartsAt:     r.StartsAt,
	}
}

func copyStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
