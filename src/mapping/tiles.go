package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	server = "tile.openstreetmap.org"
)

func FetchTile(path string) (data interface{}, err error) {
	url := "http://" + server + "/" + path
	fmt.Println(url)
	resp, err := http.Get(url)
	fmt.Println(resp.Status, err)
	if err != nil {
		fmt.Println(err)
		return nil, err
	} else {
		defer resp.Body.Close()
		data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		return data, nil
	}
}
