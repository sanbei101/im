package config

import "testing"

func TestValidateHTTPTopology(t *testing.T) {
	cfg := NewTest()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	cfg.API.HTTPAddrs = nil
	if err := cfg.Validate(); err == nil {
		t.Fatal("config without api http_addrs was accepted")
	}

	cfg = NewTest()
	cfg.Gateway.APIHTTPAddrs = nil
	if err := cfg.Validate(); err == nil {
		t.Fatal("config without gateway api_http_addrs was accepted")
	}

	cfg = NewTest()
	cfg.Gateway.MetaHTTPAddr = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("config without gateway meta_http_addr was accepted")
	}
}
