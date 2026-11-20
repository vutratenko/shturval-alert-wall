package source

import (
	"context"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

type Source interface {
	Cluster() string
	Fetch(ctx context.Context) ([]model.Alert, error)
}