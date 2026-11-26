package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/auth"
	"github.com/vutratenko/shturval-alert-wall/internal/model"
	"github.com/vutratenko/shturval-alert-wall/internal/source"
)

func (s *Server) createSilence(w http.ResponseWriter, r *http.Request) {
	if !s.Silence {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<16)
	var body struct {
		Source      string `json:"source"`
		Fingerprint string `json:"fingerprint"`
		Duration    string `json:"duration"`
		Comment     string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	comment := strings.TrimSpace(body.Comment)
	dur, ok := silenceDuration(strings.TrimSpace(body.Duration))
	sourceName := strings.TrimSpace(body.Source)
	fingerprint := strings.TrimSpace(body.Fingerprint)
	if !ok || sourceName == "" || fingerprint == "" || comment == "" || len(comment) > 1024 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	alert, found := findAlert(s.Hub.Snapshot(), sourceName, fingerprint)
	if !found {
		http.NotFound(w, r)
		return
	}
	matchers := model.MatchersFromLabels(alert.Labels)
	if len(matchers) == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	silencer, found := s.silencer(sourceName)
	if !found {
		http.NotFound(w, r)
		return
	}
	now := time.Now().UTC()
	ref, err := silencer.CreateSilence(r.Context(), model.SilenceDraft{
		Matchers:  matchers,
		StartsAt:  now,
		EndsAt:    now.Add(dur),
		CreatedBy: auth.CreatedBy(r.Context()),
		Comment:   comment,
	})
	if err != nil {
		http.Error(w, "alertmanager error", http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": ref.ID, "endsAt": ref.EndsAt})
}

func (s *Server) expireSilence(w http.ResponseWriter, r *http.Request) {
	if !s.Silence {
		http.NotFound(w, r)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	sourceName := strings.TrimSpace(r.URL.Query().Get("source"))
	if id == "" || sourceName == "" || !silenceVisible(s.Hub.Snapshot(), sourceName, id) {
		http.NotFound(w, r)
		return
	}
	silencer, found := s.silencer(sourceName)
	if !found {
		http.NotFound(w, r)
		return
	}
	if err := silencer.ExpireSilence(r.Context(), id); err != nil {
		http.Error(w, "alertmanager error", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func silenceDuration(raw string) (time.Duration, bool) {
	switch raw {
	case "1h", "4h", "24h":
		d, err := time.ParseDuration(raw)
		return d, err == nil
	default:
		return 0, false
	}
}

func (s *Server) silencer(name string) (source.Silencer, bool) {
	if s.Sources == nil {
		return nil, false
	}
	for _, src := range s.Sources() {
		if src.Cluster() != name {
			continue
		}
		silencer, ok := src.(source.Silencer)
		return silencer, ok
	}
	return nil, false
}

func findAlert(snap model.Snapshot, sourceName, fp string) (model.Alert, bool) {
	for _, c := range snap.Clusters {
		if !ownedBy(c, sourceName) {
			continue
		}
		for _, a := range c.Alerts {
			if a.Fingerprint == fp {
				return a, true
			}
		}
		for _, a := range c.Silenced {
			if a.Fingerprint == fp {
				return a.Alert, true
			}
		}
	}
	return model.Alert{}, false
}

func silenceVisible(snap model.Snapshot, sourceName, id string) bool {
	for _, c := range snap.Clusters {
		if !ownedBy(c, sourceName) {
			continue
		}
		for _, a := range c.Silenced {
			for _, s := range a.Silences {
				if s.ID == id {
					return true
				}
			}
		}
	}
	return false
}

func ownedBy(c model.ClusterSummary, sourceName string) bool {
	if c.Source != "" {
		return c.Source == sourceName
	}
	return c.Name == sourceName
}
