package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/charles-l/gitamite"
	"github.com/tucnak/climax"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

func errx(code int, s string) {
	fmt.Fprintf(os.Stderr, "gitamite: %s\n", s)
	os.Exit(code)
}

func makeRequest(args []string, f func(url.URL, []byte) *http.Response) {
	u := url.URL{
		Scheme: "http",
		Host:   "localhost:8000", // TODO: read this from the config file
		Path:   "/repo",
	}

	// TODO: generalize the request struct when needed
	if len(args) < 2 {
		errx(1, "need a name")
	}

	a, err := gitamite.CreateAuthRequest(struct {
		Name string
	}{
		args[1],
	})
	if err != nil {
		errx(1, err.Error())
	}

	blob, _ := json.Marshal(a)
	r := f(u, blob)

	if err != nil || r.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(r.Body)
		errx(2, "request to remote failed: "+string(b))
	}
}

func createRepoRequest(ctx climax.Context) int {
	makeRequest(ctx.Args, func(u url.URL, blob []byte) *http.Response {
		r, err := http.Post(u.String(), "text/json", bytes.NewReader(blob))
		if err != nil {
			errx(3, err.Error())
		}
		return r
	})
	return 0
}

func deleteRepoRequest(ctx climax.Context) int {
	makeRequest(ctx.Args, func(u url.URL, blob []byte) *http.Response {
		d, _ := http.NewRequest(http.MethodDelete, u.String(), bytes.NewReader(blob))
		client := &http.Client{}
		r, err := client.Do(d)
		if err != nil {
			errx(3, err.Error())
		}
		return r
	})
	return 0
}

func main() {
	cli := climax.New("gitamite")
	cli.Brief = "gitamite client"
	cli.Version = "1.0"

	createCmd := climax.Command{
		Name:   "create",
		Brief:  "creates a new repo",
		Usage:  "[REPO]",
		Help:   "creates a new repo",
		Handle: createRepoRequest,
	}
	cli.AddCommand(createCmd)

	deleteCmd := climax.Command{
		Name:   "delete",
		Brief:  "deletes a repo",
		Usage:  "[REPO]",
		Help:   "deletes a repo",
		Handle: deleteRepoRequest,
	}
	cli.AddCommand(deleteCmd)

	cli.Run()
	return
}
