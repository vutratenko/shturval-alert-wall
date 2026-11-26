package model

import "testing"

func TestNormalizeSeverity(t *testing.T) {
	tests := []struct {
		in   string
		want Bucket
	}{
		{"Critical", BucketCritical},
		{"EMERGENCY", BucketCritical},
		{"warning", BucketWarning},
		{"Error", BucketWarning},
		{"info", BucketInfo},
		{"debug", BucketInfo},
		{"", BucketInfo},
	}
	for _, tt := range tests {
		if got := NormalizeSeverity(tt.in); got != tt.want {
			t.Fatalf("%q => %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMatchersFromLabels(t *testing.T) {
	got := MatchersFromLabels(map[string]string{
		"alertname": "DiskFull",
		"namespace": "orders",
		"":          "skip",
		"pod":       "  ",
	})
	want := []Matcher{
		{Name: "alertname", Value: "DiskFull", IsEqual: true},
		{Name: "namespace", Value: "orders", IsEqual: true},
	}
	if !SameMatchers(got, want) || !SameMatchers(want, got) {
		t.Fatalf("%+v", got)
	}
	if SameMatchers(nil, nil) || SameMatchers(got, []Matcher{want[0]}) {
		t.Fatal("empty or partial sets must not match")
	}
}
