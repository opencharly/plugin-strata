package strata

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opencharly/sdk"
	pb "github.com/opencharly/spec/proto"
)

// provider.go — the Invoke(OpRun) surface for the COMPILED-IN command:strata placement: the host's
// command dispatch invokes this in-process with the pass-through args. The command is
// SELF-CONTAINED (plain HTTP to the engine), so it uses NO reverse channel; Invoke and CliMain both
// just run runStrataCLI.

type provider struct{ pb.UnimplementedProviderServer }

// Invoke runs `charly strata` in-process for the compiled-in placement: it decodes the
// pass-through args and runs the strata logic directly. It RETURNS the error so a non-zero exit
// propagates to the caller.
func (provider) Invoke(_ context.Context, req *pb.InvokeRequest) (*pb.InvokeReply, error) {
	if req.GetOp() != sdk.OpRun {
		return nil, fmt.Errorf("plugin-strata: unsupported op %q (want %q)", req.GetOp(), sdk.OpRun)
	}
	var in struct {
		Args []string `json:"args"`
	}
	if len(req.GetParamsJson()) > 0 {
		if err := json.Unmarshal(req.GetParamsJson(), &in); err != nil {
			return nil, fmt.Errorf("plugin-strata: decode args: %w", err)
		}
	}
	if err := runStrataCLI(in.Args); err != nil {
		return nil, err
	}
	return &pb.InvokeReply{}, nil
}
