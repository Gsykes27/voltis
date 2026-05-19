package runtime

import (
	"encoding/json"
	"os"
)

type Config struct {
	AppDir  string `json:"appDir"`
	DistDir string `json:"distDir"`
	HTTP    struct {
		Addr string `json:"addr"`
	} `json:"http"`
	Dev struct {
		VitePort int `json:"vitePort"`
	} `json:"dev"`
	Security struct {
		ActionSecret string `json:"actionSecret"`
	} `json:"security"`
}

func DefaultConfig() Config {
	var c Config
	c.AppDir = "./app"
	c.DistDir = "./dist"
	c.HTTP.Addr = ":3000"
	c.Dev.VitePort = 5173
	c.Security.ActionSecret = "dev-secret-change-me"
	return c
}

func LoadConfig(path string) (Config, error) {
	c := DefaultConfig()
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if c.AppDir == "" {
		c.AppDir = "./app"
	}
	if c.DistDir == "" {
		c.DistDir = "./dist"
	}
	if c.HTTP.Addr == "" {
		c.HTTP.Addr = ":3000"
	}
	if c.Dev.VitePort == 0 {
		c.Dev.VitePort = 5173
	}
	if c.Security.ActionSecret == "" {
		c.Security.ActionSecret = "dev-secret-change-me"
	}
	return c, nil
}
