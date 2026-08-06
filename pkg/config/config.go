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
	Auth     AuthConfig     `yaml:"auth"`
	API      APIConfig      `yaml:"api"`
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
		Auth: AuthConfig{JWTSecret: "test-jwt-secret"},
		API:  APIConfig{AllowedOrigins: []string{"http://localhost:3000"}},
		Gateway: GatewayConfig{
			MaxTimeout:     10,
			MaxFrameBytes:  1 << 20,
			AllowedOrigins: []string{"http://localhost:3000"},
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

type AuthConfig struct {
	JWTSecret string `yaml:"jwt_secret"`
}

type APIConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

type GatewayConfig struct {
	MaxTimeout     int      `yaml:"max_timeout"`
	MaxFrameBytes  int      `yaml:"max_frame_bytes"`
	AllowedOrigins []string `yaml:"allowed_origins"`
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
	if len(c.API.AllowedOrigins) == 0 {
		c.API.AllowedOrigins = []string{"http://localhost:3000"}
	}
	if len(c.Gateway.AllowedOrigins) == 0 {
		c.Gateway.AllowedOrigins = append([]string(nil), c.API.AllowedOrigins...)
	}
	if c.Gateway.MaxFrameBytes == 0 {
		c.Gateway.MaxFrameBytes = 1 << 20
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		c.Auth.JWTSecret = v
	}
	if c.Gateway.MaxTimeout == 0 {
		c.Gateway.MaxTimeout = 10
	}
	if v := os.Getenv("GATEWAY_MAX_TIMEOUT"); v != "" {
		if timeout, err := strconv.Atoi(v); err == nil {
			c.Gateway.MaxTimeout = timeout
		}
	}
	if v := os.Getenv("GATEWAY_MAX_FRAME_BYTES"); v != "" {
		if size, err := strconv.Atoi(v); err == nil {
			c.Gateway.MaxFrameBytes = size
		}
	}
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		origins := strings.Split(v, ",")
		c.API.AllowedOrigins = origins
		c.Gateway.AllowedOrigins = append([]string(nil), origins...)
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
	if c.Auth.JWTSecret == "" {
		return errors.New("jwt secret is required")
	}
	if c.Gateway.MaxFrameBytes <= 0 {
		return errors.New("gateway max frame bytes must be positive")
	}
	for _, origin := range append(c.API.AllowedOrigins, c.Gateway.AllowedOrigins...) {
		if origin == "" || origin == "*" || strings.Contains(origin, "*") {
			return errors.New("wildcard CORS origins are not allowed")
		}
	}
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
