package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/phuslu/log"
	"gopkg.in/yaml.v3"
)

const DefaultConfigPath = "config.yaml"

type Config struct {
	Gateway  GatewayConfig  `yaml:"gateway"`
	Postgres PostgresConfig `yaml:"postgres"`
	Redis    RedisConfig    `yaml:"redis"`
}

func New(path ...string) *Config {
	p := DefaultConfigPath
	if len(path) > 0 {
		p = path[0]
	}
	config, err := Load(p)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
	return config
}

func NewTest() *Config {
	return &Config{
		Gateway: GatewayConfig{
			MaxTimeout: 10,
		},
		Postgres: PostgresConfig{
			DSN: "host=localhost port=5433 user=myuser password=mypassword dbname=database sslmode=disable",
		},
		Redis: RedisConfig{
			Addr:     "localhost:4999",
			DB:       0,
			Password: "123456",
		},
	}
}

type GatewayConfig struct {
	MaxTimeout int `yaml:"max_timeout"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type PostgresConfig struct {
	DSN string `yaml:"dsn"`
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.Default()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Default() {
	if c.Gateway.MaxTimeout == 0 {
		c.Gateway.MaxTimeout = 10
	}
	if v := os.Getenv("GATEWAY_MAX_TIMEOUT"); v != "" {
		if timeout, err := strconv.Atoi(v); err == nil {
			c.Gateway.MaxTimeout = timeout
		}
	}
	if v := os.Getenv("POSTGRES_DSN"); v != "" {
		c.Postgres.DSN = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		c.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		c.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if db, err := strconv.Atoi(v); err == nil {
			c.Redis.DB = db
		}
	}
}

// Validate 校验关键配置,避免服务启动后再暴露问题。
func (c *Config) Validate() error {
	if c.Postgres.DSN == "" {
		return errors.New("postgres dsn is required")
	}
	if c.Redis.Addr == "" {
		return errors.New("redis addr is required")
	}
	if c.Redis.DB < 0 {
		return errors.ErrUnsupported
	}
	if c.Redis.Password == "" {
		return errors.New("redis password is required")
	}
	return nil
}
