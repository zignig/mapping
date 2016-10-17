package main

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"html/template"
	"io"
	"strconv"
	"net/http"
)

const (
	port = ":8099"
	maxDepth = 18
)

type WebHandler struct {
	router    *gin.Engine
	templates *template.Template
	ts        *tileStore
}

func getTemplate() *template.Template {
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
	return templ
}

func NewWebServer() *WebHandler {
	wh := &WebHandler{}
	wh.router = gin.Default()
	wh.router.GET("/tiles/:zoom/:x/:y", wh.GetTile)

	// templates
	wh.templates = getTemplate()
	wh.router.GET("/", wh.Index)

	// static assets
	wh.router.GET("/static/*path", wh.Static)
	wh.ts = NewTileStore()
	return wh
}

func (w *WebHandler) Run() {
	w.router.Run(port)
}

func (w *WebHandler) Static(c *gin.Context) {
	path := c.Params.ByName("path")
	if debug {
		fmt.Println(path)
	}
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
	if debug {
		fmt.Println(data)
	}
	if debug {
		getTemplate().ExecuteTemplate(c.Writer, "index.html", data)
	} else {
		w.templates.ExecuteTemplate(c.Writer, "index.html", data)
	}
}

func (w *WebHandler) GetTile(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			c.HTML(500, "error", r)
		}
	}()

	var size int64
	zoom := c.Params.ByName("zoom")
	x := c.Params.ByName("x")
	y := c.Params.ByName("y")

	img, err := w.ts.fetch(zoom, x, y)
	if err != nil {
		c.String(http.StatusNotFound, "Not Found %s", err)
	} else {
		c.Writer.Header().Set("Content-Type", "image/png")
		size = int64(len(img))
		c.Writer.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		io.Copy(c.Writer, bytes.NewReader(img))
	}
}
