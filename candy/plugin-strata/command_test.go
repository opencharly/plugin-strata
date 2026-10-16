package strata

import (
	"os"
	"testing"
)

// The endpoint-resolution rule is the one piece of this plugin that is pure enough to prove without
// a running engine, and it is the piece every command depends on — so it gets a test that FAILS
// without the behaviour: the flag wins, a schemeless value gains a scheme, and the engine's own
// container bind address is mapped to loopback rather than dialled.
func TestResolveServer(t *testing.T) {
	t.Setenv("HOST", "")
	t.Setenv("PORT", "")

	if got := resolveServer(""); got != "http://127.0.0.1:8080" {
		t.Errorf("default: got %q", got)
	}
	if got := resolveServer("127.0.0.1:9999"); got != "http://127.0.0.1:9999" {
		t.Errorf("schemeless flag: got %q", got)
	}
	if got := resolveServer("https://strata.example/v1"); got != "https://strata.example/v1" {
		t.Errorf("flag with scheme must pass through unchanged: got %q", got)
	}
	// The flag wins over the environment, always.
	if err := os.Setenv("HOST", "0.0.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := os.Setenv("PORT", "18080"); err != nil {
		t.Fatal(err)
	}
	if got := resolveServer("127.0.0.1:9999"); got != "http://127.0.0.1:9999" {
		t.Errorf("flag must win over env: got %q", got)
	}
	// With no flag, the engine's bind address is mapped for client use: 0.0.0.0 is not dialable.
	if got := resolveServer(""); got != "http://127.0.0.1:18080" {
		t.Errorf("0.0.0.0 must map to loopback: got %q", got)
	}
}

// The API key is read under either name, because a charly deployment supplies the candy's
// STRATA_API_KEY while the engine's own contract reads API_KEY.
func TestAPIKeyPrefersEngineName(t *testing.T) {
	t.Setenv("API_KEY", "")
	t.Setenv("STRATA_API_KEY", "from-candy")
	if got := apiKey(); got != "from-candy" {
		t.Errorf("got %q", got)
	}
	t.Setenv("API_KEY", "from-engine")
	if got := apiKey(); got != "from-engine" {
		t.Errorf("API_KEY must win: got %q", got)
	}
}

// The route table is the part of this plugin that a green build CANNOT protect: a wrong path compiles,
// vets, formats and unit-tests clean, and only fails at a live bed -- where a 404 from a management route
// looks like a deployment fault rather than a wrong path. So the paths are asserted here, against the
// engine's own router at the pinned tag. This test FAILS if someone re-introduces a route the engine does
// not serve.
func TestManagementRoutesAreTheVerifiedOnes(t *testing.T) {
	verified := map[string]string{
		"health":  pathHealth,
		"status":  pathStatus,
		"models":  pathModels,
		"props":   pathProps,
		"config":  pathConfig,
		"metrics": pathMetrics,
	}
	want := map[string]string{
		"health":  "/health",
		"status":  "/status",
		"models":  "/v1/models", // NOT /models: the engine serves no management models route
		"props":   "/props",
		"config":  "/config",
		"metrics": "/metrics",
	}
	for leaf, path := range want {
		if got := verified[leaf]; got != path {
			t.Errorf("%s: route is %q, want %q", leaf, got, path)
		}
	}
	// The two routes the engine does NOT serve. /logs has no route at all (charly logs owns that
	// surface); /models is only real under the /v1 API prefix. Named explicitly so that re-adding
	// either fails here rather than at a bed.
	for _, forbidden := range []string{"/models", "/logs"} {
		for leaf, path := range verified {
			if path == forbidden {
				t.Errorf("%s must not use %q: the engine serves no such route", leaf, forbidden)
			}
		}
	}
}
