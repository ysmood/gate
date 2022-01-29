package cert

import (
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/go-acme/lego/v4/providers/dns/dnspod"
	"github.com/go-acme/lego/v4/providers/dns/namedotcom"
)

func getProvider(provider, token string) (challenge.Provider, error) {
	switch provider {
	case "cloudflare":
		conf := cloudflare.NewDefaultConfig()
		conf.AuthToken = token
		return cloudflare.NewDNSProviderConfig(conf)
	case "dnspod":
		conf := dnspod.NewDefaultConfig()
		conf.LoginToken = token
		conf.HTTPClient.Timeout = 30 * time.Second
		return dnspod.NewDNSProviderConfig(conf)
	case "name":
		conf := namedotcom.NewDefaultConfig()
		conf.APIToken = token
		return namedotcom.NewDNSProviderConfig(conf)
	default:
		panic("provider not supported: " + provider)
	}
}
