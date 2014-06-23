package main

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/robfig/cron"
	"github.com/zenazn/goji/web"
)

func RouteChecksGetAll(c web.C, w http.ResponseWriter, r *http.Request) {
	db := c.Env["db"].(*bolt.DB)

	checks := []*Check{}
	err := GetAllChecks(db, &checks)
	if err != nil {
		panic(err)
	}

	err = json.NewEncoder(w).Encode(checks)
	if err != nil {
		panic(err)
	}
}

func RouteChecksNew(c web.C, w http.ResponseWriter, r *http.Request) {
	db := c.Env["db"].(*bolt.DB)

	params := struct {
		URL      string `json:"url"`
		Selector string `json:"selector"`
		Schedule string `json:"schedule"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		panic(err)
	}

	if len(params.URL) == 0 {
		http.Error(w, "missing URL parameter", http.StatusBadRequest)
		return
	}
	if len(params.Selector) == 0 {
		http.Error(w, "missing Selector parameter", http.StatusBadRequest)
		return
	}
	if len(params.Schedule) == 0 {
		http.Error(w, "missing Schedule parameter", http.StatusBadRequest)
		return
	}

	check := Check{
		URL:      params.URL,
		Selector: params.Selector,
		Schedule: params.Schedule,
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
}

/*
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
*/
