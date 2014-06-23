package main

import (
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
)

type fieldEntry struct {
	Name  string
	Value interface{}
}

type ErrorLog struct {
	ID      uint64                 `json:"-"`
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Time    string                 `json:"time"`
	Fields  map[string]interface{} `json:"fields"`

	// Time in a prettier format
	PrettyTime string `json:"-"`

	// Fields in a format that can be interpreted by mustache
	PrettyFields []fieldEntry `json:"-"`
}

func (l *ErrorLog) PrepareForDisplay() {
	parsed, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST",
		l.Time)
	if err == nil {
		l.PrettyTime = parsed.Format("2006-01-02 15:04:05 MST")
	} else {
		log.Printf("could not parse time: %s", err)
		l.PrettyTime = l.Time
	}

	for k, v := range l.Fields {
		// If there's an "err" value that's an "error" instance, we turn it
		// into a string.
		if k == "err" {
			switch err := v.(type) {
			case error:
				l.PrettyFields = append(l.PrettyFields, fieldEntry{
					Name:  "err",
					Value: err.Error(),
				})
				continue
			}
		}

		l.PrettyFields = append(l.PrettyFields, fieldEntry{
			Name:  k,
			Value: v,
		})
	}
}
func GetAllLogs(db *bolt.DB, output *[]*ErrorLog) error {
	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(LogsBucket)
		b.ForEach(func(k, v []byte) error {
			entry := &ErrorLog{}
			if err := json.Unmarshal(v, entry); err != nil {
				log.WithFields(logrus.Fields{
					"err": err,
				}).Error("error unmarshaling json")
				return nil
			}

			entry.ID = binary.LittleEndian.Uint64(k)
			entry.PrepareForDisplay()

			*output = append(*output, entry)
			return nil
		})
		return nil
	})
}
