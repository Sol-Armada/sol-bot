package sos

import "github.com/rs/xid"

func (t *Ticket) Save() error {
	if t.Id == "" {
		t.Id = xid.New().String()
	}

	return store.Upsert(t.Id, t)
}
