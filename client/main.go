package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/charles-l/gitamite"
	"github.com/tucnak/climax"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

func errx(code int, s string) {
	fmt.Fprintf(os.Stderr, "gitamite: %s\n", s)
	os.Exit(code)
}

func createRepoRequest(ctx climax.Context) int {
	u := url.URL{
		Scheme: "http",
		Host:   "localhost:8000",
		Path:   "/repo",
	}

	if len(ctx.Args) < 2 {
		errx(1, "need a name")
	}
	a, err := gitamite.CreateAuthRequest(struct {
		Name string
	}{
		ctx.Args[1],
	})
	if err != nil {
		log.Fatal(err)
	}

	j, _ := json.Marshal(a)
	r, err := http.Post(u.String(), "text/json", bytes.NewReader(j))
	if err != nil || r.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(r.Body)
		errx(2, "request to remote failed: "+string(b))
	}
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
	cli.Run()
	return
}
