package aggregate

import (
	"testing"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

func TestExpandByClusterLabel(t *testing.T) {
	t.Run("multiple sources unchanged", func(t *testing.T) {
		in := []ClusterFetch{{Name: "a"}, {Name: "b"}}
		if got := ExpandByClusterLabel(in); len(got) != 2 {
			t.Fatalf("got %d", len(got))
		}
	})

	t.Run("splits central AM", func(t *testing.T) {
		in := []ClusterFetch{{
			Name: "victoria-metrics",
			Alerts: []model.Alert{
				{Fingerprint: "1", Cluster: "demo214", Severity: model.BucketWarning},
				{Fingerprint: "2", Cluster: "demo215", Severity: model.BucketInfo},
			},
		}}
		got := ExpandByClusterLabel(in)
		if len(got) != 2 {
			t.Fatalf("got %d clusters", len(got))
		}
	})

	t.Run("single cluster label", func(t *testing.T) {
		in := []ClusterFetch{{
			Name: "victoria-metrics",
			Alerts: []model.Alert{{Fingerprint: "1", Cluster: "demo214"}},
		}}
		got := ExpandByClusterLabel(in)
		if len(got) != 1 || got[0].Name != "demo214" {
			t.Fatalf("got %+v", got)
		}
	})
}