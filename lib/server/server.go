package server

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"github.com/ysmood/gate/lib/cert"
	"github.com/ysmood/gate/lib/conf"
)

// Server ...
type Server struct {
	Logger *logrus.Logger

	conf *conf.Conf

	cert  *cert.Manager
	cache *gocache.Cache

	tlsListener  net.Listener
	httpListener net.Listener
}

// New ...
func New(c *conf.Conf) *Server {
	s := &Server{
		Logger: logrus.New(),
		conf:   c,
		cert:   cert.New(),
		cache:  gocache.New(5*time.Minute, 10*time.Minute),
	}

	s.cert.Start()
	s.preheatCerts()

	ln, err := net.Listen("tcp", c.TLSAddr)
	if err != nil {
		s.Logger.Fatal(err)
	}
	s.tlsListener = ln
	s.Logger.Println("listen tls on", c.TLSAddr)

	ln, err = net.Listen("tcp", c.HTTPAddr)
	if err != nil {
		s.Logger.Fatal(err)
	}
	s.httpListener = ln
	s.Logger.Println("listen http on", c.HTTPAddr)

	return s
}

// Serve ...
func (s *Server) Serve() {
	go s.handleHTTP()

	for {
		src, err := s.tlsListener.Accept()
		if err != nil {
			s.Logger.Trace(err)
			continue
		}
		go s.handleTLS(src)
	}
}

func (s *Server) handleHTTP() {
	addr, err := net.ResolveTCPAddr("tcp", s.conf.TLSAddr)
	if err != nil {
		s.Logger.Fatal(err)
	}

	err = http.Serve(s.httpListener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, port, _ := net.SplitHostPort(r.Host)
		target := fmt.Sprintf("https://%s%s", r.Host, r.RequestURI)
		if port != "" {
			target = fmt.Sprintf("https://%s:%d%s", host, addr.Port, r.RequestURI)
		}
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	}))

	if err != nil {
		s.Logger.Fatal(err)
	}
}

func (s *Server) handleTLS(conn net.Conn) {
	dstAddr := ""

	src := tls.Server(conn, &tls.Config{
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			if v, has := s.cache.Get(chi.ServerName); has {
				c := v.(*certCache)
				dstAddr = c.destination
				return c.cert, nil
			}

			d, has := s.conf.Get(chi.ServerName)
			if !has {
				return nil, fmt.Errorf("cert not found")
			}

			cert, err := s.cert.Get(d)
			if err != nil {
				return nil, err
			}

			dstAddr, has = d.MatchDestination(chi.ServerName)
			if !has {
				return nil, fmt.Errorf("not proxy destination found")
			}

			c := &certCache{cert: cert.TLS(), destination: dstAddr}
			s.cache.Set(chi.ServerName, c, 0)
			return c.cert, nil
		},
	})

	err := src.Handshake()
	if err != nil {
		s.Logger.Trace(err)
		return
	}

	dst, err := net.Dial("tcp", dstAddr)
	if err != nil {
		s.Logger.Trace(err)
		return
	}

	go func() {
		_, _ = io.Copy(dst, src)
	}()

	_, _ = io.Copy(src, dst)
}

func (s *Server) preheatCerts() {
	for _, d := range s.conf.Domains {
		_, err := s.cert.Get(d)
		if err != nil {
			s.Logger.Fatal(err)
		}
	}
}

type certCache struct {
	cert        *tls.Certificate
	destination string
}
