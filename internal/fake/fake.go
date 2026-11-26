package fake

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
	"github.com/vutratenko/shturval-alert-wall/internal/source"
)

// FileSource loads alerts from a JSON file (array of model.Alert) for local dev.
// Silences live in memory on this instance and survive polls.
type FileSource struct {
	ClusterName string
	Path        string

	mu   sync.Mutex
	next int
	byID map[string]storedSilence
}

type storedSilence struct {
	ref      model.SilenceRef
	matchers []model.Matcher
}

func (f *FileSource) Cluster() string { return f.ClusterName }

func (f *FileSource) Fetch(context.Context) (source.Batch, error) {
	b, err := os.ReadFile(f.Path)
	if err != nil {
		return source.Batch{}, err
	}
	var alerts []model.Alert
	if err := json.Unmarshal(b, &alerts); err != nil {
		return source.Batch{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now().UTC()
	for id, s := range f.byID {
		if !s.ref.EndsAt.After(now) {
			delete(f.byID, id)
		}
	}
	var batch source.Batch
	for _, alert := range alerts {
		var refs []model.SilenceRef
		for _, s := range f.byID {
			if labelsMatch(alert.Labels, s.matchers) {
				refs = append(refs, s.ref)
			}
		}
		if len(refs) == 0 {
			batch.Alerts = append(batch.Alerts, alert)
			continue
		}
		batch.Silenced = append(batch.Silenced, model.SilencedAlert{Alert: alert, Silences: refs})
	}
	return batch, nil
}

func (f *FileSource) CreateSilence(_ context.Context, draft model.SilenceDraft) (model.SilenceRef, error) {
	if len(draft.Matchers) == 0 {
		return model.SilenceRef{}, fmt.Errorf("invalid silence")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.byID == nil {
		f.byID = map[string]storedSilence{}
	}
	now := time.Now().UTC()
	for _, s := range f.byID {
		if s.ref.EndsAt.After(now) && model.SameMatchers(s.matchers, draft.Matchers) {
			return s.ref, nil
		}
	}
	f.next++
	ref := model.SilenceRef{
		ID:        fmt.Sprintf("fake-%d", f.next),
		EndsAt:    draft.EndsAt,
		CreatedBy: draft.CreatedBy,
		Comment:   draft.Comment,
		Matchers:  draft.Matchers,
	}
	f.byID[ref.ID] = storedSilence{ref: ref, matchers: append([]model.Matcher(nil), draft.Matchers...)}
	return ref, nil
}

func (f *FileSource) ExpireSilence(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.byID[id]; !ok {
		return fmt.Errorf("silence %s not found", id)
	}
	delete(f.byID, id)
	return nil
}

func labelsMatch(labels map[string]string, matchers []model.Matcher) bool {
	if len(matchers) == 0 {
		return false
	}
	for _, m := range matchers {
		if m.IsRegex || !m.IsEqual || labels[m.Name] != m.Value {
			return false
		}
	}
	return true
}
