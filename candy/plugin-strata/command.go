package strata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kong"
)

// command.go — the `charly strata` grammar and the HTTP calls behind it. One `GET` per leaf: the
// engine's management API is read-only for everything this plugin exposes, so there is no state
// here and no reverse channel.
//
// `config get` is the only config surface, BY DESIGN: at the pinned engine version a config SAVE
// replaces the run-config symlink inside the image with a regular file, so the volume's file never
// changes and the edit dies with the container (upstream #2547, still open — verified against the
// pin). Nothing here may offer a write that would silently lose the operator's change.

// The engine's management routes, as CONSTANTS rather than string literals at each call site, so
// they can be asserted by a test. Each one was verified against the engine's own router
// (serve/server.py) at the pinned tag; `models` and `props` are the two that are easy to get wrong
// (there is no management `/models`), and there is no `/logs` route at all.
const (
	pathHealth  = "/health"
	pathStatus  = "/status"
	pathModels  = "/v1/models"
	pathProps   = "/props"
	pathConfig  = "/config"
	pathMetrics = "/metrics"
)

// CLI is the `charly strata` command tree. --server is accepted on the root and on every leaf.
type CLI struct {
	Server string `help:"Strata server root, e.g. http://127.0.0.1:8080 (default: $HOST:$PORT, else http://127.0.0.1:8080)."`

	Version VersionCmd `cmd:"" help:"Print the engine version the server reports."`
	Status  StatusCmd  `cmd:"" help:"Show the server status document."`
	Models  ModelsCmd  `cmd:"" help:"List the models the server serves (the OpenAI-compatible list)."`
	Props   PropsCmd   `cmd:"" help:"Show the loaded model's properties."`
	Config  ConfigCmd  `cmd:"" help:"Read the engine's runtime key catalogue. Read-only by design."`
	Metrics MetricsCmd `cmd:"" help:"Fetch the engine metrics document."`
}

// NOTE -- no `logs` leaf, deliberately. The engine serves NO logs route (verified against
// serve/server.py at the pinned tag: /config, /status, /metrics, /props, /slots, /load, /unload,
// /api-monitor, /api/requests, /settings, /mcp exist; /logs does not), and `charly logs <name>`
// already owns viewing a deployment's output. A second way to read the same thing would be a
// duplicate surface (R2).
//
// Model listing is a case where the engine's own naming matters: there is no management `/models`,
// so this reads the OpenAI-compatible `/v1/models` -- the same list a client sees.

// VersionCmd prints the version the ENGINE reports, not this plugin's.
type VersionCmd struct{}

// StatusCmd shows the server status document.
type StatusCmd struct{}

// ModelsCmd lists configured and loaded models.
type ModelsCmd struct{}

// ConfigCmd reads the key catalogue. With an argument it prints one key.
type ConfigCmd struct {
	Key string `arg:"" optional:"" help:"Print only this key from the catalogue."`
}

// PropsCmd shows the loaded model's properties.
type PropsCmd struct{}

// MetricsCmd fetches the metrics document.
type MetricsCmd struct{}

func (VersionCmd) Run(cli *CLI) error { return cli.get(pathHealth) }
func (StatusCmd) Run(cli *CLI) error  { return cli.get(pathStatus) }
func (ModelsCmd) Run(cli *CLI) error  { return cli.get(pathModels) }
func (PropsCmd) Run(cli *CLI) error   { return cli.get(pathProps) }
func (MetricsCmd) Run(cli *CLI) error { return cli.get(pathMetrics) }

// Run reads the catalogue and either prints it whole or extracts one key. The catalogue is the
// engine's OWN published list, so this plugin never hard-codes a key name that could drift from
// the engine.
func (c ConfigCmd) Run(cli *CLI) error {
	if c.Key == "" {
		return cli.get(pathConfig)
	}
	code, body, err := cli.fetch(pathConfig)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("GET /config: %s: %s", http.StatusText(code), strings.TrimSpace(string(body)))
	}
	var doc struct {
		Keys []struct {
			Key   string `json:"key"`
			Value any    `json:"value"`
			Help  string `json:"help"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("decode /config: %w", err)
	}
	for _, k := range doc.Keys {
		if k.Key == c.Key {
			fmt.Printf("%s = %v\n", k.Key, k.Value)
			return nil
		}
	}
	return fmt.Errorf("the engine publishes no key %q (the catalogue is authoritative; `charly strata config` lists it)", c.Key)
}

// get prints a management document, pretty-printed when it is JSON.
func (cli *CLI) get(path string) error {
	code, body, err := cli.fetch(path)
	if err != nil {
		return err
	}
	var pretty any
	if json.Unmarshal(body, &pretty) == nil {
		out, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Print(string(body))
	}
	if code != http.StatusOK {
		return fmt.Errorf("GET %s: %s", path, http.StatusText(code))
	}
	return nil
}

// fetch performs one authenticated GET. The API key is read from API_KEY or the candy's
// STRATA_API_KEY name, so the plugin works with a deployment configured either way.
func (cli *CLI) fetch(path string) (int, []byte, error) {
	url := strings.TrimSuffix(resolveServer(cli.Server), "/") + path
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}
	if key := apiKey(); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return 0, nil, fmt.Errorf("GET %s: read body: %w", url, err)
	}
	return resp.StatusCode, body, nil
}

func apiKey() string {
	if k := strings.TrimSpace(os.Getenv("API_KEY")); k != "" {
		return k
	}
	return strings.TrimSpace(os.Getenv("STRATA_API_KEY"))
}

// resolveServer mirrors plugin-ollama's resolution rule: the flag wins, then the engine's own HOST
// and PORT variables (with the container bind address mapped to loopback for client use), then the
// documented default. A charly deployment auto-allocates the host port, so `charly status <name>`
// is the authority on what to pass here.
func resolveServer(flag string) string {
	if s := strings.TrimSpace(flag); s != "" {
		return withScheme(s)
	}
	host := strings.TrimSpace(os.Getenv("HOST"))
	if host == "" {
		return "http://127.0.0.1:8080"
	}
	if host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	return "http://" + host + ":" + port
}

// withScheme prepends http:// to a schemeless value so `--server 127.0.0.1:8080` works.
func withScheme(s string) string {
	if strings.Contains(s, "://") {
		return s
	}
	return "http://" + s
}

// runStrataCLI parses the gramar and runs the selected leaf.
func runStrataCLI(args []string) error {
	cli := &CLI{}
	parser, err := kong.New(cli,
		kong.Name("strata"),
		kong.Description("Manage a deployed Strata engine (Qwen3.8-Flash-Next) from the host."),
		kong.UsageOnError(),
	)
	if err != nil {
		return err
	}
	ctx, err := parser.Parse(args)
	if err != nil {
		return err
	}
	return ctx.Run(cli)
}
