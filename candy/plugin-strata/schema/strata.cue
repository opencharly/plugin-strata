// plugin-strata's OWN self-contained CUE schema — the SINGLE SOURCE for this plugin's served
// declaration surface (there is no schema-less plugin: every plugin ships a non-empty schema over
// Describe).
//
// SELF-CONTAINED and PACKAGE-LESS: it references no base def and carries no package clause, so it
// compiles STANDALONE — the property the SDK's serve-side compile needs, and the property that lets
// the host splice `base ++ plugin` at the load gate. A self-contained schema that will not splice is
// a LOUD load failure.
//
// NO GO CONSUMER: the plugin declares no typed `plugin_input` (its authored input is its
// pass-through CLI grammar), so this schema generates NO `params` package and has NO
// `cue exp gengotypes` artifact — it is the SERVED documentation/config surface, not a
// code-generation source.
//
// It DOCUMENTS the `command: strata` contract, the endpoint resolution the CLI reads, and the
// read-only boundary that is a deliberate property of this plugin rather than an omission.
#StrataPlugin: {
	// The command word the plugin serves.
	command: "strata"

	// What the command does, in one line (the public-docs surface).
	contract: string & !=""

	// Endpoint resolution: --server flag > HOST and PORT (a container bind address is mapped to
	// loopback for client use) > this default. NOTE this is the SERVER ROOT: /health, /status,
	// /config, /metrics and /logs live here, while the candy's provided STRATA_BASE_URL is the
	// OpenAI-compatible API root under /v1 — a different surface.
	default_endpoint: "http://127.0.0.1:8080"

	// The management routes this plugin reads, each one verified against the engine's own router
	// (serve/server.py) at the pinned tag: /health, /status, /v1/models (there is NO management
	// /models), /props, /config, /metrics. There is no /logs route at all, so viewing a deployment's
	// output stays with `charly logs <name>` rather than becoming a second surface (R2).
	//
	// All GET: the plugin exposes no mutation. A config
	// SAVE is deliberately absent because at the pinned engine version it replaces the volume's
	// run-config symlink with a file inside the image, losing the edit when the container is
	// recreated (upstream #2547). Non-runtime-editable configuration belongs in the deployment's
	// `env:` — that is the `strata` layer's design.
	routes: [string]: =~"^GET /"
}
