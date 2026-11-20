package discovery

import "testing"

func TestAlertmanagerServiceURL(t *testing.T) {
	got := alertmanagerServiceURL("vmalertmanager-demo", "prod-ns", 9093)
	want := "http://vmalertmanager-demo.prod-ns.svc:9093"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}