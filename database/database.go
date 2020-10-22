package database

import (
	"encoding/json"
	"log"
	"time"

	bolt "go.etcd.io/bbolt"
)

// DB - Bold db connection
var db *bolt.DB

func init() {
	var err error
	db, err = bolt.Open("data/data.db", 0600, &bolt.Options{Timeout: 10 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	// Closing DB in main.go defer
}

// GetGuildSettings - Get settings struct for a guild
func GetGuildSettings(gid string) (gs GuildSettings, err error) {
	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("guilds"))
		if err != nil {
			return err
		}
		// Decode settings
		v := b.Get([]byte(gid))
		if v == nil {
			// Insert new doc
			gs.ID = gid
			gs.EnabledChannels = make(map[string]ReactiveChannel)
			bts, err := json.Marshal(gs)
			if err != nil {
				return err
			}
			return b.Put([]byte([]byte(gid)), []byte(bts))
		}
		return json.Unmarshal(v, &gs)
	})
	return gs, err
}

// UpdateGuildSettings - Update guild setting in DB
func UpdateGuildSettings(gs GuildSettings) (err error) {
	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("guilds"))
		if err != nil {
			return err
		}
		// Encode settings and update db
		bts, err := json.Marshal(gs)
		return b.Put([]byte([]byte(gs.ID)), bts)
	})
	return err
}

// CloseDB - Close DB connection
func CloseDB() {
	db.Close()
}
