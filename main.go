package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/robfig/cron"
	"github.com/zenazn/goji/bind"
	"github.com/zenazn/goji/graceful"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
)

var _ = fmt.Printf

var (
	UrlsBucket = []byte("urls")
	LogsBucket = []byte("logs")

	log = logrus.New()
)

func ServeAsset(name, mime string) func(w http.ResponseWriter, r *http.Request) {
	// Assert that the asset exists.
	_, err := Asset(name)
	if err != nil {
		panic(fmt.Sprintf("asset named '%s' does not exist", name))
	}

	return func(w http.ResponseWriter, r *http.Request) {
		asset, _ := Asset(name)
		w.Header().Set("Content-Type", mime)
		w.Write(asset)
	}
}

func TryUpdate(db *bolt.DB, id uint64) {
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
	check.Update(db)
}

type ErrorsHook struct {
	DB *bolt.DB
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

		if err = b.Put(KeyFor(seq), data); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		// Note: shouldn't try to send another error+ message here, since we
		// might just recurse forever.
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

func main() {
	dbPath := "./monitor.db"
	db, err := bolt.Open(dbPath, 0666)
	if err != nil {
		log.WithFields(logrus.Fields{
			"path": dbPath,
			"err":  err,
		}).Fatal("error opening db")
	}
	defer db.Close()

	c := cron.New()

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
		DB: db,
	})

	// Initialize for each of the existing URLs
	var items []*Check
	if err = GetAllChecks(db, &items); err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Fatal("error loading checks")
	}

	for _, v := range items {
		// Trigger the update now...
		go v.Update(db)

		// ... and add a cron task for later.  Note that we pull out the ID
		// into a new variable so that we don't keep the entire Check structure
		// from being garbage collected.
		id := v.ID
		c.AddFunc(v.Schedule, func() {
			TryUpdate(db, id)
		})
	}

	// Start our cron scheduler.
	c.Start()
	defer c.Stop()

	mux := web.New()

	mux.Use(middleware.RequestID)
	mux.Use(LoggerMiddleware)
	mux.Use(RecovererMiddleware)
	mux.Use(middleware.AutomaticOptions)
	mux.Use(DbInjectMiddleware(db))
	mux.Use(CronInjectMiddleware(c))

	mux.Get("/", ServeAsset("index.html", "text/html"))

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
		mux.Get("/"+asset.Path, ServeAsset(asset.Path, asset.Mime))
	}

	api := web.New()
	api.Use(func(h http.Handler) http.Handler {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(handler)
	})
	api.Get("/api/checks", RouteChecksGetAll)
	api.Post("/api/checks", RouteChecksNew)
	api.Patch("/api/checks/:id", RouteChecksModify)
	api.Delete("/api/checks/:id", RouteChecksDelete)
	api.Post("/api/checks/:id/update", RouteChecksUpdateOne)
	api.Get("/api/stats", RouteStatsGetAll)
	api.Get("/api/logs", RouteLogsGetAll)
	api.Delete("/api/logs", RouteLogsDeleteAll)

	// Mount the API mux on the main one.
	mux.Handle("/api/*", api)

	// We re-create what Goji does to serve here.
	http.Handle("/", mux)
	listener := bind.Default()
	log.Println("starting server on", listener.Addr())
	bind.Ready()

	err = graceful.Serve(listener, http.DefaultServeMux)
	if err != nil {
		// TODO: what?
	}
	graceful.Wait()

	log.Info("Finished")
}
