package cli

import (
	"context"

	"zscalerctl/internal/config"
	"zscalerctl/internal/machine"
	machineruntime "zscalerctl/internal/runtime"
)

func inspectRuntimeStatus(
	ctx context.Context,
	cfg config.Config,
	opts globalOptions,
	operation machine.Operation,
) (machine.StatusResult, error) {
	inspector, err := machineruntime.NewStatusInspectorFromConfig(
		ctx,
		cfg,
		machineruntime.Options{Timeout: opts.timeout},
	)
	if err != nil {
		return machine.StatusResult{}, err
	}
	return inspector.Inspect(ctx, machine.StatusRequest{Operation: operation})
}
