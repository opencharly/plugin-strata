package strata

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fetch() is the one function every leaf funnels through, so its contract is the plugin's contract:
// it must send the key, ask for the leaf's own path, and hand the caller the status code rather than
// deciding for it. Before this test it was the only untested path in the module -- and the earlier
// draft's wrong routes (/models, /logs) show why an untested path here is dangerous: those compiled,
// vetted and unit-tested clean and would only have failed live, where a 404 from a management route
// reads as a deployment fault rather than a wrong path.
func TestFetchSendsTheKeyAndAsksForTheLeafPath(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath = r.Header.Get("Authorization"), r.URL.Path
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	t.Setenv("STRATA_API_KEY", "")
	t.Setenv("API_KEY", "k3y")

	cli := &CLI{Server: srv.URL}
	code, body, err := cli.fetch(pathStatus)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if code != http.StatusOK {
		t.Errorf("status: got %d, want 200", code)
	}
	if !strings.Contains(string(body), "ok") {
		t.Errorf("body not returned: %q", body)
	}
	if gotAuth != "Bearer k3y" {
		t.Errorf("Authorization: got %q, want %q", gotAuth, "Bearer k3y")
	}
	if gotPath != pathStatus {
		t.Errorf("path: got %q, want %q", gotPath, pathStatus)
	}
}

// A non-200 is RETURNED, not swallowed and not turned into an error here: the caller decides, which is
// what lets `get` report the status text while `config` still parses a 200 body. A plugin that raised on
// the first 401 would break the (real, expected) case of probing a server whose key is not set yet.
func TestFetchHandsBackANon200RatherThanDeciding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	cli := &CLI{Server: srv.URL}
	code, body, err := cli.fetch(pathModels)
	if err != nil {
		t.Fatalf("fetch must not error on a non-200: %v", err)
	}
	if code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", code)
	}
	if !strings.Contains(string(body), "unauthorized") {
		t.Errorf("the server's own text must survive: %q", body)
	}
}

// And a schemeless --server is normalised, because that is how an operator will type it.
func TestFetchNormalisesASchemelessServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cli := &CLI{Server: strings.TrimPrefix(srv.URL, "http://")}
	if code, _, err := cli.fetch(pathHealth); err != nil || code != http.StatusOK {
		t.Errorf("schemeless server: code=%d err=%v", code, err)
	}
}
