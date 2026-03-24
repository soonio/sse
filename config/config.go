package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

const (
	StreamName         = "pusher:message:stream"
	ConsumerGroup      = "pusher:message:consumer:group"
	ConsumerNamePatten = "pusher:message:consumer:0001"
)

type Redis struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	Database int    `yaml:"database"`
}

type Config struct {
	Addr  string `yaml:"addr"`
	Redis Redis  `yaml:"redis"`
	Debug bool   `yaml:"debug"`
}

func Load(f string, c *Config) error {
	bs, err := os.ReadFile(f)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(bs, c)
}

func MustLoad(f string, c *Config) {
	var err = Load(f, c)
	if err != nil {
		panic(err)
	}
}
