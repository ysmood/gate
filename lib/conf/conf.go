package conf

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

// Conf ...
type Conf struct {
	// Domains ...
	Domains []*Domain `json:"domains"`

	// TLSAddr to listen to, default is ":443"
	TLSAddr string `json:"tlsAddr"`

	// HTTPAddr to listen to, default is ":80". Set it to ":0" to disable it.
	// If it's not disabled the http requests to 80 will be redirect to https port.
	HTTPAddr string `json:"httpAddr"`
}

// Domain ...
type Domain struct {
	// Domain should be something like test.com, not abc.test.com
	Domain string `json:"domain"`

	// Mail for emergency notification such as when the cert is compromised
	Mail string `json:"mail"`

	// Provider is the dns provider's name, such as cloudflare, dnspod, etc
	Provider string `json:"provider"`

	// Token for the provider's api to modify the dns record to complete cert challenge
	// Such as https://developers.cloudflare.com/api/tokens/create
	Token string `json:"token"`

	// To target
	Routes []Route `json:"routes"`

	// CaDirURL of the cert issuer
	CaDirURL string `json:"caDirURL"`
}

// Route ...
type Route struct {
	// Priority of the route, the smaller the higher
	Priority int `json:"priority"`

	Token string `json:"token"`

	Selector Selector `json:"selector"`
}

// SelectorType ...
type SelectorType int

const (
	// SelectorTypeString ...
	SelectorTypeString SelectorType = iota
	// SelectorTypeRegexp ...
	SelectorTypeRegexp
)

// Selector for selecting the destination tcp endpoint.
// For example, for host "a.b.test.com" the "a.b" is the subdomain.
// If the Type is "regexp" and the Exp is "^a.b$", the selector will match the "a.b.test.com".
type Selector struct {
	// Type of the selector, available ones are "string", "regexp". Default is "string".
	Type SelectorType `json:"type"`

	// Exp is the expression of the selector
	Exp string `json:"exp"`

	reg *regexp.Regexp
}

// New ...
func New(path string) *Conf {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatalln(err)
	}

	var c Conf
	err = json.Unmarshal(b, &c)
	if err != nil {
		log.Fatalln(err)
	}

	if c.TLSAddr == "" {
		c.TLSAddr = ":443"
	}

	if c.HTTPAddr == "" {
		c.TLSAddr = ":80"
	}

	return &c
}

// Get ...
func (c *Conf) Get(domain string) (*Domain, bool) {
	for _, d := range c.Domains {
		if strings.HasSuffix(domain, d.Domain) {
			return d, true
		}
	}
	return nil, false
}

// Match ...
func (d *Domain) Match(domain string) bool {
	subdomain := strings.TrimRight(strings.TrimRight(domain, d.Domain), ".")

	for _, r := range d.Routes {
		switch r.Selector.Type {
		case SelectorTypeString:
			if r.Selector.Exp == subdomain {
				return true
			}
		case SelectorTypeRegexp:
			if r.Selector.reg == nil {
				r.Selector.reg = regexp.MustCompile(r.Selector.Exp)
			}
			if r.Selector.reg.MatchString(subdomain) {
				return true
			}
		}
	}
	return false
}

// Duration ...
type Duration struct {
	time.Duration
}

// NewDuration ...
func NewDuration(d time.Duration) Duration {
	return Duration{d}
}

// MarshalJSON ...
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration)
}

// UnmarshalJSON ...
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}
