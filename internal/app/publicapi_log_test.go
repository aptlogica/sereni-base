package app

import (
	"context"
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/config"
	"github.com/aptlogica/sereni-base/internal/platform"
)

func logTestConfig(mutate func(*config.PublicAPIConfig)) *config.Config {
	cfg := &config.Config{}
	cfg.PublicAPI = config.PublicAPIConfig{LogMode: "all", LogBodies: "errors", LogMaxBodyBytes: 8192,
		LogQueueSize: 10, LogBatchSize: 10, LogFlushSeconds: 1}
	if mutate != nil {
		mutate(&cfg.PublicAPI)
	}
	return cfg
}

func TestConfigurePublicAPILog(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*config.PublicAPIConfig)
		wantErr  bool
		wantShip bool
	}{
		{"all, no log service", nil, false, false},
		{"empty mode is all", func(c *config.PublicAPIConfig) { c.LogMode = "" }, false, false},
		{"errors", func(c *config.PublicAPIConfig) { c.LogMode = "Errors" }, false, false},
		{"off ships nothing even with a url", func(c *config.PublicAPIConfig) { c.LogMode = "off"; c.LogServiceURL = "http://x/api/v1" }, false, false},
		{"bad mode", func(c *config.PublicAPIConfig) { c.LogMode = "some" }, true, false},
		{"with log service", func(c *config.PublicAPIConfig) { c.LogServiceURL = "http://127.0.0.1:1/api/v1" }, false, true},
		{"bad bodies level", func(c *config.PublicAPIConfig) { c.LogServiceURL = "http://x/api/v1"; c.LogBodies = "some" }, true, false},
		{"bodies level ignored without log service", func(c *config.PublicAPIConfig) { c.LogBodies = "some" }, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api := platform.NewAPI(nil, "s", nil, 0)
			shipper, err := configurePublicAPILog(api, logTestConfig(tc.mutate))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v", err)
			}
			if (shipper != nil) != tc.wantShip {
				t.Fatalf("shipper = %v, want shipper: %v", shipper, tc.wantShip)
			}
			if shipper != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = shipper.Shutdown(ctx)
			}
		})
	}
}

func TestSplitList(t *testing.T) {
	got := splitList(" a, b ,,c ")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("got %v", got)
	}
	if splitList("") != nil {
		t.Error("empty must give nil")
	}
}
