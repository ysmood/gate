package cert

import (
	"bytes"
	"encoding/gob"
	"log"
	"sync"
	"time"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/ysmood/gate/lib/conf"
)

// Manager ...
type Manager struct {
	DBPath string
	// RenewalCheckInterval for renewal, default is a day
	RenewalCheckInterval time.Duration

	lock sync.Mutex
	db   *badger.DB
}

// New Manager with default options
func New() *Manager {
	return &Manager{
		DBPath:               "cert.db",
		RenewalCheckInterval: 24 * time.Hour,
	}
}

// Start ...
func (m *Manager) Start() {
	var err error
	m.db, err = badger.Open(badger.DefaultOptions(m.DBPath))
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			time.Sleep(m.RenewalCheckInterval)
			err := m.AutoRenewAll()
			if err != nil {
				log.Print(err)
			}
		}
	}()
}

// Get from cache or obtain a new one if necessary.
func (m *Manager) Get(d *conf.Domain) (*Cert, error) {
	txn := m.db.NewTransaction(false)
	cache, err := m.getCert(txn, m.domainID(d))
	txn.Discard()
	if err == nil {
		return cache, nil
	} else if err != badger.ErrKeyNotFound {
		return nil, err
	}

	cert, err := m.obtain(d)
	if err != nil {
		return nil, err
	}

	return cert, m.db.Update(func(txn *badger.Txn) error {
		return m.saveCert(txn, cert)
	})
}

func (m *Manager) getCert(txn *badger.Txn, id []byte) (*Cert, error) {
	var cache Cert
	item, err := txn.Get(id)
	if err != nil {
		return nil, err
	}

	err = item.Value(func(val []byte) error {
		buf := bytes.NewBuffer(val)
		return gob.NewDecoder(buf).Decode(&cache)
	})
	if err != nil {
		return nil, err
	}

	return &cache, err
}

func (m *Manager) saveCert(txn *badger.Txn, cert *Cert) error {
	buf := bytes.NewBuffer(nil)
	err := gob.NewEncoder(buf).Encode(cert)
	if err != nil {
		return err
	}

	item := badger.NewEntry(m.domainID(cert.Domain), buf.Bytes())
	item.ExpiresAt = uint64(cert.X509()[0].NotAfter.Unix())
	return txn.SetEntry(item)
}

func (m *Manager) domainID(d *conf.Domain) []byte {
	return []byte(d.Token)
}

func (m *Manager) obtain(d *conf.Domain) (*Cert, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	request := certificate.ObtainRequest{
		Domains: []string{d.Domain, "*." + d.Domain},
		Bundle:  true,
	}

	client, err := m.client(d)
	if err != nil {
		return nil, err
	}

	res, err := client.Certificate.Obtain(request)
	if err != nil {
		return nil, err
	}

	return &Cert{
		Certificate: res.Certificate,
		PrivateKey:  res.PrivateKey,
		Domain:      d,
	}, nil
}

func (m *Manager) client(d *conf.Domain) (*lego.Client, error) {
	u, err := newUser(d.Mail, d.CaDirURL)
	if err != nil {
		return nil, err
	}

	config := lego.NewConfig(u)

	if d.CaDirURL != "" {
		config.CADirURL = d.CaDirURL
	}
	config.Certificate.KeyType = certcrypto.RSA2048

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, err
	}

	provider, err := getProvider(d.Provider, d.Token)
	if err != nil {
		return nil, err
	}

	err = client.Challenge.SetDNS01Provider(provider)
	if err != nil {
		return nil, err
	}

	return client, nil
}
