package plugins

import "time"

type DataPointInterface interface {
	GetDelay() time.Duration
	GetResult() bool
}

// DataPoint represents a single data point
type DataPoint struct {
	delay  time.Duration
	result bool
}

// GetDelay returns the delay until the result was ready, return value undefined if result was false
func (p DataPoint) GetDelay() time.Duration {
	return p.delay
}

// GetResult returns the result of the test
func (p DataPoint) GetResult() bool {
	return p.result
}
