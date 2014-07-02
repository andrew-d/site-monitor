package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/gocraft/web"
)

type ChecksContext struct {
	*ApiContext
	id uint64
}

func (ctx *ChecksContext) ParseIdMiddleware(w web.ResponseWriter, r *web.Request, next web.NextMiddlewareFunc) {
	var err error

	if id, ok := r.PathParams["id"]; ok {
		ctx.id, err = strconv.ParseUint(id, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	next(w, r)
}

func (ctx *ChecksContext) GetAll(w web.ResponseWriter, r *web.Request) {
	checks := []*Check{}
	err := GetAllChecks(ctx.db, &checks)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(checks)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (ctx *ChecksContext) Post(w web.ResponseWriter, r *web.Request) {
	params := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		http.Error(w, "bad input JSON", http.StatusBadRequest)
		return
	}

	for _, key := range []string{"url", "selector", "schedule"} {
		val, ok := params[key]
		if !ok || len(val) == 0 {
			http.Error(w, fmt.Sprintf("missing '%s' parameter", key), http.StatusBadRequest)
			return
		}
	}

	check := Check{
		URL:      params["url"],
		Selector: params["selector"],
		Schedule: params["schedule"],
	}

	err = ctx.db.Update(func(tx *bolt.Tx) error {
		data, err := json.Marshal(check)
		if err != nil {
			return err
		}

		b := tx.Bucket(UrlsBucket)

		seq, err := b.NextSequence()
		if err != nil {
			return err
		}
		check.ID = uint64(seq)

		if err = b.Put(KeyFor(seq), data); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.WithFields(logrus.Fields{
			"err":   err,
			"check": check,
		}).Error("error inserting new item")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If we succeeded, we update right now...
	check.Update(ctx.db, ctx.updates)

	// ... and add a new Cron callback
	ctx.cron.AddFunc(check.Schedule, func() {
		TryUpdate(ctx.db, check.ID, ctx.updates)
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(check)
}

func (ctx *ChecksContext) Patch(w web.ResponseWriter, r *web.Request) {
	bodyJson := map[string]interface{}{}

	err := json.NewDecoder(r.Body).Decode(&bodyJson)
	if err != nil {
		http.Error(w, "bad input JSON", http.StatusBadRequest)
		return
	}

	check := &Check{}
	err = GetOneCheck(ctx.db, ctx.id, check)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update each of the fields in the check
	updated := false
	if v, ok := bodyJson["url"].(string); ok {
		check.URL = v
		updated = true
	}
	if v, ok := bodyJson["selector"].(string); ok {
		check.Selector = v
		updated = true
	}
	if v, ok := bodyJson["schedule"].(string); ok {
		check.Schedule = v
		updated = true
	}
	if v, ok := bodyJson["seen"].(bool); ok {
		check.SeenChange = v
		updated = true
	}

	if !updated {
		log.WithFields(logrus.Fields{
			"body": bodyJson,
		}).Warn("no modifications given in PATCH request")
		return
	}

	err = ctx.db.Update(func(tx *bolt.Tx) error {
		data, err := json.Marshal(check)
		if err != nil {
			return err
		}

		if err = tx.Bucket(UrlsBucket).Put(KeyFor(check.ID), data); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: http status
	json.NewEncoder(w).Encode(check)
}

func (ctx *ChecksContext) UpdateOne(w web.ResponseWriter, r *web.Request) {
	check := &Check{}
	err := GetOneCheck(ctx.db, ctx.id, check)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	check.Update(ctx.db, ctx.updates)

	// TODO: http status
	json.NewEncoder(w).Encode(check)
}

func (ctx *ChecksContext) Delete(w web.ResponseWriter, r *web.Request) {
	err := ctx.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(UrlsBucket).Delete(KeyFor(ctx.id))
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Del("Content-Type")
	w.WriteHeader(http.StatusNoContent)
}

func RegisterCheckRoutes(router *web.Router) {
	checksRouter := router.Subrouter(ChecksContext{}, "/checks")
	checksRouter.Middleware((*ChecksContext).ParseIdMiddleware)

	checksRouter.Get("", (*ChecksContext).GetAll)
	checksRouter.Post("", (*ChecksContext).Post)
	checksRouter.Patch("/:id", (*ChecksContext).Patch)
	checksRouter.Delete("/:id", (*ChecksContext).Delete)
	checksRouter.Post("/:id/update", (*ChecksContext).UpdateOne)
}
