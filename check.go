package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/Sirupsen/logrus"
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
}

var updateCheckFunc = func(c *Check) error {
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
		return err
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		log.WithFields(logrus.Fields{
			"id":  c.ID,
			"err": err,
		}).Error("error parsing check")
		return err
	}

	// Get all nodes matching the given selector
	sel := doc.Find(c.Selector)
	if sel.Length() == 0 {
		log.WithFields(logrus.Fields{
			"id":       c.ID,
			"selector": c.Selector,
		}).Error("error in check: no nodes in selection")
		return err
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
	return nil
}

func (c *Check) Update() error {
	return updateCheckFunc(c)
}
