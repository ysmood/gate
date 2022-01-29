package cert

import badger "github.com/dgraph-io/badger/v3"

// AutoRenewAll certs
// TODO: distribute the contention of renew requests.
func (m *Manager) AutoRenewAll() error {
	return m.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			id := it.Item().Key()
			cert, err := m.getCert(txn, id)
			if err != nil {
				return err
			}

			need, err := cert.NeedRenew()
			if err != nil {
				return err
			}

			if !need {
				continue
			}

			newCert, err := m.obtain(cert.Domain)
			if err != nil {
				return err
			}

			err = m.saveCert(txn, newCert)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
