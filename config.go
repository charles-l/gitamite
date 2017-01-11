package gitamite

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
)

const serverConfigPath = "/etc/gitamite.conf"
const clientConfigPath = ".gitamiterc"

type ConfigType int

const (
	Client ConfigType = iota
	Server
)

var config *map[string]interface{}

func LoadConfig(c ConfigType) {
	var p string
	switch c {
	case Client:
		p = path.Join(os.Getenv("HOME"), clientConfigPath)
	case Server:
		p = serverConfigPath
	}
	config = ParseConfig(p)

	if config == nil {
		fmt.Fprintf(os.Stderr, "Need config a valid config file %s\n", p)
		return
	}
}

func GetConfigValue(field string) (string, error) {
	if v, ok := (*config)[field]; ok {
		return v.(string), nil
	}
	log.Printf("field not found '%s'", field)
	return "", fmt.Errorf("key not found")
}

func ParseConfig(path string) *map[string]interface{} {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	err = json.Unmarshal(f, &m)
	if err != nil {
		log.Printf("invalid config: %s", err.Error())
		return nil
	}
	return &m
}

func WriteConfig(c *map[string]interface{}, p string) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(p, data, 0600)
}
