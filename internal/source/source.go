package source

import (
	"context"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

type Batch struct {
	Alerts   []model.Alert
	Silenced []model.SilencedAlert
}

type Source interface {
	Cluster() string
	Fetch(ctx context.Context) (Batch, error)
}

// Silencer creates and expires silences on one alert source.
type Silencer interface {
	CreateSilence(ctx context.Context, draft model.SilenceDraft) (model.SilenceRef, error)
	ExpireSilence(ctx context.Context, id string) error
}
