package tests

import (
	"strings"
	"testing"

	appPkg "github.com/aptlogica/sereni-base/internal/app"
)

func TestAppNew_RefusesOpenPublicAPIInProduction(t *testing.T) {
	for _, env := range []string{"prod", "production"} {
		cfg := validConfig()
		cfg.Server.Env = env
		cfg.PublicAPI.OpenAccess = true

		appInstance, err := appPkg.New(cfg)
		if err == nil || !strings.Contains(err.Error(), "PUBLIC_API_OPEN_ACCESS") {
			t.Fatalf("%s: expected the open public API to be refused, got %v", env, err)
		}
		if appInstance != nil {
			t.Fatalf("%s: expected no app", env)
		}
	}
}

func TestAppNew_OpenPublicAPIOutsideProduction(t *testing.T) {
	cleanup := withFakeDatabase(t)
	defer cleanup()

	cfg := validConfig()
	cfg.PublicAPI.OpenAccess = true
	cfg.PublicAPI.RateLimitPerMinute = 600

	appInstance, err := appPkg.New(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	found := false
	for _, route := range appInstance.Router().Routes() {
		if route.Path == "/api/v1/:ws/bases/:base/tables/:table/records" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected the public API routes to be registered")
	}
}
