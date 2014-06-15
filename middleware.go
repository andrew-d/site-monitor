package main

import (
	"net/http"
	"runtime"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/robfig/cron"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
)

func DbInjectMiddleware(db *bolt.DB) func(c *web.C, h http.Handler) http.Handler {
	middleware := func(c *web.C, h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			c.Env["db"] = db
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return middleware
}

func CronInjectMiddleware(cr *cron.Cron) func(c *web.C, h http.Handler) http.Handler {
	middleware := func(c *web.C, h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			c.Env["cron"] = cr
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return middleware
}

func LoggerMiddleware(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		reqId := middleware.GetReqID(*c)

		log.WithFields(logrus.Fields{
			"requestId":  reqId,
			"method":     r.Method,
			"url":        r.URL.String(),
			"remoteAddr": r.RemoteAddr,
		}).Info("Request start")

		t1 := time.Now()
		h.ServeHTTP(w, r)
		t2 := time.Now()

		log.WithFields(logrus.Fields{
			"reqId":       reqId,
			"elapsedTime": t2.Sub(t1),
		}).Info("Request finished")
	}
	return http.HandlerFunc(fn)
}

func RecovererMiddleware(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		reqId := middleware.GetReqID(*c)

		defer func() {
			if err := recover(); err != nil {
				// Get a stacktrace.
				buf := make([]byte, 1<<16)
				amt := runtime.Stack(buf, false)
				stack := "\n" + string(buf[:amt])

				log.WithFields(logrus.Fields{
					"err":        err,
					"requestId":  reqId,
					"stacktrace": stack,
				}).Error("recovered from panic")
				http.Error(w, http.StatusText(500), 500)
			}
		}()

		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
