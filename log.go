package main

import (
	"encoding/binary"
	"encoding/json"

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

			*output = append(*output, entry)
			return nil
		})
		return nil
	})
}
