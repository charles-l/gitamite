package model

import (
	"github.com/boltdb/bolt"
	"log"
)

var db *bolt.DB

func InitDB() *bolt.DB {
	var err error
	db, err = bolt.Open("/tmp/gitamite.db", 0666, nil)
	if err != nil {
		log.Fatal(err)
	}
	db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("blobCache"))
		return nil
	})
	return db
}
