package main

import (
	"encoding/json"
	"net/http"

	"github.com/boltdb/bolt"
	"github.com/zenazn/goji/web"
)

func RouteLogsGetAll(c web.C, w http.ResponseWriter, r *http.Request) {
	db := c.Env["db"].(*bolt.DB)

	// Fetch a list of all items.
	items := []*ErrorLog{}
	GetAllLogs(db, &items)

	json.NewEncoder(w).Encode(items)
}

func RouteLogsDeleteAll(c web.C, w http.ResponseWriter, r *http.Request) {
	db := c.Env["db"].(*bolt.DB)

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(LogsBucket)
		b.ForEach(func(k, v []byte) error {
			b.Delete(k)
			return nil
		})
		return nil
	})

	w.Header().Del("Content-Type")
	w.WriteHeader(http.StatusNoContent)
}
