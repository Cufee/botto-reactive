package database

import (
	"encoding/json"
	"fmt"
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
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("guilds"))
		if b == nil {
			if b, err = addBucket("guilds"); err != nil {
				return err
			}
		}
		// Decode settings
		v := b.Get([]byte(gid))
		if v == nil {
			// Insert new doc
			gs.ID = gid
			gs.EnabledChannels = make(map[string]ReactiveChannel)
			return UpdateGuildSettings(gs)
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

// GetVerificationMsg - Get verification message details
func GetVerificationMsg(mid string) (vmsg VerificationMessage, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("ver_messages"))
		if b == nil {
			if b, err = addBucket("users"); err != nil {
				return err
			}
		}
		// Decode settings
		v := b.Get([]byte(mid))
		if v == nil {
			return fmt.Errorf("verification message data not found")
		}
		return json.Unmarshal(v, &vmsg)
	})
	return vmsg, err
}

// AddVerificationMsg - Add verification message details
func AddVerificationMsg(vmsg VerificationMessage) error {
	err := db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("ver_messages"))
		if err != nil {
			return err
		}
		// Encode settings and update db
		bts, err := json.Marshal(vmsg)
		if err != nil {
			return err
		}
		return b.Put([]byte([]byte(vmsg.ID)), bts)
	})
	return err
}

// DelVerificationMsg - Relete verification message details
func DelVerificationMsg(vmsg VerificationMessage) error {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("ver_messages"))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}
		return b.Delete([]byte([]byte(vmsg.ID)))
	})
	return err
}

// GetUserData - Get user reactions data
func GetUserData(uid string) (user User, err error) {
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		if b == nil {
			if b, err = addBucket("users"); err != nil {
				return err
			}
		}
		// Decode settings
		v := b.Get([]byte(uid))
		if v == nil {
			// Add new user
			user := User{ID: uid, RoleRequests: make(map[string]map[string]Request)}
			if err := UpdateUserData(user); err != nil {
				return err
			}
			return nil
		}
		return json.Unmarshal(v, &user)
	})
	return user, err
}

// UpdateUserData - Update user reactions data
func UpdateUserData(user User) (err error) {
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		if b == nil {
			if b, err = addBucket("users"); err != nil {
				return err
			}
		}
		// Encode settings and update db
		bts, err := json.Marshal(user)
		if err != nil {
			return err
		}
		return b.Put([]byte([]byte(user.ID)), bts)
	})
	return err
}

// CompleteUserRequest - Set user request Active to false
func CompleteUserRequest(uid string, cid string, roleID string) error {
	// Get user data
	userData, err := GetUserData(uid)
	if err != nil {
		return err
	}
	delete(userData.RoleRequests[cid], roleID)
	return UpdateUserData(userData)
}

func addBucket(name string) (b *bolt.Bucket, err error) {
	return b, db.Update(func(tx *bolt.Tx) error {
		b, err = tx.CreateBucket([]byte(name))
		return err
	})
}

// CloseDB - Close DB connection
func CloseDB() {
	db.Close()
}
