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