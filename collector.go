package main

import (
	"github.com/Anthrazz/parallel-check/plugins"
	"time"
)

// Collector provides an interface for a single data source (e.g. server) which should be regularly be tested
type Collector interface {
	SetConfig(map[string]string) error                // SetConfig is used to set a config for this Collector
	GetName() string                                  // Return a name for the thing that is tested by this Collector
	ExecuteTest() (plugins.DataPointInterface, error) // Method is regularly called to collect a new DataPoint
	New() Collector                                   // Return a new instance of this Collector
	SetTimeout(time.Duration)                         // SetTimeout is used to set/update a timeout for this Collector on the run
}
