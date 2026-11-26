package fake

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

func TestFileSource_SilenceMovesAlert(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alerts.json")
	raw := []byte(`[{"fingerprint":"fp1","alertname":"DiskFull","severity":"critical","labels":{"alertname":"DiskFull","namespace":"orders"}}]`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	src := &FileSource{ClusterName: "demo", Path: path}
	batch, err := src.Fetch(context.Background())
	if err != nil || len(batch.Alerts) != 1 || len(batch.Silenced) != 0 {
		t.Fatalf("%v %+v", err, batch)
	}
	matchers := model.MatchersFromLabels(batch.Alerts[0].Labels)
	ref, err := src.CreateSilence(context.Background(), model.SilenceDraft{
		Matchers:  matchers,
		EndsAt:    time.Now().UTC().Add(time.Hour),
		CreatedBy: "a@b.c",
		Comment:   "hold",
	})
	if err != nil || ref.ID == "" {
		t.Fatal(err)
	}
	again, err := src.CreateSilence(context.Background(), model.SilenceDraft{
		Matchers: matchers, EndsAt: time.Now().UTC().Add(time.Hour), Comment: "hold",
	})
	if err != nil || again.ID != ref.ID {
		t.Fatalf("reuse %v %+v", err, again)
	}
	batch, err = src.Fetch(context.Background())
	if err != nil || len(batch.Alerts) != 0 || len(batch.Silenced) != 1 {
		t.Fatalf("%v %+v", err, batch)
	}
	if batch.Silenced[0].Silences[0].ID != ref.ID {
		t.Fatalf("%+v", batch.Silenced[0].Silences)
	}
	if err := src.ExpireSilence(context.Background(), ref.ID); err != nil {
		t.Fatal(err)
	}
	batch, err = src.Fetch(context.Background())
	if err != nil || len(batch.Alerts) != 1 || len(batch.Silenced) != 0 {
		t.Fatalf("%v %+v", err, batch)
	}
}
