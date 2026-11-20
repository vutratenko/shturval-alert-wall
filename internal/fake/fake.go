package fake

import (
	"context"
	"encoding/json"
	"os"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

// FileSource loads alerts from a JSON file (array of model.Alert) for local dev.
type FileSource struct {
	ClusterName string
	Path        string
}

func (f *FileSource) Cluster() string { return f.ClusterName }

func (f *FileSource) Fetch(context.Context) ([]model.Alert, error) {
	b, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, err
	}
	var alerts []model.Alert
	if err := json.Unmarshal(b, &alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}