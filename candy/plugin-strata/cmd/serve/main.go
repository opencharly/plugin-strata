// Command serve is the OUT-OF-PROCESS entrypoint for the strata command plugin: dual-mode
// sdk.Main (serve OR CLI). charly fork/execs this binary in CLI mode for command:strata dispatch
// when the plugin is NOT compiled-in (-> CliMain); the serve half backs the out-of-process
// provider placement. The SAME NewProvider()/NewMeta() compile INTO charly in-process when listed
// in compiled_plugins — placement is invisible.
package main

import (
	strata "github.com/opencharly/plugin-strata/candy/plugin-strata"
	"github.com/opencharly/sdk"
)

func main() { sdk.Main(strata.NewProvider(), strata.NewMeta(), strata.CliMain) }
