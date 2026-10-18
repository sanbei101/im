package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/phuslu/log"
	"gopkg.in/yaml.v3"

	"github.com/sanbei101/im/pkg"
)

const defaultConfigPath = "config.yaml"

type Config struct {
	API       APIConfig     `yaml:"api"`
	Gateway   GatewayConfig `yaml:"gateway"`
	Meta      MetaConfig    `yaml:"meta"`
	Store     StoreConfig   `yaml:"store"`
	Shard     ShardConfig   `yaml:"shard"`
	Storage   StorageConfig `yaml:"storage"`
	JWTSecret string        `yaml:"jwt_secret"`
}

type StorageConfig struct {
	Endpoint      string `yaml:"endpoint"`
	Bucket        string `yaml:"bucket"`
	AccessKey     string `yaml:"access_key"`
	SecretKey     string `yaml:"secret_key"`
	Region        string `yaml:"region"`
	UseSSL        bool   `yaml:"use_ssl"`
	PathStyle     bool   `yaml:"path_style"`
	PublicBaseURL string `yaml:"public_base_url"`
}

type APIConfig struct {
	NodeID    string `yaml:"node_id"`
	Addr      string `yaml:"addr"`
	HTTPAddr  string `yaml:"http_addr"`
	PProfAddr string `yaml:"pprof_addr"`
	NodeIndex int    `yaml:"node_index"`
	NodeCount int    `yaml:"node_count"`
}

type GatewayConfig struct {
	Addr         string   `yaml:"addr"`
	APIAddrs     []string `yaml:"api_addrs"`
	APIHTTPAddrs []string `yaml:"api_http_addrs"`
}

type MetaConfig struct {
	ListenAddr     string `yaml:"listen_addr"`
	HTTPAddr       string `yaml:"http_addr"`
	PProfAddr      string `yaml:"pprof_addr"`
	StorePath      string `yaml:"store_path"`
	BackupPath     string `yaml:"backup_path"`
	BackupInterval int    `yaml:"backup_interval_seconds"`
}

type StoreConfig struct {
	Path           string `yaml:"path"`
	BackupPath     string `yaml:"backup_path"`
	BackupInterval int    `yaml:"backup_interval_seconds"`
}

type ShardConfig struct {
	Slots int `yaml:"slots"`
}

func New() *Config {
	cfg, err := Load(defaultConfigPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
	return cfg
}

// NewMeta loads the subset of config used by the Meta process; object storage
// and room sharding are not required there.
func NewMeta() *Config {
	cfg, err := LoadMeta(defaultConfigPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load meta config")
	}
	return cfg
}

func LoadMeta(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if err := cfg.Default(); err != nil {
		return nil, err
	}
	return cfg, cfg.ValidateMeta()
}

// NewTest returns a self-contained config for tests; it never reads the
// environment or the filesystem.
func NewTest() *Config {
	return &Config{
		API: APIConfig{
			NodeID: "api-1", Addr: ":9000", HTTPAddr: ":8801",
			PProfAddr: "127.0.0.1:6061", NodeCount: 1,
		},
		Gateway: GatewayConfig{
			Addr: ":8800", APIAddrs: []string{"127.0.0.1:9000"},
			APIHTTPAddrs: []string{"127.0.0.1:8801"},
		},
		Meta: MetaConfig{
			ListenAddr: ":8802", HTTPAddr: "127.0.0.1:8802",
			PProfAddr: "127.0.0.1:6063", StorePath: "data/meta",
			BackupPath: "backup/meta", BackupInterval: 3600,
		},
		Store: StoreConfig{Path: "data/api-1", BackupPath: "backup/api-1", BackupInterval: 3600},
		Shard: ShardConfig{Slots: 1024},
		Storage: StorageConfig{
			Endpoint: "127.0.0.1:19901", Bucket: "im-media",
			AccessKey: "rustfsadmin", SecretKey: "rustfsadmin",
			Region: "us-east-1", PathStyle: true, PublicBaseURL: "https://cdn.example.test",
		},
	}
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
	if err := cfg.Default(); err != nil {
		return nil, err
	}
	return cfg, cfg.Validate()
}

func (c *Config) Default() error {
	envString("API_NODE_ID", &c.API.NodeID)
	envString("API_ADDR", &c.API.Addr)
	envString("API_HTTP_ADDR", &c.API.HTTPAddr)
	envString("API_PPROF_ADDR", &c.API.PProfAddr)
	if c.API.NodeID == "" {
		c.API.NodeID = "api-1"
	}
	if c.API.Addr == "" {
		c.API.Addr = ":9000"
	}
	if c.API.HTTPAddr == "" {
		c.API.HTTPAddr = ":8801"
	}
	if c.API.PProfAddr == "" {
		c.API.PProfAddr = "127.0.0.1:6061"
	}
	if c.Gateway.Addr == "" {
		c.Gateway.Addr = ":8800"
	}
	if len(c.Gateway.APIAddrs) == 0 {
		c.Gateway.APIAddrs = []string{"127.0.0.1:9000"}
	}
	if len(c.Gateway.APIHTTPAddrs) == 0 {
		c.Gateway.APIHTTPAddrs = []string{"127.0.0.1:8801"}
	}
	if c.Meta.ListenAddr == "" {
		c.Meta.ListenAddr = ":8802"
	}
	if c.Meta.HTTPAddr == "" {
		c.Meta.HTTPAddr = "127.0.0.1:8802"
	}
	if c.Meta.PProfAddr == "" {
		c.Meta.PProfAddr = "127.0.0.1:6063"
	}
	if c.Meta.StorePath == "" {
		c.Meta.StorePath = "./data/meta"
	}
	if c.Meta.BackupPath == "" {
		c.Meta.BackupPath = "./backup/meta"
	}
	if c.Meta.BackupInterval == 0 {
		c.Meta.BackupInterval = 3600
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
	envString("GATEWAY_ADDR", &c.Gateway.Addr)
	envList("GATEWAY_API_ADDRS", &c.Gateway.APIAddrs)
	envList("GATEWAY_API_HTTP_ADDRS", &c.Gateway.APIHTTPAddrs)
	envString("META_LISTEN_ADDR", &c.Meta.ListenAddr)
	envString("META_HTTP_ADDR", &c.Meta.HTTPAddr)
	envString("META_PPROF_ADDR", &c.Meta.PProfAddr)
	envString("META_STORE_PATH", &c.Meta.StorePath)
	envString("META_BACKUP_PATH", &c.Meta.BackupPath)
	if err := envInt("META_BACKUP_INTERVAL_SECONDS", &c.Meta.BackupInterval); err != nil {
		return err
	}
	envString("STORE_PATH", &c.Store.Path)
	envString("BACKUP_PATH", &c.Store.BackupPath)
	if value := os.Getenv("SHARD_SLOTS"); value != "" {
		slots, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse SHARD_SLOTS %q: %w", value, err)
		}
		c.Shard.Slots = slots
	}
	if c.API.NodeCount == 0 {
		c.API.NodeCount = len(c.Gateway.APIAddrs)
	}
	if c.Storage.Region == "" {
		c.Storage.Region = "us-east-1"
	}
	envString("S3_ENDPOINT", &c.Storage.Endpoint)
	envString("S3_BUCKET", &c.Storage.Bucket)
	envString("S3_ACCESS_KEY", &c.Storage.AccessKey)
	envString("S3_ACCESS_SECRET", &c.Storage.SecretKey)
	envString("S3_REGION", &c.Storage.Region)
	if err := envBool("S3_USE_SSL", &c.Storage.UseSSL); err != nil {
		return err
	}
	if err := envBool("S3_PATH_STYLE", &c.Storage.PathStyle); err != nil {
		return err
	}
	envString("S3_PUBLIC_BASE_URL", &c.Storage.PublicBaseURL)
	return nil
}

func envString(key string, target *string) {
	if value := os.Getenv(key); value != "" {
		*target = value
	}
}

func envList(key string, target *[]string) {
	value := os.Getenv(key)
	if value == "" {
		return
	}
	parts := strings.Split(value, ",")
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			trimmed = append(trimmed, part)
		}
	}
	if len(trimmed) > 0 {
		*target = trimmed
	}
}

func envInt(key string, target *int) error {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("parse %s %q: %w", key, value, err)
	}
	*target = parsed
	return nil
}

func envBool(key string, target *bool) error {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("parse %s %q: %w", key, value, err)
	}
	*target = parsed
	return nil
}

func (c *Config) ValidateMeta() error {
	if strings.TrimSpace(c.Meta.ListenAddr) == "" || strings.TrimSpace(c.Meta.HTTPAddr) == "" {
		return errors.New("meta listen_addr and http_addr are required")
	}
	if strings.TrimSpace(c.Meta.StorePath) == "" || strings.TrimSpace(c.Meta.BackupPath) == "" {
		return errors.New("meta store_path and backup_path are required")
	}
	return nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.API.NodeID) == "" || strings.TrimSpace(c.API.Addr) == "" {
		return errors.New("api node_id and addr are required")
	}
	if strings.TrimSpace(c.API.HTTPAddr) == "" || strings.TrimSpace(c.API.PProfAddr) == "" {
		return errors.New("api http_addr and pprof_addr are required")
	}
	if len(c.Gateway.APIAddrs) == 0 {
		return errors.New("at least one api address is required")
	}
	if c.Storage.Endpoint == "" || c.Storage.Bucket == "" ||
		c.Storage.AccessKey == "" || c.Storage.SecretKey == "" || c.Storage.PublicBaseURL == "" {
		return errors.New(
			"storage endpoint, bucket, access_key, secret_key and public_base_url are required",
		)
	}
	if c.Shard.Slots < 1 || c.Shard.Slots > pkg.MaxSlots {
		return fmt.Errorf("shard slots must be within [1, %d]", pkg.MaxSlots)
	}
	if c.API.NodeIndex < 0 || c.API.NodeIndex >= c.API.NodeCount {
		return errors.New("api node_index is outside node_count")
	}
	// Slot ownership is computed as NodeIndex(slot, slots, node_count) on the
	// API side and looked up in api_addrs[index] on the gateway side, so the
	// two counts must agree or messages route to the wrong node.
	if c.API.NodeCount != len(c.Gateway.APIAddrs) {
		return errors.New("api node_count must match the number of gateway api_addrs")
	}
	if c.API.NodeCount != len(c.Gateway.APIHTTPAddrs) {
		return errors.New("api node_count must match the number of gateway api_http_addrs")
	}
	if strings.TrimSpace(c.Meta.ListenAddr) == "" || strings.TrimSpace(c.Meta.HTTPAddr) == "" {
		return errors.New("meta listen_addr and http_addr are required")
	}
	if strings.TrimSpace(c.Meta.StorePath) == "" || strings.TrimSpace(c.Meta.BackupPath) == "" {
		return errors.New("meta store_path and backup_path are required")
	}
	for _, address := range c.Gateway.APIHTTPAddrs {
		if strings.TrimSpace(address) == "" {
			return errors.New("gateway api_http_addrs must not contain empty addresses")
		}
	}
	return nil
}
