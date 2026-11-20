package mood

import "github.com/vutratenko/shturval-alert-wall/internal/model"

type Thresholds struct {
	MaxWarningForCalm int
	MinCriticalPanic  int
	MinWarningPanic   int
}

func DefaultThresholds() Thresholds {
	return Thresholds{
		MaxWarningForCalm: 2,
		MinCriticalPanic:  2,
		MinWarningPanic:   10,
	}
}

// Compute derives fox mood from aggregate counts and unreachable clusters.
func Compute(total model.Counts, unreachable int, th Thresholds) model.Mood {
	if th.MinCriticalPanic <= 0 {
		th = DefaultThresholds()
	}
	if unreachable > 0 {
		return model.MoodWorried
	}
	if total.Critical >= th.MinCriticalPanic || total.Warning >= th.MinWarningPanic {
		return model.MoodPanic
	}
	if total.Critical == 0 && total.Warning <= th.MaxWarningForCalm {
		return model.MoodCalm
	}
	return model.MoodWorried
}