package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Sirupsen/logrus"
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
	checks, err := ctx.manager.GetAllChecks()
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

	check := &Check{
		URL:      params["url"],
		Selector: params["selector"],
		Schedule: params["schedule"],
	}

	err = ctx.manager.AddCheck(check, true)
	if err != nil {
		log.WithFields(logrus.Fields{
			"err":   err,
			"check": check,
		}).Error("error inserting new item")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

	check, changed, err := ctx.manager.ModifyCheck(ctx.id, bodyJson)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !changed {
		log.WithFields(logrus.Fields{
			"body": bodyJson,
		}).Warn("no modifications given in PATCH request")
	}

	// TODO: http status
	json.NewEncoder(w).Encode(check)
}

func (ctx *ChecksContext) UpdateOne(w web.ResponseWriter, r *web.Request) {
	check, err := ctx.manager.RunCheck(ctx.id)
	if err != nil {
		// TODO: better error code
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: http status
	json.NewEncoder(w).Encode(check)
}

func (ctx *ChecksContext) Delete(w web.ResponseWriter, r *web.Request) {
	err := ctx.manager.DeleteCheck(ctx.id)
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
