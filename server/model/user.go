package model

import (
	"bytes"
	"fmt"
	"github.com/charles-l/gitamite"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"strings"
)

func main() {
	fmt.Println("vim-go")
}

type User struct {
	Name   string
	Email  string
	Entity *openpgp.Entity
}

func ArmoredPublicKey(u *User) *bytes.Buffer {
	b := bytes.NewBuffer([]byte{})
	w, err := armor.Encode(b, "PUBLIC KEY BLOCK", map[string]string{})
	if err != nil {
		return nil
	}
	u.Entity.Serialize(w)
	w.Close()
	return b
}

func UserFromEmail(email string) *User {
	p, err := gitamite.GetConfigValue("pubkeyring_path")
	if err != nil {
		return nil
	}
	keys, _ := gitamite.ReadKeyringFile(p)

	var u User

	for _, e := range keys {
		for k, _ := range e.Identities {
			s := strings.Split(k, "<")
			m := s[1][:len(s[1])-1]
			if m == email {
				u.Name = s[0]
				u.Email = m
				u.Entity = e
			}
		}
	}

	if u == (User{}) {
		return nil
	}
	return &u
}
