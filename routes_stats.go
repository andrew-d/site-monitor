package main

import (
	"encoding/json"
	"net/http"

	"github.com/boltdb/bolt"
	"github.com/zenazn/goji/web"
)

func RouteStatsGetAll(c web.C, w http.ResponseWriter, r *http.Request) {
	context := map[string]interface{}{}
	db := c.Env["db"].(*bolt.DB)

	db.View(func(tx *bolt.Tx) error {
		context["url-stats"] = tx.Bucket(UrlsBucket).Stats()
		context["log-count"] = tx.Bucket(LogsBucket).Stats().KeyN
		return nil
	})

	json.NewEncoder(w).Encode(context)
}
