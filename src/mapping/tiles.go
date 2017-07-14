package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"github.com/hashicorp/golang-lru"
	"errors"
	"sync"
)

const (
	server = "tile.openstreetmap.org"
)

type tileStore struct {
	zoom           map[string]*lru.Cache
	store          *store

	waitRemote     map[string]*sync.Mutex
	waitRemoteLock sync.Mutex

	reqNumLock sync.Mutex
	reqNum int
}

// zoom level and cache size
var zoomLevelCacheSizes = map[string]int{
	"0" : 1,
	"1" : 4,
	"2" : 16,
	"3" : 64,
	"4" : 256,
	"5" : 1024,
	"6" : 1024,
	"7" : 1024,
	"8" : 1024,
	"9" : 1024,
	"10" : 1024,
	"11" : 1024,
	"12" : 1024,
	"13" : 1024,
	"14" : 1024,
	"15" : 1024,
	"16" : 1024,
	"17" : 1024,
	"18" : 1024,
}

func NewTileStore() *tileStore {
	ts := new(tileStore)

	// init our list of tiles to wait for while fetching
	ts.waitRemote = make(map[string]*sync.Mutex)

	// build the caches
	ts.zoom = make(map[string]*lru.Cache)

	for k, size := range zoomLevelCacheSizes {
		c, err := lru.New(size)
		if nil != err {
			fmt.Println(k, err)
		}
		ts.zoom[k] = c
	}

	// init the store
	var err error
	ts.store, err = NewStore("", "")
	if err != nil {
		fmt.Println(err)
	}

	return ts
}

func (ts *tileStore)debugMessage(message string, a ...interface{}) {
	if debug {
		fmt.Printf(message + "\n", a...)
	}
}

func (ts *tileStore)fetch(zoom, x, y string) ([]byte, error) {
	path := zoom + "/" + x + "/" + y
	ts.reqNumLock.Lock()
	ts.reqNum++
	reqNum := ts.reqNum
	ts.reqNumLock.Unlock()
	ts.debugMessage("#%d: Fetching: %s", reqNum, path)

	// is it in memory?
	if tile, err := ts.fetchLru(zoom, path); err == nil {
		return tile, nil
	}

	// get it from the db, if we have it
	if tile, err := ts.fetchDb(path); err == nil {
		ts.storeLru(zoom, path, tile)
		return tile, nil
	}

	ts.debugMessage("#%d: Miss: %s is not stored locally", reqNum, path)
	// not local? fetch from remote
	ts.debugMessage("#%d: Locking for tile fetch %s...", reqNum, path)
	ts.waitRemoteLock.Lock();
	ts.debugMessage("#%d:  -> Got Lock for tile acquisition %s!", reqNum, path)
	if _, ok := ts.waitRemote[path]; !ok {
		ts.debugMessage("#%d:  -> %s is not in currently wait list", reqNum, path)
		ts.waitRemote[path] = &sync.Mutex{}
	}
	ts.debugMessage("#%d:  -> Locking tile %s...", reqNum, path)
	ts.waitRemoteLock.Unlock();
	ts.waitRemote[path].Lock();
	ts.debugMessage("#%d:  -> tile locked %s...", reqNum, path)
	ts.debugMessage("#%d:  -> global tile lock released %s...", reqNum, path)
	defer func() {
		ts.debugMessage("#%d: Unlocking tile %s...", reqNum, path)
		ts.waitRemoteLock.Lock();
		ts.debugMessage("#%d:  -> global tile lock aquired %s", reqNum, path)
		ts.waitRemote[path].Unlock()
		ts.debugMessage("#%d:  -> tile lock released %s", reqNum, path)
		ts.waitRemoteLock.Unlock();
		ts.debugMessage("#%d:  -> global tile lock released %s", reqNum, path)
	}()

	ts.debugMessage("#%d:  -> checking LRU... %s", reqNum, path)
	// did it appear while we were waiting this lock?
	if tile, err := ts.fetchLru(zoom, path); err == nil {
		ts.debugMessage("#%d:  -> Found: %s was loaded by another request", reqNum, path)
		return tile, nil
	}

	// no? ok - fetch it ourselves
	ts.debugMessage("#%d:  -> Fetching tile from upstream... %s", reqNum, path)
	tile, err := ts.fetchTile(path)
	if nil != err {
		ts.debugMessage("#%d:  -> NotFound: %s was not fetched", reqNum, path)
		return nil, err
	}

	// add to LRU
	ts.debugMessage("#%d:  -> updating LRU with tile... %s", reqNum, path)
	ts.storeLru(zoom, path, tile)

	// add to DB
	ts.debugMessage("#%d:  -> updating DB with tile... %s", reqNum, path)
	ts.store.Put(path, tile)
	ts.debugMessage("#%d: Found: %s was found remotely and cached locally", reqNum, path)

	return tile, nil
}

func (ts *tileStore)fetchLru(zoom, path string) ([]byte, error) {
	if _, exists := ts.zoom[zoom]; !exists {
		return nil, errors.New("LRU Not Cached: zoom level not supported")
	}
	data, ok := ts.zoom[zoom].Get(path)
	if !ok {
		ts.debugMessage("LRU NotFound: %s is not in cache", path)
		return nil, errors.New("Tile not found in LRU")
	}
	tile := data.([]byte)
	if 0 == len(tile) {
		ts.debugMessage("LRU Error: Cached tile %s was zero length", path)
		ts.zoom[zoom].Remove(path)
		return nil, errors.New("Busted Tile")
	}
	ts.debugMessage("LRU Found: %s in cache. Size=%d", path, len(tile))
	return tile, nil
}
func (ts *tileStore)storeLru(zoom, path string, tile []byte) {
	if _, exists := ts.zoom[zoom]; exists {
		ts.zoom[zoom].Add(path, tile)
	}
}


func (ts *tileStore)fetchDb(path string) ([]byte, error) {
	data, err := ts.store.Get(path)
	if nil != err {
		ts.debugMessage("DB NotFound: %s in DB cache", path)
		return nil, errors.New("Tile not found in DB")
	}
	tile := data.([]byte)
	if 0 == len(tile) {
		ts.debugMessage("DB Error: Cached tile %s was zero length", path)
		ts.store.Remove(path)
		return nil, errors.New("Busted Tile")
	}
	ts.debugMessage("DB Found: %s in cache. Size=%d", path, len(tile))
	return data.([]byte), nil
}

func (ts *tileStore)fetchTile(path string) (tile []byte, err error) {
    client := http.Client{}
	url := "http://" + server + "/" + path
	ts.debugMessage("fetching " + path + " as " + url)
	//resp, err := http.Get(url)
    req , err := http.NewRequest("GET",url,nil)
	if err != nil {
		ts.debugMessage("Req Error: %s", err)
		return nil, err
	} else {
        req.Header.Set("User-Agent","github.com/zignig/mapping_caching")
        resp , err := client.Do(req)
	    ts.debugMessage(" -> Status %s", resp.Status)
	    if err != nil {
            ts.debugMessage("Resp Error: %s", err)
    		return nil, err
    	}
		defer resp.Body.Close()
		tile, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			ts.debugMessage("Error: %s", err)
			return nil, err
		}
		ts.debugMessage("Remote Found: %s was fetched. Size=%d", path, len(tile))
		return tile, nil
	}
}
