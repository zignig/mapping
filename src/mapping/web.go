package main

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/golang-lru"
	"html/template"
	"io"
	"math"
	"strconv"
)

const (
	tileDepth   = 8
	maxDepth    = 18
	defaultSize = 1000
	port        = ":8099"
)

type WebHandler struct {
	router    *gin.Engine
	store     *store
	zoom      map[int]*lru.Cache
	sizes     map[int]int
	templates *template.Template
}

func NewWebServer() *WebHandler {
	wh := &WebHandler{}
	wh.router = gin.Default()
	wh.router.GET("/tiles/:zoom/:x/:y", wh.GetTile)

	// templates
	templ := template.New("")
	data, err := Asset("asset/index.html")
	if err != nil {
		fmt.Println("Asset error ", err)
	}
	_, err = templ.New("index.html").Parse(string(data))
	if err != nil {
		fmt.Println("Template error ", err)
	}
	wh.templates = templ
	wh.router.GET("/", wh.Index)

	// static assets
	wh.router.GET("/static/*path", wh.Static)
	// build the caches
	var t int
	wh.zoom = make(map[int]*lru.Cache)
	wh.sizes = make(map[int]int)
	for t = 0; t <= tileDepth; t++ {
		size := math.Pow(2, float64(t)*2.0)
		fmt.Println(t, ":", size)
		c, err := lru.New(int(size))
		if err != nil {
			return nil
		}
		wh.sizes[t] = int(size)
		wh.zoom[t] = c
	}
	for t = tileDepth + 1; t <= maxDepth; t++ {
		fmt.Println(t, ":", defaultSize)
		c, err := lru.New(defaultSize)
		if err != nil {
			return nil
		}
		wh.sizes[t] = int(defaultSize)
		wh.zoom[t] = c
	}

	if err != nil {
		fmt.Println(err)
	}

	// init the store
	wh.store, err = NewStore("")
	if err != nil {
		fmt.Println(err)
	}
	return wh
}

func (w *WebHandler) Run() {
	w.router.Run(port)
}

func (w *WebHandler) Static(c *gin.Context) {
	path := c.Params.ByName("path")
	fmt.Println(path)
	data, err := Asset("asset/static" + path)
	if err != nil {
		fmt.Println("Asset Error ", err)
	}
	size := int64(len(data))
	c.Writer.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	io.Copy(c.Writer, bytes.NewReader(data))
}

func (w *WebHandler) Index(c *gin.Context) {
	data := gin.H{
		"Maxdepth": maxDepth,
	}
	fmt.Println(data)
	w.templates.ExecuteTemplate(c.Writer, "index.html", data)
}

func (w *WebHandler) GetTile(c *gin.Context) {
	var size int64
	zoom := c.Params.ByName("zoom")
	zoomint, err := strconv.Atoi(zoom)
	if err != nil {
		fmt.Println("zoom fail ", err)
		return
	}
	x := c.Params.ByName("x")
	y := c.Params.ByName("y")
	path := zoom + "/" + x + "/" + y
	c.Writer.Header().Set("Content-Type", "image/png")
	data, ok := w.zoom[zoomint].Get(path)
	if !ok {
		//fmt.Println("no cache in", zoomint, "-", w.zoom[zoomint].Len(), " of ", w.sizes[zoomint])
		data, err := w.store.Get(path)
		if err != nil {
			data, err = FetchTile(path)
			w.store.Put(path, data)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		w.zoom[zoomint].Add(path, data)
		size = int64(len(data.([]byte)))
		//fmt.Println(path, " ", size)
		c.Writer.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		io.Copy(c.Writer, bytes.NewReader(data.([]byte)))
		return
	}
	if data != nil {
		size = int64(len(data.([]byte)))
		c.Writer.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		io.Copy(c.Writer, bytes.NewReader(data.([]byte)))
	} else {
		fmt.Println("Nil data ")
	}
}
