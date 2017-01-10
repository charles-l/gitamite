package gitamite

import (
	"encoding/json"
	"io/ioutil"
)

var GlobalConfig *Config

const ConfigPath = "/etc/gitamite.conf"

type Config struct {
	RepoDir string
	Auth    struct {
		PrivateKeyring string
		PublicKeyring  string
	}
}

func ParseConfig(path string) *Config {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}
	var c Config
	err = json.Unmarshal(f, &c)
	if err != nil {
		println("invalid config: " + err.Error())
		return nil
	}
	return &c
}

func WriteConfig(c *Config, path string) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0600)
}
