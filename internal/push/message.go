package push

import (
	"fmt"

	"github.com/vutratenko/shturval-alert-wall/internal/aggregate"
)

func Message(events []aggregate.Event) (title, body string, ok bool) {
	opened := groupEvents(events, "group.opened")
	promoted := groupEvents(events, "group.promoted")
	if len(opened) == 1 {
		return "Штурвал", openedBody(opened[0]), true
	}
	if len(opened) > 1 {
		return "Штурвал", fmt.Sprintf("%d новых групп", len(opened)), true
	}
	if len(promoted) == 1 {
		return "Штурвал", promotedBody(promoted[0]), true
	}
	if len(promoted) > 1 {
		return "Штурвал", fmt.Sprintf("%d групп стали тяжелее", len(promoted)), true
	}
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

func groupEvents(events []aggregate.Event, kind string) []aggregate.Event {
	var out []aggregate.Event
	for _, ev := range events {
		if ev.Type != kind || ev.Group == nil || ev.Group.Root.Alertname == "" {
			continue
		}
		out = append(out, ev)
	}
	return out
}

func openedBody(ev aggregate.Event) string {
	name := ev.Group.Root.Alertname
	extra := len(ev.Group.Members) - 1
	if extra < 1 {
		return withCluster(name, ev.Cluster)
	}
	if ev.Cluster == "" {
		return fmt.Sprintf("%s · ещё %d", name, extra)
	}
	return fmt.Sprintf("%s · %s · ещё %d", name, ev.Cluster, extra)
}

func promotedBody(ev aggregate.Event) string {
	body := withCluster(ev.Group.Root.Alertname, ev.Cluster)
	if ev.Group.Severity != "" {
		body += " · " + string(ev.Group.Severity)
	}
	return body
}

func withCluster(name, cluster string) string {
	if cluster == "" {
		return name
	}
	return name + " · " + cluster
}
