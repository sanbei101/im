package config

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/phuslu/log"
	"gopkg.in/yaml.v3"
)

const DefaultConfigPath = "config.yaml"

type Config struct {
	API     APIConfig     `yaml:"api"`
	Gateway GatewayConfig `yaml:"gateway"`
	Store   StoreConfig   `yaml:"store"`
	Shard   ShardConfig   `yaml:"shard"`
	Storage StorageConfig `yaml:"storage"`
}

type StorageConfig struct {
	Endpoint        string `yaml:"endpoint"`
	Bucket          string `yaml:"bucket"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	UseSSL          bool   `yaml:"use_ssl"`
	PublicURLPrefix string `yaml:"public_url_prefix"`
}

type APIConfig struct {
	NodeID    string `yaml:"node_id"`
	Addr      string `yaml:"addr"`
	NodeIndex int    `yaml:"node_index"`
	NodeCount int    `yaml:"node_count"`
}

type GatewayConfig struct {
	Addr       string   `yaml:"addr"`
	APIAddrs   []string `yaml:"api_addrs"`
	MaxTimeout int      `yaml:"max_timeout"`
}

type StoreConfig struct {
	Path           string `yaml:"path"`
	BackupPath     string `yaml:"backup_path"`
	BackupInterval int    `yaml:"backup_interval_seconds"`
}

type ShardConfig struct {
	Slots int `yaml:"slots"`
}

func New(path ...string) *Config {
	p := DefaultConfigPath
	if len(path) > 0 && path[0] != "" {
		p = path[0]
	}
	cfg, err := Load(p)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
	return cfg
}

func NewTest() *Config {
	cfg := &Config{}
	cfg.Default()
	return cfg
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.Default()
	return cfg, cfg.Validate()
}

func (c *Config) Default() {
	if value := os.Getenv("API_NODE_ID"); value != "" {
		c.API.NodeID = value
	}
	if value := os.Getenv("API_ADDR"); value != "" {
		c.API.Addr = value
	}
	if c.API.NodeID == "" {
		c.API.NodeID = "api-1"
	}
	if c.API.Addr == "" {
		c.API.Addr = ":9000"
	}
	if c.Gateway.Addr == "" {
		c.Gateway.Addr = ":8800"
	}
	if len(c.Gateway.APIAddrs) == 0 {
		c.Gateway.APIAddrs = []string{"127.0.0.1:9000"}
	}
	if c.Gateway.MaxTimeout == 0 {
		c.Gateway.MaxTimeout = 10
	}
	if c.Store.Path == "" {
		c.Store.Path = "./data/" + c.API.NodeID
	}
	if c.Store.BackupPath == "" {
		c.Store.BackupPath = "./backup/" + c.API.NodeID
	}
	if c.Store.BackupInterval == 0 {
		c.Store.BackupInterval = 3600
	}
	if c.Shard.Slots == 0 {
		c.Shard.Slots = 1024
	}
	if value := os.Getenv("GATEWAY_ADDR"); value != "" {
		c.Gateway.Addr = value
	}
	if value := os.Getenv("STORE_PATH"); value != "" {
		c.Store.Path = value
	}
	if value := os.Getenv("BACKUP_PATH"); value != "" {
		c.Store.BackupPath = value
	}
	if value := os.Getenv("SHARD_SLOTS"); value != "" {
		if slots, err := strconv.Atoi(value); err == nil {
			c.Shard.Slots = slots
		}
	}
	if c.API.NodeCount == 0 {
		c.API.NodeCount = len(c.Gateway.APIAddrs)
	}
	if c.Storage.Endpoint == "" {
		c.Storage.Endpoint = "127.0.0.1:9000"
	}
	if c.Storage.Bucket == "" {
		c.Storage.Bucket = "im-media"
	}
	if c.Storage.AccessKeyID == "" {
		c.Storage.AccessKeyID = "minioadmin"
	}
	if c.Storage.SecretAccessKey == "" {
		c.Storage.SecretAccessKey = "minioadmin"
	}
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.API.NodeID) == "" || strings.TrimSpace(c.API.Addr) == "" {
		return errors.New("api node_id and addr are required")
	}
	if len(c.Gateway.APIAddrs) == 0 {
		return errors.New("at least one api address is required")
	}
	if c.Shard.Slots < 1 {
		return errors.New("shard slots must be positive")
	}
	if c.API.NodeIndex < 0 || c.API.NodeIndex >= c.API.NodeCount {
		return errors.New("api node_index is outside node_count")
	}
	return nil
}
