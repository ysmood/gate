package cert_test

import (
	"os"
	"testing"
	"time"

	"github.com/ysmood/gate/lib/cert"
	"github.com/ysmood/gate/lib/conf"
	"github.com/ysmood/got"
)

type T struct {
	got.G
}

func Test(t *testing.T) {
	got.Each(t, func(t *testing.T) T {
		return T{
			G: got.New(t),
		}
	})
}

func (t T) Basic() {
	m := cert.New()
	_ = os.RemoveAll(m.DBPath)
	m.Start()

	d := conf.New("../../config.json").Domains[0]

	// get a new one
	ts := time.Now()
	cert, err := m.Get(d)
	t.E(err)
	t.Gt(time.Since(ts), 3*time.Second)
	t.Eq(cert.X509()[0].DNSNames[1], d.Domain)

	need, err := cert.NeedRenew()
	t.E(err)
	t.False(need)

	// get from cache
	ts = time.Now()
	cert2, err := m.Get(d)
	t.Lt(time.Since(ts), time.Second)
	t.E(err)
	t.Eq(cert2.X509(), cert.X509())

	t.E(m.AutoRenewAll())
}
