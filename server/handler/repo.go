package handler

import (
	"encoding/json"
	"github.com/charles-l/gitamite"
	"github.com/charles-l/gitamite/server/helper"
	"io/ioutil"
	"log"
	"net/http"
)

type CreateHandler struct {
	Render helper.Renderer
}

func (h CreateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	blob, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "error in body: "+err.Error(), 400)
	}

	var a gitamite.AuthRequest
	err = json.Unmarshal(blob, &a)
	if err != nil {
		http.Error(w, "malformed json "+err.Error(), 400)
	}

	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if len(a.Signature) == 0 || a.Data == nil {
		http.Error(w, "Need data and signature", 400)
		return
	}

	if err := a.VerifyRequest(); err != nil {
		http.Error(w, "Invalid signature", 401)
		return
	}

	log.Printf("creating new repo: %s", a.Data.(map[string]interface{})["Name"])
}
