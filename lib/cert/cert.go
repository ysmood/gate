package cert

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/ysmood/gate/lib/conf"
	"golang.org/x/crypto/ocsp"
)

// Cert ...
type Cert struct {
	Certificate []byte
	PrivateKey  []byte
	Domain      *conf.Domain
}

// TLS ...
func (c *Cert) TLS() *tls.Certificate {
	tlsCert, err := tls.X509KeyPair(c.Certificate, c.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}
	return &tlsCert
}

// X509 ...
func (c *Cert) X509() []*x509.Certificate {
	certs, err := certcrypto.ParsePEMBundle(c.Certificate)
	if err != nil {
		log.Fatal(err)
	}
	if err != nil {
		log.Fatal(err)
	}
	return certs
}

// NeedRenew ...
func (c *Cert) NeedRenew() (bool, error) {
	if time.Until(c.X509()[0].NotAfter) < 10*24*time.Hour {
		return true, nil
	}

	err := c.Verify()
	if err != nil {
		return true, err
	}
	return false, nil
}

// Verify if the cert is valid on CA. Such as when a cert is compromised.
func (c *Cert) Verify() error {
	certs := c.X509()
	body, err := ocsp.CreateRequest(certs[0], certs[1], nil)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, certs[0].OCSPServer[0], bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	ocspRes, err := ocsp.ParseResponse(body, certs[1])
	if err != nil {
		return err
	}

	if ocspRes.Status != 0 {
		return fmt.Errorf("ocsp response status is %d", ocspRes.Status)
	}

	return nil
}
