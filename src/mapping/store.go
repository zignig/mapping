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
	bucket []byte
	path string
}

func NewStore(path, bucket string) (s *store, err error) {
	s = &store{}
	if path == "" {
		path = "./tiles.db"
	}
	if "" == bucket {
		bucket = "tiles"
	}
	s.path = path
	db, err := bolt.Open(s.path, 0600, nil)
	if err != nil {
		fmt.Println(err)
	}

	s.bucket = []byte(bucket)
	err = db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists(s.bucket)
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
	//fmt.Println("store ", path)
	err = s.db.View(func(tx *bolt.Tx) error {
		data = tx.Bucket(s.bucket).Get([]byte(path))
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
		b := tx.Bucket(s.bucket)
		b.Put([]byte(path), v.([]byte))
		return nil
	})
	s.lock.Unlock()
	return
}

func (s *store) Remove(path string) (err error) {
	s.lock.Lock()
	err = s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		b.Delete([]byte(path))
		return nil
	})
	s.lock.Unlock()
	return
}
