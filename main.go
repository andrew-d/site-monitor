package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/gocraft/web"
	"github.com/joeshaw/envdecode"
	"github.com/robfig/cron"
	"github.com/stretchr/graceful"
)

var (
	UrlsBucket = []byte("urls")
	LogsBucket = []byte("logs")

	log = logrus.New()
)

type GlobalContext struct {
	RequestID string
}

func ServeAsset(name, mime string) func(w web.ResponseWriter, r *web.Request) {
	// Assert that the asset exists.
	_, err := Asset(name)
	if err != nil {
		panic(fmt.Sprintf("asset named '%s' does not exist", name))
	}

	return func(w web.ResponseWriter, r *web.Request) {
		asset, _ := Asset(name)
		w.Header().Set("Content-Type", mime)
		w.Write(asset)
	}
}

type ApiContext struct {
	*GlobalContext

	db      *bolt.DB
	cron    *cron.Cron
	updates chan interface{}
}

func (ctx *ApiContext) ContentTypeMiddleware(w web.ResponseWriter, r *web.Request, next web.NextMiddlewareFunc) {
	w.Header().Set("Content-Type", "application/json")
	next(w, r)
}

func TryUpdate(db *bolt.DB, id uint64, updates chan interface{}) {
	// The task may have been deleted from the DB, so we try to fetch it first
	check := &Check{}
	found := false

	err := db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket(UrlsBucket).Get(KeyFor(id))
		if data == nil {
			return nil
		}

		if err := json.Unmarshal(data, check); err != nil {
			log.WithFields(logrus.Fields{
				"err": err,
			}).Error("error unmarshaling json")
			return err
		}

		found = true
		return nil
	})

	if err != nil {
		// TODO: log something
		return
	}

	if !found {
		log.WithFields(logrus.Fields{
			"id": id,
		}).Info("skipping update for deleted check")
		return
	}

	// Got a check.  Trigger an update.
	check.Update(db, updates)
}

type ErrorsHook struct {
	DB      *bolt.DB
	Updates chan interface{}
}

func (hook *ErrorsHook) Fire(entry *logrus.Entry) error {
	filteredFields := make(map[string]interface{})
	for k, v := range entry.Data {
		if k != "level" && k != "msg" && k != "time" {
			filteredFields[k] = v
		}
	}

	logEntry := ErrorLog{
		Level:   entry.Data["level"].(string),
		Message: entry.Data["msg"].(string),
		Fields:  filteredFields,
	}

	// Parse the time we're given and reformat it
	tm := entry.Data["time"].(string)
	ptime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", tm)
	if err != nil {
		logEntry.Time = tm
	} else {
		logEntry.Time = ptime.Format(time.RFC3339)
	}

	err = hook.DB.Update(func(tx *bolt.Tx) error {
		data, err := json.Marshal(logEntry)
		if err != nil {
			return err
		}

		b := tx.Bucket(LogsBucket)
		seq, err := b.NextSequence()
		if err != nil {
			return err
		}
		logEntry.ID = uint64(seq)

		if err = b.Put(KeyFor(seq), data); err != nil {
			return err
		}
		return nil
	})

	hook.Updates <- map[string]interface{}{
		"type": "new_log",
		"data": logEntry,
	}

	if err != nil {
		// Note: shouldn't try to send another message with severity error+ here,
		// since we might just recurse forever.
		log.WithFields(logrus.Fields{
			"err": err,
		}).Warn("failed to log error to db")
	}
	return nil
}

func (hook *ErrorsHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.Error,
		logrus.Fatal,
		logrus.Panic,
	}
}

type Config struct {
	Hostname string `env:"HOST,default=localhost"`
	Port     string `env:"PORT,default=8000"`
	DB       string `env:"DATABASE,default=./monitor.db"`
}

func main() {
	config := Config{}
	err := envdecode.Decode(&config)
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Fatal("configuration error")
	}

	db, err := bolt.Open(config.DB, 0666)
	if err != nil {
		log.WithFields(logrus.Fields{
			"path": config.DB,
			"err":  err,
		}).Fatal("error opening db")
	}
	defer db.Close()

	c := cron.New()
	updatesChan := make(chan interface{})

	// Create collections.
	buckets := [][]byte{UrlsBucket, LogsBucket}
	db.Update(func(tx *bolt.Tx) error {
		for _, v := range buckets {
			b := tx.Bucket(v)
			if b == nil {
				tx.CreateBucket(v)
			}
		}
		return nil
	})

	// Add a hook to our logger that will catch errors (and above) and will add
	// them to our error log.
	log.Hooks.Add(&ErrorsHook{
		DB:      db,
		Updates: updatesChan,
	})

	// Initialize for each of the existing URLs
	var items []*Check
	if err = GetAllChecks(db, &items); err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Fatal("error loading checks")
	}

	for _, v := range items {
		// Trigger the update now (nil channel since we won't have any listening
		// clients at this point).
		go v.Update(db, nil)

		c.AddFunc(v.Schedule, func() {
			TryUpdate(db, v.ID, updatesChan)
		})
	}

	// Start our cron scheduler.
	c.Start()
	defer c.Stop()

	router := web.New(GlobalContext{})
	router.
		Middleware((*GlobalContext).RequestIdMiddleware).
		Middleware((*GlobalContext).LogMiddleware).
		Middleware((*GlobalContext).RecoverMiddleware)

	router.Get("/", ServeAsset("index.html", "text/html"))

	// TODO: serve map file in debug mode
	assets := []struct {
		Path string
		Mime string
	}{
		{"index.html", "text/html"},
		{"js/bundle.js", "text/javascript"},
		{"js/lib/bootstrap.min.js", "text/javascript"},
		{"css/bootstrap.min.css", "text/css"},
		{"fonts/glyphicons-halflings-regular.woff", "application/font-woff"},
		{"fonts/glyphicons-halflings-regular.ttf", "application/x-font-ttf"},
	}
	for _, asset := range assets {
		router.Get("/"+asset.Path, ServeAsset(asset.Path, asset.Mime))
	}

	apiRouter := router.Subrouter(ApiContext{}, "/api")
	apiRouter.
		Middleware((*ApiContext).ContentTypeMiddleware).
		Middleware(func(ctx *ApiContext, w web.ResponseWriter, r *web.Request, next web.NextMiddlewareFunc) {
		ctx.db = db
		ctx.cron = c
		ctx.updates = updatesChan
		next(w, r)
	})

	RegisterCheckRoutes(apiRouter)
	RegisterLogRoutes(apiRouter)
	RegisterStatsRoutes(apiRouter)
	RegisterWebsockets(router, updatesChan)

	addr := fmt.Sprintf("%s:%s", config.Hostname, config.Port)
	log.Printf("Starting server on %s", addr)
	graceful.Run(addr, 10*time.Second, router)
	log.Info("Finished")
}
