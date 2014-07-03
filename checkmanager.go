package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/robfig/cron"
)

type newCheckMsg struct {
	id       uint64
	schedule string
}

type CheckManager struct {
	db   *bolt.DB
	cron *cron.Cron

	newCheck    chan newCheckMsg
	removeCheck chan uint64
	quit        chan struct{}
}

func NewCheckManager(db *bolt.DB, cron *cron.Cron) (ret *CheckManager) {
	ret = &CheckManager{
		db:   db,
		cron: cron,

		newCheck:    make(chan newCheckMsg),
		removeCheck: make(chan uint64),
		quit:        make(chan struct{}),
	}
	go ret.cronLoop()
	return
}

func (m *CheckManager) Close() {
	close(m.quit)
}

func (m *CheckManager) cronLoop() {
	for {
		select {
		case msg := <-m.newCheck:
			m.cron.AddFunc(msg.schedule, func() {
				m.RunCheck(msg.id)
			})

		case id := <-m.removeCheck:
			// TODO: implement
			var _ = id

		case _, ok := <-m.quit:
			if !ok {
				return
			}
		}
	}
}

func (m *CheckManager) getCheckInternal(tx *bolt.Tx, id uint64, output *Check) (err error) {
	data := tx.Bucket(UrlsBucket).Get(KeyFor(id))
	if data == nil {
		return fmt.Errorf("no such check: %d", id)
	}

	if err := json.Unmarshal(data, output); err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Error("error unmarshaling json")
		return err
	}

	return nil
}

func (m *CheckManager) saveCheckInternal(tx *bolt.Tx, id uint64, c *Check) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	if err = tx.Bucket(UrlsBucket).Put(KeyFor(c.ID), data); err != nil {
		return err
	}
	return nil
}

func (m *CheckManager) GetCheck(id uint64) (output *Check, err error) {
	err = m.db.View(func(tx *bolt.Tx) error {
		return m.getCheckInternal(tx, id, output)
	})
	return
}

func (m *CheckManager) GetAllChecks() (output []*Check, err error) {
	err = m.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(UrlsBucket)
		b.ForEach(func(k, v []byte) error {
			check := &Check{}
			if err := json.Unmarshal(v, check); err != nil {
				log.WithFields(logrus.Fields{
					"err": err,
				}).Error("error unmarshaling json")
				return nil
			}

			check.ID = binary.LittleEndian.Uint64(k)

			output = append(output, check)
			return nil
		})
		return nil
	})
	return
}

func (m *CheckManager) AddCheck(c *Check) (err error) {
	if len(c.URL) == 0 {
		return fmt.Errorf("no URL given")
	}

	if len(c.Selector) == 0 {
		return fmt.Errorf("no selector given")
	}

	if len(c.Schedule) == 0 {
		return fmt.Errorf("no schedule given")
	}

	err = m.db.Update(func(tx *bolt.Tx) error {
		seq, err := tx.Bucket(UrlsBucket).NextSequence()
		if err != nil {
			return err
		}
		c.ID = uint64(seq)

		return m.saveCheckInternal(tx, c.ID, c)
	})
	if err != nil {
		return
	}

	// Now, we need to add a new Cron task here
	m.newCheck <- newCheckMsg{c.ID, c.Schedule}
	return
}

func (m *CheckManager) ModifyCheck(id uint64, fields map[string]interface{}) (changed bool, err error) {
	changed = false

	err = m.db.Update(func(tx *bolt.Tx) error {
		check := &Check{}
		if err := m.getCheckInternal(tx, id, check); err != nil {
			return err
		}

		if v, ok := fields["url"].(string); ok {
			check.URL = v
			changed = true
		}
		if v, ok := fields["selector"].(string); ok {
			check.Selector = v
			changed = true
		}
		if v, ok := fields["schedule"].(string); ok {
			check.Schedule = v
			changed = true
		}
		if v, ok := fields["seen"].(bool); ok {
			check.SeenChange = v
			changed = true
		}

		return m.saveCheckInternal(tx, id, check)
	})
	return
}

func (m *CheckManager) DeleteCheck(id uint64) (err error) {
	err = m.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(UrlsBucket).Delete(KeyFor(id))
	})
	if err != nil {
		return
	}

	m.removeCheck <- id
	return
}

func (m *CheckManager) RunCheck(id uint64) (err error) {
	check := &Check{}
	err = m.db.View(func(tx *bolt.Tx) error {
		return m.getCheckInternal(tx, id, check)
	})
	if err != nil {
		return
	}

	// Do the update.
	check.Update(m.db, nil)

	// TODO: determine if it updated or not
	// TODO: modify Update() to just mutate self

	err = m.db.Update(func(tx *bolt.Tx) error {
		// We want to know if the check was just deleted or not
		tmp := &Check{}
		if err := m.getCheckInternal(tx, id, tmp); err != nil {
			return fmt.Errorf("check was deleted mid-update: %d", id)
		}

		// Save the real check back now
		return m.saveCheckInternal(tx, id, check)
	})
	return
}
