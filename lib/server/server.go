package server

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/ysmood/gate/lib/cert"
	"github.com/ysmood/gate/lib/conf"
)

// Server ...
type Server struct {
	Logger *logrus.Logger

	conf *conf.Conf

	certManager *cert.Manager

	tlsListener  net.Listener
	httpListener net.Listener

	clients sync.Map
}

// New ...
func New(c *conf.Conf) *Server {
	s := &Server{
		Logger:      logrus.New(),
		conf:        c,
		certManager: cert.New(),
		clients:     sync.Map{},
	}

	s.certManager.Start()
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
			return s.route(chi)
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

func (s *Server) route(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	d, has := s.conf.Get(chi.ServerName)
	if !has {
		return nil, fmt.Errorf("cert not found")
	}

	cert, err := s.certManager.Get(d)
	if err != nil {
		return nil, err
	}

	has = d.Match(chi.ServerName)
	if !has {
		return nil, fmt.Errorf("not proxy destination found")
	}

	return cert.TLS(), nil
}

func (s *Server) preheatCerts() {
	for _, d := range s.conf.Domains {
		_, err := s.certManager.Get(d)
		if err != nil {
			s.Logger.Fatal(err)
		}
	}
	s.certManager.AutoRenewAll()
}
