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

	// This function, if non-nil, is called whenever a site has
	// been updated.
	OnUpdate func(*Check)

	newCheck    chan newCheckMsg
	removeCheck chan uint64
	resp        chan error
	quit        chan struct{}
}

func KeyFor(id interface{}) (key []byte) {
	key = make([]byte, 8)

	switch v := id.(type) {
	case uint:
		binary.LittleEndian.PutUint64(key, uint64(v))
	case uint64:
		binary.LittleEndian.PutUint64(key, v)
	default:
		panic("unknown id type")
	}
	return
}

func NewCheckManager(db *bolt.DB) (ret *CheckManager, err error) {
	ret = &CheckManager{
		db:   db,
		cron: cron.New(),

		newCheck:    make(chan newCheckMsg),
		removeCheck: make(chan uint64),
		resp:        make(chan error),
		quit:        make(chan struct{}),
	}
	go ret.cronLoop()

	err = ret.setupInitialChecks()
	return
}

func (m *CheckManager) setupInitialChecks() error {
	checks, err := m.GetAllChecks()
	if err != nil {
		return err
	}

	for _, v := range checks {
		// Trigger the update now.
		// TODO: error handling here
		go v.Update()

		m.cron.AddFunc(v.Schedule, func() {
			m.RunCheck(v.ID)
		})
	}

	return nil
}

func (m *CheckManager) Close() {
	close(m.quit)
}

func (m *CheckManager) cronLoop() {
	// Stop the cron runner when this function exits.  Note that we need
	// to call a function, as opposed to just `defer m.cron.Stop()`,
	// because the value of `m.cron` might change.
	m.cron.Start()
	defer func() {
		m.cron.Stop()
	}()

	for {
		select {
		case msg := <-m.newCheck:
			// Note: we can't update the DB here, since the calling routine,
			// below, holds a writable transaction below.
			m.cron.AddFunc(msg.schedule, func() {
				m.RunCheck(msg.id)
			})
			m.resp <- nil

		case id := <-m.removeCheck:
			// Create a new cron instance and add all the checks to it.
			newc := cron.New()
			checks, err := m.GetAllChecks()
			if err != nil {
				m.resp <- err
				return
			}

			for _, c := range checks {
				// Don't add the one we're deleting
				if c.ID == id {
					continue
				}

				newc.AddFunc(c.Schedule, func() {
					m.RunCheck(c.ID)
				})
			}

			// Close / stop the old one and swap
			m.cron.Stop()
			m.cron = newc
			m.cron.Start()

			m.resp <- nil

		case <-m.quit:
			return
		}
	}
}

func (m *CheckManager) getCheckInternal(tx *bolt.Tx, id uint64, output *Check) error {
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
	output.ID = id

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

func (m *CheckManager) AddCheck(c *Check, update bool) (err error) {
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

	// Now, we need to add a new Cron task.
	m.newCheck <- newCheckMsg{c.ID, c.Schedule}
	<-m.resp // always nil

	// Update if we're asked to.  Ignoring errors, since they will be
	// written to the error log.
	if update {
		m.runInternal(c)
	}
	return nil
}

func (m *CheckManager) ModifyCheck(id uint64, fields map[string]interface{}) (ret *Check, changed bool, err error) {
	changed = false

	check := &Check{}
	err = m.db.Update(func(tx *bolt.Tx) error {
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

		// Don't bother with the save if we didn't change anything
		if !changed {
			return nil
		}

		return m.saveCheckInternal(tx, id, check)
	})

	if err != nil {
		ret = check
	}
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
	return <-m.resp
}

func (m *CheckManager) runInternal(check *Check) error {
	// Do the update.  This modifies the returned check in place.
	err := check.Update()
	if err != nil {
		return err
	}

	err = m.db.Update(func(tx *bolt.Tx) error {
		// We want to know if the check was just deleted or not
		tmp := &Check{}
		if err := m.getCheckInternal(tx, check.ID, tmp); err != nil {
			return fmt.Errorf("check was deleted mid-update: %d", check.ID)
		}

		// Save the real check back now
		return m.saveCheckInternal(tx, check.ID, check)
	})
	if err != nil {
		return err
	}

	// Callback that this check was updated.
	if m.OnUpdate != nil {
		m.OnUpdate(check)
	}

	return nil
}

func (m *CheckManager) RunCheck(id uint64) (*Check, error) {
	check := &Check{}
	err := m.db.View(func(tx *bolt.Tx) error {
		return m.getCheckInternal(tx, id, check)
	})
	if err != nil {
		return nil, err
	}

	err = m.runInternal(check)
	if err != nil {
		return nil, err
	}
	return check, nil
}
