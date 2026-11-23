package push

import (
	"fmt"

	"github.com/vutratenko/shturval-alert-wall/internal/aggregate"
)

func Message(events []aggregate.Event) (title, body string, ok bool) {
	var names []string
	var cluster string
	for _, ev := range events {
		if ev.Type != "alert.firing" || ev.Alert == nil || ev.Alert.Alertname == "" {
			continue
		}
		names = append(names, ev.Alert.Alertname)
		cluster = ev.Cluster
	}
	if len(names) == 0 {
		return "", "", false
	}
	if len(names) == 1 {
		body = names[0]
		if cluster != "" {
			body += " · " + cluster
		}
		return "Штурвал", body, true
	}
	return "Штурвал", fmt.Sprintf("%d новых алертов", len(names)), true
}
