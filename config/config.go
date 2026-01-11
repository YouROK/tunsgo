package config

import (
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Tuns []*TunConfig `yaml:"tuns"`

	DNS struct {
		ForwardTun        string `yaml:"forward_tun"`
		Listen            string `yaml:"listen"`
		Upstream          string `yaml:"upstream"`
		CacheTimeoutHours int    `yaml:"cache_timeout_hours"`
	} `yaml:"dns"`

	FillRouteTable bool `yaml:"fill_route_table"`
}

type TunConfig struct {
	TunName     string `yaml:"tun"`
	TableID     int    `yaml:"table_id"`
	ForwardMark uint32 `yaml:"fw_mark"`
}

var Cfg *Config

func Load() error {
	dir := filepath.Dir(os.Args[0])
	path := filepath.Join(dir, "config.yaml")

	Cfg = &Config{}

	file, err := os.ReadFile(path)
	if err != nil {
		setDefaults()
		err = Save()
		log.Println("Error reading config file:", err)
		log.Println("Using default configuration")
		log.Println("Please set config")
		if err != nil {
			log.Println("Error save def config:", err)
		}
		os.Exit(1)
		return err
	}

	if err = yaml.Unmarshal(file, Cfg); err != nil {
		log.Println("Error parse config:", err)
		return err
	}

	return nil
}

func setDefaults() {
	Cfg.Tuns = []*TunConfig{
		{
			TunName:     "tun0",
			TableID:     100,
			ForwardMark: 1,
		},
	}

	Cfg.DNS.Listen = ":53"
	Cfg.DNS.Upstream = "9.9.9.9:53"
	Cfg.DNS.CacheTimeoutHours = 24
	Cfg.DNS.ForwardTun = "tun0"

	Cfg.FillRouteTable = true
}

func Save() error {
	dir := filepath.Dir(os.Args[0])
	path := filepath.Join(dir, "config.yaml")

	data, err := yaml.Marshal(Cfg)
	if err != nil {
		return err
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
