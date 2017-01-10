package gitamite

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
)

var GlobalConfig *Config

const ConfigPath = "/etc/gitamite.conf"

// TODO: use different config for client and server
type Config struct {
	ServerAddr string // client
	RepoDir    string // server

	Auth struct {
		PrivateKeyring string // client
		PublicKeyring  string // server
	}
}

func LoadConfig() {
	GlobalConfig = ParseConfig(ConfigPath)
	if GlobalConfig == nil {
		fmt.Fprintf(os.Stderr, "Need config file in %s\n", ConfigPath)
		return
	}
}

// TODO: remove global and use a closure instead
func GetConfigValue(field string) (string, error) {
	r := reflect.ValueOf(GlobalConfig)
	if f := reflect.Indirect(r).FieldByName(field); f.IsValid() && f.String() != "" {
		return f.String(), nil
	} else {
		return "", fmt.Errorf("config value '%s' not set", field)
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
