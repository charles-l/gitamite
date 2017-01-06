package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type AuthRequest struct {
	Signature, Data []byte
}

func main() {
	f, _ := ioutil.ReadFile("/tmp/file")
	s, _ := ioutil.ReadFile("/tmp/file.asc")

	a := AuthRequest{s, f}

	j, _ := json.Marshal(a)

	u := url.URL{
		Scheme: "http",
		Host:   "localhost:8000",
		Path:   "/create",
	}

	resp, err := http.Post(u.String(), "text/json", bytes.NewReader(j))
	if err != nil {
		fmt.Printf("%g\n", err)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("%s\n", body)
}
