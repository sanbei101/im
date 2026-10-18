package config

import "testing"

func TestValidateHTTPTopology(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		if err := NewTest().Validate(); err != nil {
			t.Fatalf("valid config rejected: %v", err)
		}
	})

	t.Run("missing gateway api addresses", func(t *testing.T) {
		cfg := NewTest()
		cfg.Gateway.APIHTTPAddrs = nil
		if err := cfg.Validate(); err == nil {
			t.Fatal("config without gateway api_http_addrs was accepted")
		}
	})

	t.Run("missing postgres url", func(t *testing.T) {
		cfg := NewTest()
		cfg.Postgres.URL = ""
		if err := cfg.Validate(); err == nil {
			t.Fatal("config without postgres url was accepted")
		}
	})
}
