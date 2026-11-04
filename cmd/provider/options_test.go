package main

import (
	"testing"
	"time"
)

func TestMetricInterval(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Second} {
		if err := validateMetricInterval(interval); err == nil {
			t.Fatalf("accepted %s, which would panic in the state recorder ticker", interval)
		}
	}
	for _, interval := range []time.Duration{time.Nanosecond, 5 * time.Second} {
		if err := validateMetricInterval(interval); err != nil {
			t.Fatal(err)
		}
	}
}
