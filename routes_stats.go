package main

import (
	"encoding/json"

	"github.com/boltdb/bolt"
	"github.com/gocraft/web"
)

type StatsContext struct {
	*ApiContext
}

func (ctx *StatsContext) GetAll(w web.ResponseWriter, r *web.Request) {
	context := map[string]interface{}{}

	ctx.db.View(func(tx *bolt.Tx) error {
		context["url-stats"] = tx.Bucket(UrlsBucket).Stats()
		context["log-count"] = tx.Bucket(LogsBucket).Stats().KeyN
		return nil
	})

	json.NewEncoder(w).Encode(context)
}

func RegisterStatsRoutes(router *web.Router) {
	statsRouter := router.Subrouter(StatsContext{}, "/stats")
	statsRouter.Get("", (*StatsContext).GetAll)
}
