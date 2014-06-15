package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/robfig/cron"
	"github.com/zenazn/goji/bind"
	"github.com/zenazn/goji/graceful"
	"github.com/zenazn/goji/param"
	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
)

var _ = fmt.Printf

// Helper struct for serialization.  Struct tags are used as follows:
//	- 'json':  For serializing to the DB.  A tag of "-" will not be stored.
//	- 'param': For retrieving from the user.  A tag of "-" will be ignored.
type Check struct {
	ID          uint64    `param:"-",json:"-"`
	URL         string    `param:"url",json:"url"`
	Selector    string    `param:"selector",json:"selector"`
	Schedule    string    `param:"schedule",json:"schedule"`
	LastChecked time.Time `param:"-",json:"last_checked"`
	LastHash    string    `param:"-",json:"last_hash"`
	SeenChange  bool      `param:"-",json:"seen"`

	// The last-checked date, as a string.
	LastCheckedPretty string `param:"-",json:"-"`

	// The first 8 characters of the hash
	ShortHash string `param:"-",json:"-"`
}

type fieldEntry struct {
	Name  string
	Value interface{}
}

type ErrorLog struct {
	ID      uint64                 `json:"-"`
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Time    string                 `json:"time"`
	Fields  map[string]interface{} `json:"fields"`

	// Time in a prettier format
	PrettyTime string `json:"-"`

	// Fields in a format that can be interpreted by mustache
	PrettyFields []fieldEntry `json:"-"`
}

var (
	UrlsBucket = []byte("urls")
	LogsBucket = []byte("logs")

	log = logrus.New()
)

type byStatus []*Check

func (s byStatus) Len() int      { return len(s) }
func (s byStatus) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byStatus) Less(i, j int) bool {
	if s[i].SeenChange {
		return false
	}
	if s[j].SeenChange {
		return true
	}

	return s[i].ID < s[j].ID
}

func KeyFor(id interface{}) (key []byte) {
	key = make([]byte, 8)

	switch v := id.(type) {
	case int:
		binary.LittleEndian.PutUint64(key, uint64(v))
	case uint:
		binary.LittleEndian.PutUint64(key, uint64(v))
	case uint64:
		binary.LittleEndian.PutUint64(key, v)
	default:
		panic("unknown id type")
	}
	return
}

func (c *Check) PrepareForDisplay() {
	if c.LastChecked.IsZero() {
		c.LastCheckedPretty = "never"
	} else {
		c.LastCheckedPretty = c.LastChecked.Format(
			"Jan 2, 2006 at 3:04pm (MST)")
	}

	if len(c.LastHash) > 0 {
		c.ShortHash = c.LastHash[0:8]
	} else {
		c.ShortHash = "none"
	}
}

func (l *ErrorLog) PrepareForDisplay() {
	parsed, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST",
		l.Time)
	if err == nil {
		l.PrettyTime = parsed.Format("2006-01-02 15:04:05 MST")
	} else {
		log.Printf("could not parse time: %s", err)
		l.PrettyTime = l.Time
	}

	for k, v := range l.Fields {
		// If there's an "err" value that's an "error" instance, we turn it
		// into a string.
		if k == "err" {
			switch err := v.(type) {
			case error:
				l.PrettyFields = append(l.PrettyFields, fieldEntry{
					Name:  "err",
					Value: err.Error(),
				})
				continue
			}
		}

		l.PrettyFields = append(l.PrettyFields, fieldEntry{
			Name:  k,
			Value: v,
		})
	}
}

func GetAllChecks(db *bolt.DB, output *[]*Check) error {
	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(UrlsBucket)
		b.ForEach(func(k, v []byte) error {
			check := &Check{}
			if err := json.Unmarshal(v, check); err != nil {
				log.WithFields(logrus.Fields{
					"err": err,
				}).Error("error unmarshaling json")
				return nil
			}

			check.ID = binary.LittleEndian.Uint64(k)
			check.PrepareForDisplay()

			*output = append(*output, check)
			return nil
		})
		return nil
	})
}

func GetAllLogs(db *bolt.DB, output *[]*ErrorLog) error {
	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(LogsBucket)
		b.ForEach(func(k, v []byte) error {
			entry := &ErrorLog{}
			if err := json.Unmarshal(v, entry); err != nil {
				log.WithFields(logrus.Fields{
					"err": err,
				}).Error("error unmarshaling json")
				return nil
			}

			entry.ID = binary.LittleEndian.Uint64(k)
			entry.PrepareForDisplay()

			*output = append(*output, entry)
			return nil
		})
		return nil
	})
}

func (c *Check) Update(db *bolt.DB) {
	log.WithFields(logrus.Fields{
		"id":  c.ID,
		"url": c.URL,
	}).Info("updating document")

	resp, err := http.Get(c.URL)
	if err != nil {
		log.WithFields(logrus.Fields{
			"id":  c.ID,
			"err": err.Error(),
			"url": c.URL,
		}).Error("error fetching check")
		return
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		log.WithFields(logrus.Fields{
			"id":  c.ID,
			"err": err,
		}).Error("error parsing check")
		return
	}

	// Get all nodes matching the given selector
	sel := doc.Find(c.Selector)
	if sel.Length() == 0 {
		log.WithFields(logrus.Fields{
			"id":       c.ID,
			"selector": c.Selector,
		}).Error("error in check: no nodes in selection")
		return
	}

	// Hash the content
	hash := sha256.New()
	io.WriteString(hash, sel.Text())
	sum := hex.EncodeToString(hash.Sum(nil))

	// Check for update
	if c.LastHash != sum {
		log.WithFields(logrus.Fields{
			"id":       c.ID,
			"lastHash": c.LastHash,
			"sum":      sum,
		}).Info("document changed")
		c.LastHash = sum
		c.SeenChange = false
	}

	c.LastChecked = time.Now()

	// Need to update the database now, since we've changed (at least the last
	// checked time).
	err = db.Update(func(tx *bolt.Tx) error {
		data, err := json.Marshal(c)
		if err != nil {
			return err
		}

		if err = tx.Bucket(UrlsBucket).Put(KeyFor(c.ID), data); err != nil {
			return err
		}
		return nil
	})
}

func IndexRoute(c web.C, w http.ResponseWriter, r *http.Request) {
	context := map[string]interface{}{}
	db := c.Env["db"].(*bolt.DB)

	// Fetch a list of all items.
	var items []*Check
	GetAllChecks(db, &items)

	// Sort the checks with our custom comparator.
	sort.Sort(byStatus(items))
	context["items"] = items

	// Show the number of unread logs.
	db.View(func(tx *bolt.Tx) error {
		context["log-count"] = tx.Bucket(LogsBucket).Stats().KeyN
		return nil
	})

	RenderTemplateTo(w, "index", context)
}

func NewCheckRoute(c web.C, w http.ResponseWriter, r *http.Request) {
	db := c.Env["db"].(*bolt.DB)

	var check Check
	r.ParseForm()
	err := param.Parse(r.Form, &check)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(check.URL) == 0 {
		http.Error(w, "missing URL parameter", http.StatusBadRequest)
		return
	}
	if len(check.Selector) == 0 {
		http.Error(w, "missing Selector parameter", http.StatusBadRequest)
		return
	}
	if len(check.Schedule) == 0 {
		http.Error(w, "missing Schedule parameter", http.StatusBadRequest)
		return
	}

	err = db.Update(func(tx *bolt.Tx) error {
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
	check.Update(db)

	// ... and add a new Cron callback
	cr := c.Env["cron"].(*cron.Cron)
	cr.AddFunc(check.Schedule, func() {
		TryUpdate(db, check.ID)
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func UpdateCheckRoute(c web.C, w http.ResponseWriter, r *http.Request) {
	db := c.Env["db"].(*bolt.DB)

	id, err := strconv.ParseUint(c.URLParams["id"], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	check := &Check{}
	err = db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket(UrlsBucket).Get(KeyFor(id))
		if data == nil {
			return fmt.Errorf("no such check: %d", id)
		}

		if err := json.Unmarshal(data, check); err != nil {
			log.WithFields(logrus.Fields{
				"err": err,
			}).Error("error unmarshaling json")
			return err
		}

		check.ID = id
		return nil
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	check.Update(db)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func DeleteCheckRoute(c web.C, w http.ResponseWriter, r *http.Request) {
	db := c.Env["db"].(*bolt.DB)

	id, err := strconv.ParseUint(c.URLParams["id"], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(UrlsBucket).Delete(KeyFor(id))
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func SeenCheckRoute(c web.C, w http.ResponseWriter, r *http.Request) {
	db := c.Env["db"].(*bolt.DB)

	id, err := strconv.ParseUint(c.URLParams["id"], 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = db.Update(func(tx *bolt.Tx) error {
		data := tx.Bucket(UrlsBucket).Get(KeyFor(id))
		if data == nil {
			return fmt.Errorf("no such check: %d", id)
		}

		check := &Check{}
		if err := json.Unmarshal(data, check); err != nil {
			log.WithFields(logrus.Fields{
				"err": err,
			}).Error("error unmarshaling json")
			return err
		}

		check.SeenChange = true

		data, err := json.Marshal(check)
		if err != nil {
			return err
		}

		if err = tx.Bucket(UrlsBucket).Put(KeyFor(id), data); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func StatsRoute(c web.C, w http.ResponseWriter, r *http.Request) {
	context := map[string]interface{}{}
	db := c.Env["db"].(*bolt.DB)

	db.View(func(tx *bolt.Tx) error {
		context["url-stats"] = tx.Bucket(UrlsBucket).Stats()
		return nil
	})

	// Show the number of unread logs.
	db.View(func(tx *bolt.Tx) error {
		context["log-count"] = tx.Bucket(LogsBucket).Stats().KeyN
		return nil
	})

	RenderTemplateTo(w, "stats", context)
}

func LogsRoute(c web.C, w http.ResponseWriter, r *http.Request) {
	context := map[string]interface{}{}
	db := c.Env["db"].(*bolt.DB)

	// Fetch a list of all items.
	var items []*ErrorLog
	GetAllLogs(db, &items)

	context["items"] = items
	context["log-count"] = len(items)
	RenderTemplateTo(w, "logs", context)
}

func LogsClearRoute(c web.C, w http.ResponseWriter, r *http.Request) {
	db := c.Env["db"].(*bolt.DB)

	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(LogsBucket)
		b.ForEach(func(k, v []byte) error {
			b.Delete(k)
			return nil
		})
		return nil
	})

	http.Redirect(w, r, "/logs", http.StatusSeeOther)
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
		Time:    entry.Data["time"].(string),
		Fields:  filteredFields,
	}

	err := hook.DB.Update(func(tx *bolt.Tx) error {
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

	mux.Get("/", IndexRoute)
	mux.Post("/addnew", NewCheckRoute)
	mux.Post("/update/:id", UpdateCheckRoute)
	mux.Post("/delete/:id", DeleteCheckRoute)
	mux.Post("/seen/:id", SeenCheckRoute)
	mux.Get("/stats", StatsRoute)
	mux.Get("/logs", LogsRoute)
	mux.Post("/logs/clear", LogsClearRoute)
	mux.Get("/static/*", http.StripPrefix("/static/",
		http.FileServer(http.Dir("./static"))))

	// We re-create what goji does to serve here.
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
