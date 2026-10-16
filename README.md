# plugin-strata

The `charly strata` management CLI — inspect a deployed **Strata** engine (Qwen3.8-Flash-Next) from the
host, over the same HTTP API your clients use.

```sh
charly strata version   # /health: model, max_context, and whether a key is set (pre-auth, no secret needed)
charly strata status    # /status
charly strata models    # /v1/models — the OpenAI-compatible list
charly strata props     # /props — the loaded model's properties
charly strata config    # the engine's own catalogue of runtime keys
charly strata config parallel   # one key's current value
charly strata metrics   # /metrics
```

## Endpoint resolution

`--server <url>` > the engine's own `HOST`/`PORT` (a container bind address is mapped to `127.0.0.1`
for client use) > `http://127.0.0.1:8080`. A schemeless value gets `http://` prepended. A charly
deployment auto-allocates the host port, so `charly status <name>` is the authority on what to pass.

The key is read from `API_KEY` (the engine's own name) or `STRATA_API_KEY` (the name a charly
deployment supplies).

## Read-only by design

There is no `config set` and no `logs`. A config *save* at the pinned engine version replaces the
volume's run-config symlink with a file inside the image, so the edit would be silently lost when the
container is recreated (upstream #2547) — configuration that is not runtime-editable belongs in the
deployment's `env:`. And the engine serves no logs route, so `charly logs <name>` owns that.

## Install

Out-of-tree: charly connects this plugin by word at runtime, so a project composes it as
`@github.com/opencharly/plugin-strata/candy/plugin-strata:<tag>` — nothing is built into the charly
binary.

## Licence

MIT, like the engine it manages. The model weights are separate and carry their own licences.
