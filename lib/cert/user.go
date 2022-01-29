package cert

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"

	"github.com/go-acme/lego/v4/acme/api"
	"github.com/go-acme/lego/v4/registration"
)

type user struct {
	mail string
	reg  *registration.Resource
	pkey crypto.PrivateKey
}

func newUser(mail, caURL string) (*user, error) {
	if mail == "" {
		mail = "admin@test.com"
	}

	u := &user{mail: mail}
	u.pkey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	core, err := api.New(http.DefaultClient, "gate", caURL, "", u.pkey)
	if err != nil {
		return nil, err
	}

	reg := registration.NewRegistrar(core, u)

	res, err := reg.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, err
	}
	u.reg = res

	return u, nil
}

func (u *user) GetEmail() string {
	return u.mail
}

func (u user) GetRegistration() *registration.Resource {
	return u.reg
}

func (u *user) GetPrivateKey() crypto.PrivateKey {
	return u.pkey
}
