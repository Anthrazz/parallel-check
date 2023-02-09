package plugins

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/go-ping/ping"
)

// PingCollector represents a single Server that should be checked
type PingCollector struct {
	timeout         time.Duration
	address         string // Address of the DNS Resolver, IP or FQDN
	networkProtocol string // 'ip', 'ip4' or 'ip6'
}

func (p *PingCollector) New() PluginInterface {
	return &PingCollector{}
}

func (p *PingCollector) SetConfig(config map[string]string) error {
	// Parse IPAddress
	if _, ok := config["IPAddress"]; !ok {
		return errors.New("missing IPAddress")
	}
	p.address = config["IPAddress"]

	// Parse Timeout
	if _, ok := config["Timeout"]; !ok {
		return errors.New("missing Timeout")
	}
	timeout, err := time.ParseDuration(config["Timeout"])
	if err != nil {
		return errors.New("invalid Timeout")
	}
	p.timeout = timeout

	// Set Config to use IPv4 and/or IPv6
	p.networkProtocol = "ip"
	if v, ok := config["IPv4"]; ok && v == "true" {
		p.networkProtocol = "ip4"
	}
	if v, ok := config["IPv6"]; ok && v == "true" {
		p.networkProtocol = "ip6"
	}

	return nil
}

// GetName returns the name of the collector
func (p *PingCollector) GetName() string {
	return fmt.Sprintf("%s (%s)", p.address, p.networkProtocol)
}

// SetTimeout sets the timeout for the pinger
func (p *PingCollector) SetTimeout(timeout time.Duration) {
	p.timeout = timeout
}

// getNewPinger returns a new pinger instance
func (p *PingCollector) getNewPinger() *ping.Pinger {
	// create pinger manually to be able to set IPv4 or IPv6
	pinger := ping.New(p.address)
	pinger.SetNetwork(p.networkProtocol)
	err := pinger.Resolve()
	if err != nil {
		panic(err)
	}

	pinger.Count = 1
	pinger.Timeout = p.timeout

	// Under Windows only the Admin user is allowed to execute ping...
	if runtime.GOOS == "windows" {
		pinger.SetPrivileged(true)
	}

	return pinger
}

func (p *PingCollector) ExecuteTest() (DataPointInterface, error) {
	pinger := p.getNewPinger()
	pinger.Run() // Blocks until finished.

	stats := pinger.Statistics()

	// no answer
	if stats.PacketsRecv == 0 {
		return &DataPoint{
			delay:  0,
			result: false,
		}, nil
	} else {
		return &DataPoint{
			delay:  stats.AvgRtt,
			result: true,
		}, nil
	}
}
