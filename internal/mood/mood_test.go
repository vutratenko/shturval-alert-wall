package mood

import (
	"testing"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

func TestCompute(t *testing.T) {
	th := DefaultThresholds()
	cases := []struct {
		name string
		c    model.Counts
		down int
		want model.Mood
	}{
		{"calm", model.Counts{}, 0, model.MoodCalm},
		{"calm_warnings", model.Counts{Warning: 2}, 0, model.MoodCalm},
		{"worried_one_critical", model.Counts{Critical: 1}, 0, model.MoodWorried},
		{"worried_three_warnings", model.Counts{Warning: 3}, 0, model.MoodWorried},
		{"panic_criticals", model.Counts{Critical: 2}, 0, model.MoodPanic},
		{"panic_warnings", model.Counts{Warning: 10}, 0, model.MoodPanic},
		{"unreachable", model.Counts{}, 1, model.MoodWorried},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Compute(tc.c, tc.down, th); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}