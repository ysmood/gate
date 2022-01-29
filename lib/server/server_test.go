package server_test

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"

	"github.com/ysmood/gate/lib/conf"
	"github.com/ysmood/gate/lib/server"
	"github.com/ysmood/got"
)

type T struct {
	got.G
}

func Test(t *testing.T) {
	got.Each(t, T{})
}

func (t T) Basic() {
	key := t.Srand(8)
	mock := t.Serve().Route("/", "", key)
	cf := conf.New("../../config.json")
	cf.Domains[0].Routes[0].Selector = conf.Selector{Exp: "test"}
	cf.Domains[0].Routes[0].Destination = mock.HostURL.Host
	s := server.New(cf)

	go s.Serve()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1"+cf.HTTPAddr, nil)
	t.E(err)
	res, err := client.Do(req)
	t.E(err)
	t.Eq(res.Header.Get("Location"), "https://127.0.0.1"+cf.TLSAddr+"/")
	t.E(res.Body.Close())

	num := 30
	wg := sync.WaitGroup{}
	wg.Add(num)

	request := func() {
		hc := http.Client{Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				n, err := net.Dial("tcp", cf.TLSAddr)
				t.E(err)

				return tls.Client(n, &tls.Config{
					ServerName:         "test." + cf.Domains[0].Domain,
					InsecureSkipVerify: true,
				}), nil
			},
		}}

		req, _ := http.NewRequest("GET", mock.URL(), nil)

		res, err := hc.Do(req)
		t.E(err)

		b, err := io.ReadAll(res.Body)
		t.E(err)
		t.E(res.Body.Close())

		t.Eq(string(b), key)
		wg.Done()
	}

	for i := 0; i < num; i++ {
		go request()
	}

	wg.Wait()
}
