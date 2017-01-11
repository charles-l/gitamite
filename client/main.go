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
	host, err := gitamite.GetConfigValue("server_addr")
	if err != nil {
		errx(1, err.Error())
	}
	u := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   "/repo",
	}

	// TODO: generalize the request struct when needed
	if len(args) < 1 {
		errx(1, "need a name")
	}

	a, err := gitamite.CreateAuthRequest(struct {
		Name string
	}{
		args[0],
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
		r, err := http.Post(u.String(), "application/json", bytes.NewReader(blob))
		if err != nil {
			errx(3, err.Error())
		}
		return r
	})
	return 0
}

func deleteRepoRequest(ctx climax.Context) int {
	var inp string
	fmt.Printf("Are you SURE you want to delete this repo? If so, type its name again:\n")
	fmt.Scanln(&inp)
	if len(ctx.Args) == 0 || ctx.Args[0] != inp {
		errx(0, "Not deleting repo")
	}

	makeRequest(ctx.Args, func(u url.URL, blob []byte) *http.Response {
		d, _ := http.NewRequest(http.MethodDelete, u.String(), bytes.NewReader(blob))
		d.Header.Set("Content-Type", "application/json")
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
	gitamite.LoadConfig()

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
