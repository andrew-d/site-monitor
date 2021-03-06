package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
)

// Helper struct for serialization.
type Check struct {
	ID          uint64    `json:"id"`
	URL         string    `json:"url"`
	Selector    string    `json:"selector"`
	Schedule    string    `json:"schedule"`
	LastChecked time.Time `json:"last_checked"`
	LastHash    string    `json:"last_hash"`
	SeenChange  bool      `json:"seen"`

	// The last-checked date, as a string.
	LastCheckedPretty string `json:"-"`

	// The first 8 characters of the hash
	ShortHash string `json:"-"`
}

func KeyFor(id interface{}) (key []byte) {
	key = make([]byte, 8)

	switch v := id.(type) {
	case int:
		binary.LittleEndian.PutUint64(key, uint64(v))
	case uint:
		binary.LittleEndian.PutUint64(key, uint64(v))
	case uint64:
		binary.LittleEndian.PutUint64(key, v)
	default:
		panic("unknown id type")
	}
	return
}

func (c *Check) PrepareForDisplay() {
	if c.LastChecked.IsZero() {
		c.LastCheckedPretty = "never"
	} else {
		c.LastCheckedPretty = c.LastChecked.Format(
			"Jan 2, 2006 at 3:04pm (MST)")
	}

	if len(c.LastHash) > 0 {
		c.ShortHash = c.LastHash[0:8]
	} else {
		c.ShortHash = "none"
	}
}

func GetAllChecks(db *bolt.DB, output *[]*Check) error {
	return db.View(func(tx *bolt.Tx) error {
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
			check.PrepareForDisplay()

			*output = append(*output, check)
			return nil
		})
		return nil
	})
}

func (c *Check) Update(db *bolt.DB) {
	log.WithFields(logrus.Fields{
		"id":  c.ID,
		"url": c.URL,
	}).Info("updating document")

	resp, err := http.Get(c.URL)
	if err != nil {
		log.WithFields(logrus.Fields{
			"id":  c.ID,
			"err": err.Error(),
			"url": c.URL,
		}).Error("error fetching check")
		return
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		log.WithFields(logrus.Fields{
			"id":  c.ID,
			"err": err,
		}).Error("error parsing check")
		return
	}

	// Get all nodes matching the given selector
	sel := doc.Find(c.Selector)
	if sel.Length() == 0 {
		log.WithFields(logrus.Fields{
			"id":       c.ID,
			"selector": c.Selector,
		}).Error("error in check: no nodes in selection")
		return
	}

	// Hash the content
	hash := sha256.New()
	io.WriteString(hash, sel.Text())
	sum := hex.EncodeToString(hash.Sum(nil))

	// Check for update
	if c.LastHash != sum {
		log.WithFields(logrus.Fields{
			"id":       c.ID,
			"lastHash": c.LastHash,
			"sum":      sum,
		}).Info("document changed")
		c.LastHash = sum
		c.SeenChange = false
	}

	c.LastChecked = time.Now()

	// Need to update the database now, since we've changed (at least the last
	// checked time).
	err = db.Update(func(tx *bolt.Tx) error {
		data, err := json.Marshal(c)
		if err != nil {
			return err
		}

		if err = tx.Bucket(UrlsBucket).Put(KeyFor(c.ID), data); err != nil {
			return err
		}
		return nil
	})
}
