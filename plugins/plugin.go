package plugins

import "time"

// PluginInterface provides an interface for a single data source (e.g. server) which should be regularly be tested
type PluginInterface interface {
	SetConfig(map[string]string) error        // SetConfig is used to set a config for this TestPlugin
	GetName() string                          // Return a name for the thing that is tested by this TestPlugin
	ExecuteTest() (DataPointInterface, error) // Method is regularly called to collect a new DataPoint
	New() PluginInterface                     // Return a new instance of this TestPlugin
	SetTimeout(time.Duration)                 // SetTimeout is used to set/update a timeout for this TestPlugin on the run
}

type PluginConfig map[string]string
