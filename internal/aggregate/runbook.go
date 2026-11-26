package aggregate

import (
	"net/url"
	"strings"

	"github.com/vutratenko/shturval-alert-wall/internal/model"
)

func applyRunbooks(base string, alerts []model.Alert, silenced []model.SilencedAlert) {
	for i := range alerts {
		alerts[i].RunbookURL = runbookLink(alerts[i].RunbookURL, base, alerts[i].Alertname)
	}
	for i := range silenced {
		silenced[i].RunbookURL = runbookLink(silenced[i].RunbookURL, base, silenced[i].Alertname)
	}
}

func runbookLink(explicit, base, alertname string) string {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit)
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	name := strings.TrimSpace(alertname)
	if base == "" || name == "" {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	joined, err := url.JoinPath(base, name)
	if err != nil {
		return ""
	}
	return joined
}
