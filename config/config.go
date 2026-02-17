package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Slots     int `yaml:"slots"`
		SlotSleep int `yaml:"slot_sleep"`
	} `yaml:"server"`

	P2P struct {
		LowConns int  `yaml:"low_conns"`
		HiConns  int  `yaml:"hi_conns"`
		IsRelay  bool `yaml:"is_relay"`
	} `yaml:"p2p"`

	Hosts struct {
		Whitelist []string `yaml:"whitelist"`
		Blacklist []string `yaml:"blacklist"`
	} `yaml:"hosts"`
}

func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.Server.Slots = 5
	cfg.Server.SlotSleep = 1
	cfg.P2P.LowConns = 30
	cfg.P2P.HiConns = 50
	cfg.P2P.IsRelay = false
	cfg.Hosts.Whitelist = []string{"api.themoviedb.org", "images.tmdb.org"}
	return cfg
}

func Load() (*Config, error) {
	cfg := DefaultConfig()
	dir := filepath.Dir(os.Args[0])
	filename := filepath.Join(dir, "tuns.yaml")

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			err = cfg.Save()
			return cfg, err
		}
		return nil, err
	}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (cfg *Config) Save() error {
	dir := filepath.Dir(os.Args[0])
	filename := filepath.Join(dir, "tuns.yaml")

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
