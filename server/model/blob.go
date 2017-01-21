package model

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"github.com/boltdb/bolt"
	"html"
	"html/template"

	"github.com/charles-l/pygments"
)

type Blob struct {
	Path string
	Type string
	Data [][]byte
}

type Blame struct {
	Users []*User
	*Blob
}

func (b *Blob) ByteArray() []byte {
	return bytes.Join(b.Data, []byte(""))
}

// TODO: possibly do this for known blobs in a separate thread when staring the server?
func HighlightedBlobHTML(b *Blob) template.HTML {
	m := md5.New()
	m.Write(b.ByteArray())
	k := []byte("blob:" + hex.EncodeToString(m.Sum(nil)))

	var htmlBlob []byte

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blobCache"))
		if e := b.Get(k); e != nil {
			htmlBlob = e
		}
		return nil
	})

	if htmlBlob != nil {
		return template.HTML(string(htmlBlob))
	}

	h, err := pygments.Highlight(b.ByteArray(), b.Type, "html", "utf-8")
	if err != nil {
		h = "<pre>" + html.EscapeString(string(b.ByteArray())) + "</pre>"
	}
	r := template.HTML(h)

	// theoretically this is safe in a goroutine
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("blobCache"))
		b.Put(k, []byte(h))
		return nil
	})

	return r
}
