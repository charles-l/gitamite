package gitamite

import (
	"bytes"
	"encoding/json"
	"golang.org/x/crypto/openpgp"
	"io/ioutil"
)

type AuthRequest struct {
	Signature []byte
	Data      interface{}
}

// TODO make sure to memoize
func readKeyringFile(path string) (openpgp.EntityList, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	keyring, err := openpgp.ReadKeyRing(bytes.NewReader(f))
	if err != nil {
		return nil, err
	}

	return keyring, err
}

func (r AuthRequest) VerifyRequest() error {
	// FIXME config keyring loc
	keyring, _ := readKeyringFile("/home/nc/.gnupg/pubring.gpg")

	blob, _ := json.Marshal(r.Data) // FIXME: unmarshaling then marshaling again
	if _, err := openpgp.CheckArmoredDetachedSignature(keyring,
		bytes.NewReader(blob),
		bytes.NewReader(r.Signature)); err != nil {
		return err
	}
	return nil
}

func CreateAuthRequest(data interface{}) (AuthRequest, error) {
	// FIXME config keyring loc
	keyring, _ := readKeyringFile("/home/nc/.gnupg/secring.gpg")

	r := AuthRequest{}
	r.Data = data
	sig := bytes.NewBufferString("")
	blob, _ := json.Marshal(data)
	err := openpgp.ArmoredDetachSign(sig, keyring[0], bytes.NewReader(blob), nil)
	if err != nil {
		return AuthRequest{}, err
	}

	r.Signature = sig.Bytes()

	return r, nil
}
