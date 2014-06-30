package main

import (
	"encoding/json"
	"net/http"

	"github.com/boltdb/bolt"
	"github.com/gocraft/web"
)

type LogsContext struct {
	*ApiContext
}

func (ctx *LogsContext) GetAll(w web.ResponseWriter, r *web.Request) {
	items := []*ErrorLog{}
	GetAllLogs(ctx.db, &items)
	json.NewEncoder(w).Encode(items)
}

func (ctx *LogsContext) DeleteAll(w web.ResponseWriter, r *web.Request) {
	ctx.db.Update(func(tx *bolt.Tx) error {
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

func RegisterLogRoutes(router *web.Router) {
	logsRouter := router.Subrouter(LogsContext{}, "/logs")
	logsRouter.Get("", (*LogsContext).GetAll)
	logsRouter.Delete("", (*LogsContext).DeleteAll)
}
