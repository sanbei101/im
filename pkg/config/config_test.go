package config

import "testing"

func TestValidateHTTPTopology(t *testing.T) {
	cfg := NewTest()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	cfg = NewTest()
	cfg.Gateway.APIHTTPAddrs = nil
	if err := cfg.Validate(); err == nil {
		t.Fatal("config without gateway api_http_addrs was accepted")
	}

	cfg = NewTest()
	cfg.Meta.HTTPAddr = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("config without meta http_addr was accepted")
	}
}
