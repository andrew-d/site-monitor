package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/boltdb/bolt"
	"github.com/robfig/cron"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/param"
	"github.com/zenazn/goji/web"
)

var _ = fmt.Printf

// Helper struct for serialization.  Struct tags are used as follows:
//	- 'json' tags are for serializing to the DB - a tag of "-" will not be stored
//	- 'param' tags are for retrieving from the user - a tag of "-" will be ignored
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

var UrlsBucket = []byte("urls")

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
		c.LastCheckedPretty = c.LastChecked.Format("Jan 2, 2006 at 3:04pm (MST)")
	}

	if len(c.LastHash) > 0 {
		c.ShortHash = c.LastHash[0:8]
	} else {
		c.ShortHash = "<none>"
	}
}

func GetAllChecks(db *bolt.DB, output *[]*Check) error {
	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(UrlsBucket)
		b.ForEach(func(k, v []byte) error {
			check := &Check{}
			if err := json.Unmarshal(v, check); err != nil {
				log.Printf("error unmarshaling json: %s", err)
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

func (c *Check) Update(db *bolt.DB) {
	log.Printf("updating document with url: %s", c.URL)

	resp, err := http.Get(c.URL)
	if err != nil {
		log.Printf("error fetching check %d: %s", c.ID, err)
		return
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		log.Printf("error parsing check %d: %s", c.ID, err)
		return
	}

	// Get all nodes matching the given selector
	sel := doc.Find(c.Selector)
	if sel.Length() == 0 {
		log.Printf("error checking %d: no nodes in selection", c.ID)
		return
	}

	// Hash the content
	hash := sha256.New()
	io.WriteString(hash, sel.Text())
	sum := hex.EncodeToString(hash.Sum(nil))

	// Check for update
	if c.LastHash != sum {
		log.Printf("document %d changed: %s --> %s", c.ID, c.LastHash, sum)
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
		log.Printf("error inserting item: %s", err)
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
			log.Printf("error unmarshaling json: %s", err)
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
			log.Printf("error unmarshaling json: %s", err)
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

	RenderTemplateTo(w, "stats", context)
}

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
			log.Printf("error unmarshaling json: %s", err)
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
		log.Printf("skipping update for deleted check: %d", id)
		return
	}

	// Got a check.  Trigger an update.
	check.Update(db)
}

func main() {
	db, err := bolt.Open("./checker.db", 0666)
	if err != nil {
		log.Fatalf("error opening db: %s", err)
	}
	defer db.Close()

	c := cron.New()

	// Create collections.
	buckets := []string{"urls"}
	db.Update(func(tx *bolt.Tx) error {
		for _, v := range buckets {
			b := tx.Bucket([]byte(v))
			if b == nil {
				tx.CreateBucket([]byte(v))
			}
		}
		return nil
	})

	// Initialize for each of the existing URLs
	var items []*Check
	if err = GetAllChecks(db, &items); err != nil {
		log.Fatalf("error loading checks: %s", err)
	}

	for _, v := range items {
		// Trigger the update now...
		go v.Update(db)

		// ... and add a cron task for later.  Note that we pull out the ID into a
		// new variable so that we don't keep the entire Check structure from being
		// garbage collected.
		id := v.ID
		c.AddFunc(v.Schedule, func() {
			TryUpdate(db, id)
		})
	}

	// Start our cron scheduler.
	c.Start()
	defer c.Stop()

	goji.Use(DbInjectMiddleware(db))
	goji.Use(CronInjectMiddleware(c))
	goji.Get("/", IndexRoute)
	goji.Post("/addnew", NewCheckRoute)
	goji.Post("/update/:id", UpdateCheckRoute)
	goji.Post("/delete/:id", DeleteCheckRoute)
	goji.Post("/seen/:id", SeenCheckRoute)
	goji.Get("/stats", StatsRoute)
	goji.Get("/static/*", http.StripPrefix("/static/",
		http.FileServer(http.Dir("./static"))))
	goji.Serve()

	log.Print("Finishing...")
}
