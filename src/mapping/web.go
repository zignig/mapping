package main

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/golang-lru"
	"io"
	"strconv"
)

type WebHandler struct {
	router *gin.Engine
	cache  *lru.Cache
	store  *store
}

func NewWebServer() *WebHandler {
	wh := &WebHandler{}
	wh.router = gin.New()
	wh.router.GET("/:zoom/:x/:y", wh.GetTile)

	var err error
	wh.cache, err = lru.New(20000)
	if err != nil {
		fmt.Println(err)
	}

	wh.store, err = NewStore("")
	if err != nil {
		fmt.Println(err)
	}
	return wh
}

func (w *WebHandler) Run() {
	w.router.Run(":8099")
}

func (w *WebHandler) GetTile(c *gin.Context) {
	var size int64
	zoom := c.Params.ByName("zoom")
	x := c.Params.ByName("x")
	y := c.Params.ByName("y")
	path := zoom + "/" + x + "/" + y
	c.Writer.Header().Set("Content-Type", "image/png")
	data, ok := w.cache.Get(path)
	if !ok {
		data, err := w.store.Get(path)
		if err != nil {
			fmt.Println(err, " not in store")
			data, err = FetchTile(path)
			w.store.Put(path, data)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		w.cache.Add(path, data)
		size = int64(len(data.([]byte)))
		fmt.Println(path, " ", size)
		c.Writer.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		io.Copy(c.Writer, bytes.NewReader(data.([]byte)))
		return
	}
	fmt.Println("serving ", path)
	if data != nil {
		size = int64(len(data.([]byte)))
		c.Writer.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		io.Copy(c.Writer, bytes.NewReader(data.([]byte)))
	} else {
		fmt.Println("Nil data ")
	}
}
