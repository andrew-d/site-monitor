package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/Sirupsen/logrus"
	"github.com/gocraft/web"
)

var (
	prefix    string
	idCounter uint64
)

func init() {
	hostname, err := os.Hostname()
	if hostname == "" || err != nil {
		hostname = "localhost"
	}

	buff := [12]byte{}
	rnd := ""
	for len(rnd) < 10 {
		rand.Read(buff[:])
		rnd = base64.StdEncoding.EncodeToString(buff[:])
		rnd = strings.NewReplacer("+", "", "/", "").Replace(rnd)
	}

	prefix = fmt.Sprintf("%s/%s", hostname, rnd[0:10])
}

func (ctx *GlobalContext) LogMiddleware(w web.ResponseWriter, r *web.Request, next web.NextMiddlewareFunc) {
	next(w, r)
}

func (ctx *GlobalContext) RequestIdMiddleware(w web.ResponseWriter, r *web.Request, next web.NextMiddlewareFunc) {
	id := atomic.AddUint64(&idCounter, 1)
	ctx.RequestID = fmt.Sprintf("%s-%06d", prefix, id)
	next(w, r)
}

func (ctx *GlobalContext) RecoverMiddleware(w web.ResponseWriter, r *web.Request, next web.NextMiddlewareFunc) {
	defer func() {
		if err := recover(); err != nil {
			// Get a stacktrace.
			buf := make([]byte, 1<<16)
			amt := runtime.Stack(buf, false)
			stack := "\n" + string(buf[:amt])

			log.WithFields(logrus.Fields{
				"err":        err,
				"requestId":  ctx.RequestID,
				"stacktrace": stack,
			}).Error("recovered from panic")

			http.Error(w, http.StatusText(500), 500)
		}
	}()

	next(w, r)
}
