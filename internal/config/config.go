package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Addr  string `yaml:"Addr"` // 监听地址
	Redis struct {
		Addr     string `yaml:"Addr"`     // redis地址 127.0.0.1:6379
		Password string `yaml:"Password"` // redis密码
		Database int    `yaml:"Database"` // redis DB
	} `yaml:"Redis"`
}

func (c *Config) Load(filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(content, c)
}

func (c *Config) MustLoad(filename string) {
	if err := c.Load(filename); err != nil {
		panic(err)
	}
}
