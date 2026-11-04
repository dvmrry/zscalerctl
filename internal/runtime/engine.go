package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"zscalerctl/internal/config"
	"zscalerctl/internal/machine"
	"zscalerctl/internal/resources"
	"zscalerctl/internal/zscaler"
)

// Engine is the common local operation coordinator. It owns host-supplied
// runtime construction options but loads config and constructs live readers
// per operation, matching the CLI's current credential lifetime.
type Engine struct {
	opts Options
}

// NewEngine constructs a coordinator without loading config, resolving a
// provider, constructing an SDK reader, or contacting Zscaler.
func NewEngine(opts Options) (*Engine, error) {
	opts.Env = append([]string(nil), opts.Env...)
	// Report resources are read only by the report command; dump, diff and
	// URL lookup never select them.
	opts.Catalog = withoutReportResources(catalogFromOptions(opts.Catalog))
	if err := resources.AssertReadOnly(opts.Catalog...); err != nil {
		return nil, fmt.Errorf("construct engine: catalog must be tenant read-only: %w", err)
	}
	return &Engine{opts: opts}, nil
}

// LookupURL validates one typed URL-classification batch before loading config
// or constructing a live reader, then returns only sanitized SDK-free values.
func (e *Engine) LookupURL(
	ctx context.Context,
	req machine.URLLookupRequest,
) (machine.URLLookupResult, error) {
	if e == nil {
		return machine.URLLookupResult{}, errors.New("engine runtime is nil")
	}
	ctx = nonNilContext(ctx)
	urls, err := prepareURLLookupRequest(ctx, req)
	if err != nil {
		return machine.URLLookupResult{}, err
	}
	lookup, err := NewURLLookup(ctx, e.options())
	if err != nil {
		return machine.URLLookupResult{}, err
	}
	return lookup.lookupPrepared(ctx, urls)
}

func (e *Engine) options() Options {
	opts := e.opts
	opts.Env = append([]string(nil), e.opts.Env...)
	opts.Catalog = copyCatalog(e.opts.Catalog)
	return opts
}

func resourceReadRuntimeBoundaryError(err error, req machine.ResourceReadRequest) error {
	if err == nil {
		return nil
	}
	if _, ok := errors.AsType[*machine.MachineError](err); ok {
		return err
	}
	machineErr := &machine.MachineError{
		Operation: req.Operation,
		Product:   strings.TrimSpace(req.Input.Product),
		Resource:  strings.TrimSpace(req.Input.Resource),
	}
	var sentinel error
	switch {
	case errors.Is(err, context.Canceled):
		machineErr.Kind = machine.ErrorKindCanceled
		machineErr.Message = "request canceled"
		sentinel = context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		machineErr.Kind = machine.ErrorKindDeadlineExceeded
		machineErr.Message = "request deadline exceeded"
		sentinel = context.DeadlineExceeded
	case errors.Is(err, config.ErrInvalidConfig):
		machineErr.Kind = machine.ErrorKindInvalidConfig
		machineErr.Message = "invalid configuration"
		sentinel = config.ErrInvalidConfig
	case errors.Is(err, zscaler.ErrInvalidProxyConfig):
		machineErr.Kind = machine.ErrorKindInvalidProxyConfig
		machineErr.Message = "invalid proxy configuration"
		sentinel = zscaler.ErrInvalidProxyConfig
	case errors.Is(err, zscaler.ErrInvalidSettings):
		machineErr.Kind = machine.ErrorKindInvalidConfig
		machineErr.Message, sentinel = sanitizedInvalidSettings(err)
	case errors.Is(err, zscaler.ErrMissingCredentials):
		machineErr.Kind = machine.ErrorKindMissingCredentials
		machineErr.Message, sentinel = sanitizedMissingCredentials(err)
		if missingErr, ok := errors.AsType[*zscaler.MissingCredentialsError](sentinel); ok {
			machineErr.Missing = append([]string(nil), missingErr.Missing...)
		}
	case errors.Is(err, zscaler.ErrUnsupportedResource):
		machineErr.Kind = machine.ErrorKindUnsupportedResource
		machineErr.Message = "unsupported resource"
		sentinel = zscaler.ErrUnsupportedResource
	case errors.Is(err, zscaler.ErrLiveAccessFailed):
		machineErr.Kind = machine.ErrorKindLiveAccessFailed
		machineErr.Message = "resource read failed"
		sentinel = zscaler.ErrLiveAccessFailed
	default:
		// Preserve unknown construction errors for callers.
		return err
	}
	machineErr.AttachFailure(err)
	return newBoundaryError(machineErr, sentinel)
}

func withoutReportResources(catalog resources.ResourceCatalog) resources.ResourceCatalog {
	out := make(resources.ResourceCatalog, 0, len(catalog))
	for _, spec := range catalog {
		if spec.EffectiveShape() != resources.ShapeReport {
			out = append(out, spec)
		}
	}
	return out
}
