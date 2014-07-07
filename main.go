package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/gocraft/web"
	"github.com/joeshaw/envdecode"
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
	manager *CheckManager
	updates chan interface{}
}

func (ctx *ApiContext) ContentTypeMiddleware(w web.ResponseWriter, r *web.Request, next web.NextMiddlewareFunc) {
	w.Header().Set("Content-Type", "application/json")
	next(w, r)
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

		if err = b.Put(KeyFor(logEntry.ID), data); err != nil {
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

	// This channel recieves the updates, which are eventually broadcasted
	// out to all connected websockets.
	updatesChan := make(chan interface{})

	// Create check manager.
	manager, err := NewCheckManager(db)
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Fatal("error creating check manager")
		return
	}
	defer manager.Close()

	// Hook up the updates from the manager to the updates channel.
	manager.OnUpdate = func(c *Check) {
		updatesChan <- map[string]interface{}{
			"type": "updated_check",
			"data": c,
		}
	}

	// Add a hook to our logger that will catch messages of severity Error
	// (and above) and will add them to our error log.
	log.Hooks.Add(&ErrorsHook{
		DB:      db,
		Updates: updatesChan,
	})

	router := web.New(GlobalContext{})
	router.
		Middleware((*GlobalContext).RequestIdMiddleware).
		Middleware((*GlobalContext).LogMiddleware).
		Middleware((*GlobalContext).RecoverMiddleware)

	router.Get("/", ServeAsset("index.html", "text/html"))
	for _, asset := range AssetDescriptors() {
		router.Get("/"+asset.Path, ServeAsset(asset.Path, asset.Mime))
	}

	apiRouter := router.Subrouter(ApiContext{}, "/api")
	apiRouter.
		Middleware((*ApiContext).ContentTypeMiddleware).
		Middleware(func(ctx *ApiContext, w web.ResponseWriter, r *web.Request, next web.NextMiddlewareFunc) {
		ctx.db = db
		ctx.manager = manager
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
