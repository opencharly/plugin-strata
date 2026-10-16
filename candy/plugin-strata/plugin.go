// Package strata is the charly plugin OWNING the `charly strata` command — the host-side
// management CLI for a DEPLOYED Strata engine (status / models / config get / metrics / logs /
// version). The plugin owns the ENTIRE logic: the subcommand grammar AND the HTTP calls, over
// the engine's own management API. There is no core strata logic and no HostBuild seam.
//
// The command speaks plain HTTP to the engine, so it is fully self-contained and works
// IDENTICALLY compiled-in OR out-of-process (the same NewProvider/NewMeta/CliMain triple as
// every charly plugin): compiled-in the host dispatches Invoke(OpRun) in-process; out-of-process
// the host fork/execs cmd/serve -> CliMain running the SAME runStrataCLI.
//
// Endpoint resolution deliberately mirrors plugin-ollama: --server flag > the engine's own HOST
// and PORT variables (0.0.0.0 mapped to 127.0.0.1 for client use) > http://127.0.0.1:8080. A
// charly deployment auto-allocates the host port, so `charly status <name>` is the authority for
// what to pass. NOTE this is the SERVER ROOT: the management routes (/health, /status, /config,
// /metrics, /logs) live there, while the candy's provided STRATA_BASE_URL is the OpenAI-compatible
// API root under /v1 — a different surface.
package strata

import (
	"embed"
	"fmt"
	"os"

	"github.com/opencharly/sdk"
	pb "github.com/opencharly/spec/proto"
)

//go:embed schema/*.cue
var schemaFS embed.FS

// NewProvider returns the strata provider.
func NewProvider() pb.ProviderServer { return &provider{} }

// NewMeta advertises command:strata — the COMPILED-IN registry path resolves it
// (resolve(ClassCommand,"strata") -> Invoke(OpRun)) — plus this plugin's OWN self-contained CUE
// schema (schema/strata.cue) served over Describe: there is NO schema-less plugin.
func NewMeta() pb.PluginMetaServer {
	// The argument is the plugin's OWN CalVer, and the parameter is named `calver` for that reason --
	// every in-org plugin passes one ("2026.181.0001", "2026.243.0001", ...), and this file first shipped
	// a semver ("0.1.0"), which is the wrong KIND of string for the field. `2026.289.0001` is the shape
	// the org mints (YYYY.DDD.HHMM) with the `.0001` placeholder time several plugins already use, and
	// the day-of-year is this work's own date (2026-10-16 -> 289). The tag-on-merge process advances it.
	return sdk.NewMeta("2026.289.0001",
		[]sdk.ProvidedCapability{{Class: "command", Word: "strata"}},
		schemaFS)
}

// CliMain is the CLI entrypoint (the out-of-process placement AND the shared entry). The command
// is self-contained (plain HTTP), so it runs runStrataCLI directly — no reverse channel.
func CliMain(args []string) int {
	if err := runStrataCLI(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
