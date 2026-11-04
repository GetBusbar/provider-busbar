package main

import (
	"errors"
	"time"
)

func validateMetricInterval(interval time.Duration) error {
	if interval <= 0 {
		return errors.New("poll-state-metric must be greater than zero")
	}
	return nil
}
