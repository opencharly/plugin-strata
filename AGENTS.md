# AGENTS.md — plugin-strata

The `charly strata` management CLI for a **deployed** Strata engine — an out-of-tree charly plugin
(`command:strata`). It talks plain HTTP to the engine, so it works identically compiled-in and
out-of-process, and it owns no server logic.

The authoritative rulebook is the umbrella `AGENTS.md` in `opencharly/opencharly` — Part I (R0–R10) and
Part II. Do not restate it here; this file adds only what is specific to this repo. `README.md` is the
user overview and never carries agent guidance.

## Load these skills first (R0)

- `/charly-internals:plugin` — the provider model, the `plugin:` block, the per-plugin CUE schema
  contract, external-vs-compiled placement. **Before reading or authoring any plugin source.**
- `/charly-internals:go` — before any Go source work; and `/charly-internals:go-quality` before a
  quality sweep. The plugin SDK is `github.com/opencharly/sdk`; the wire types are
  `github.com/opencharly/spec/proto`.
- `/charly-check:check` — the ADE plan and the disposable bed that gates a plugin.
- `/charly-internals:git-workflow` — before any git/PR action.

## What is unique here

- **External placement is deliberate and stated once**: `plugin.source` names this repo's own candy,
  so nothing is compiled into the `charly` binary and there is **no `compiled_plugins:` entry and no
  pin in `charly/charly.yml`**. Never write the placement into prose: the docs generator computes it.
- **Every route is verified against the engine's own router** (`serve/server.py` at the pinned tag),
  never inferred from a CLI's shape. `fetch()`'s tests and the route constants exist because an earlier
  draft called `/models` and `/logs`, which the engine **does not serve**; that draft compiled, vetted,
  formatted and unit-tested clean and would only have failed live, where a 404 from a management route
  reads as a *deployment* fault. Re-adding either route fails a test on purpose.
- **Read-only by design.** `config get` reads the engine's own published catalogue from `GET /config`
  rather than hard-coding a key list that would drift. There is deliberately **no `config set`**: at the
  pinned engine version a config save replaces the volume's run-config symlink with a file inside the
  image, so the volume never changes and the edit dies with the container (upstream #2547). And no
  `logs` leaf: the engine serves no logs route, and `charly logs <name>` already owns that surface — a
  second one would be a duplicate (R2).
- **`sdk.NewMeta` takes the plugin's own CalVer** — the parameter is named `calver` and every in-org
  plugin passes one; a semver is the wrong *kind* of string for it.
- The argument to `NewMeta` is the plugin's version, and the ADE plan's check is a **build-context
  marker**, mirroring `plugin-ollama` and `plugin-qdrant` exactly: a command plugin's own surface needs
  a live server, so exercising it belongs to the bed, not to the plan.

## Build / test

```sh
cd candy/plugin-strata
go build ./... && go vet ./... && gofmt -l . && go test ./...
```

Those four are the module CI in `.github/workflows/ci.yml`, and they are the gate for this half.
On a host with a capped or small root filesystem, point the Go caches at a roomy filesystem first
(`GOCACHE`, `GOTMPDIR`) — a cold Go build cache on a full root disk fails in ways that look like tool
bugs. The R10 gate for the plugin itself is a disposable bed composing a builtin **and** this external
plugin, driven to a fresh `charly update`; an external plugin is host-built at connect time, so that
bed's image needs `go`+`git`.
