package plugins

import (
	"errors"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// DNSCollector represents a single DNS Server that should be checked
type DNSCollector struct {
	timeout       time.Duration
	dnsRecordType uint16
	ipAddress     string // IP Address of the DNS Resolver, IPv4 or IPv6
	port          string // Port of the DNS service (default 53)
	domain        string // domain that should be checked
}

func (d *DNSCollector) New() PluginInterface {
	return &DNSCollector{}
}

// SetConfig is used to set a config for this TestPlugin
// The config is a map of key/value pairs
func (d *DNSCollector) SetConfig(config map[string]string) error {
	// Parse IPAddress + Port of DNS Server
	if _, ok := config["IPAddress"]; !ok {
		return errors.New("missing IPAddress")
	}
	d.ipAddress = config["IPAddress"]

	// Parse Port of DNS Server
	if _, ok := config["Port"]; !ok {
		d.port = "53" // default port
	} else {
		d.port = config["Port"]
	}

	// Parse which Domain should be requested
	if _, ok := config["Domain"]; !ok {
		return errors.New("missing Domain")
	}
	d.domain = config["Domain"]

	// Parse which DNS Record Type should be requested
	if _, ok := config["RecordType"]; !ok {
		return errors.New("missing RecordType")
	}
	switch config["RecordType"] {
	case "A":
		d.dnsRecordType = dns.TypeA
	case "AAAA":
		d.dnsRecordType = dns.TypeAAAA
	case "CNAME":
		d.dnsRecordType = dns.TypeCNAME
	case "MX":
		d.dnsRecordType = dns.TypeMX
	case "NS":
		d.dnsRecordType = dns.TypeNS
	case "PTR":
		d.dnsRecordType = dns.TypePTR
	case "SOA":
		d.dnsRecordType = dns.TypeSOA
	case "TXT":
		d.dnsRecordType = dns.TypeTXT
	default:
		return errors.New("invalid RecordType")
	}

	// Parse Timeout
	if _, ok := config["Timeout"]; !ok {
		return errors.New("missing Timeout")
	}
	timeout, err := time.ParseDuration(config["Timeout"])
	if err != nil {
		return errors.New("invalid Timeout")
	}
	d.timeout = timeout

	return nil
}

func (d *DNSCollector) GetName() string {
	return d.ipAddress + ":" + d.port
}

func (d *DNSCollector) SetTimeout(timeout time.Duration) {
	d.timeout = timeout
}

func (d *DNSCollector) ExecuteTest() (DataPointInterface, error) {
	c := dns.Client{}
	c.Timeout = d.timeout

	// execute the DNS query
	m := dns.Msg{}
	m.SetQuestion(d.domain+".", d.dnsRecordType)
	r, delay, err := c.Exchange(&m, d.getIP()+":"+d.port)

	// error or empty answer
	if err != nil || len(r.Answer) == 0 {
		return &DataPoint{
			delay:  delay,
			result: false,
		}, nil
	}

	// Correct result
	return &DataPoint{
		delay:  delay,
		result: true,
	}, nil
}

// return the IP in a ready-to-use format for the DNS lib
func (d *DNSCollector) getIP() string {
	if strings.Contains(d.ipAddress, ":") {
		return "[" + d.ipAddress + "]"
	}
	return d.ipAddress
}
