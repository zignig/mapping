package main

import (
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"sync"
)

type store struct {
	db   *bolt.DB
	lock sync.Mutex
}

func NewStore(path string) (s *store, err error) {
	s = &store{}
	if path == "" {
		path = "/tmp/tiles.db"
	}
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		fmt.Println(err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("tiles"))
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
	s.db = db
	return s, nil
}

func (s *store) Close() {
	s.db.Close()
}

func (s *store) Get(path string) (v interface{}, err error) {
	var data []byte
	fmt.Println("store ", path)
	err = s.db.View(func(tx *bolt.Tx) error {
		data = tx.Bucket([]byte("tiles")).Get([]byte(path))
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
	if len(data) == 0 {
		return nil, errors.New("Nodata")
	}
	return data, nil
}

func (s *store) Put(path string, v interface{}) (err error) {
	s.lock.Lock()
	err = s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tiles"))
		b.Put([]byte(path), v.([]byte))
		return nil
	})
	s.lock.Unlock()
	return
}
