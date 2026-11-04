package machine_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"zscalerctl/internal/machine"
	"zscalerctl/internal/resources"
)

type classifiedFailure struct {
	reason    string
	status    int
	retryable bool
}

func (f classifiedFailure) Error() string { return "classified failure" }
func (f classifiedFailure) FailureDetail() (string, int, bool) {
	return f.reason, f.status, f.retryable
}

type notedFailure struct{ classifiedFailure }

func (notedFailure) FailureNote() string {
	return "zia pagination received repeated page content at page 2"
}

func TestFailureDetailOf(t *testing.T) {
	t.Parallel()

	noted := fmt.Errorf("read: %w", notedFailure{classifiedFailure{reason: "pagination_incomplete"}})
	if got := machine.FailureDetailOf(noted); got.Note != "zia pagination received repeated page content at page 2" {
		t.Errorf("FailureDetailOf(noted).Note = %q, want the walker check", got.Note)
	}

	wrapped := fmt.Errorf("read: %w", classifiedFailure{"forbidden", 403, false})
	if got := machine.FailureDetailOf(wrapped); got != (machine.FailureDetail{Reason: "forbidden", Status: 403}) {
		t.Errorf("FailureDetailOf(wrapped classified) = %+v", got)
	}
	deadline := fmt.Errorf("zscaler list zia/users timed out: %w", context.DeadlineExceeded)
	if got := machine.FailureDetailOf(deadline); got != (machine.FailureDetail{Reason: machine.FailureReasonTimeout, Retryable: true}) {
		t.Errorf("FailureDetailOf(deadline) = %+v, want retryable timeout", got)
	}
	if got := machine.FailureDetailOf(errors.New("plain")); got != (machine.FailureDetail{}) {
		t.Errorf("FailureDetailOf(plain) = %+v, want zero", got)
	}
	for _, tc := range []struct {
		detail machine.FailureDetail
		want   string
	}{
		{machine.FailureDetail{Reason: "forbidden", Status: 403}, "resource read failed: forbidden (HTTP 403)"},
		{machine.FailureDetail{Reason: "pagination_incomplete"}, "resource read failed: pagination_incomplete"},
		{machine.FailureDetail{Reason: "pagination_incomplete", Note: "zcc pagination ended at 3 records, but totalCount declares 5"}, "resource read failed: pagination_incomplete: zcc pagination ended at 3 records, but totalCount declares 5"},
		{machine.FailureDetail{Reason: "client_error", Status: 400}, "resource read failed: client_error (HTTP 400)"},
		{machine.FailureDetail{}, "resource read failed"},
	} {
		if got := tc.detail.Describe("resource read failed"); got != tc.want {
			t.Errorf("Describe(%+v) = %q, want %q", tc.detail, got, tc.want)
		}
	}
}

// TestExecutorLiveFailureCarriesDetail keeps the cause of a failed read on the
// machine error: the message names the reason and status, and Detail carries
// them for the CLI envelope, instead of a bare "resource read failed".
func TestExecutorLiveFailureCarriesDetail(t *testing.T) {
	t.Parallel()

	loader := &fakeBrowserLoader{err: fmt.Errorf("load: %w", classifiedFailure{"rate_limited", 429, true})}
	executor := machine.Executor{
		Browser: loader,
		Catalog: resources.ResourceCatalog{
			testExecutorSpec(resources.ProductZIA, "locations", resources.ReadOperations(), "id"),
		},
	}
	_, err := executor.Execute(context.Background(), machine.Request{
		RequestID:  "req-live-failure",
		Capability: machine.CapabilityResourcesRead,
		Operation:  machine.OperationList,
		Input:      &machine.Input{Product: "zia", Resource: "locations"},
	})
	var machineErr *machine.MachineError
	if !errors.As(err, &machineErr) {
		t.Fatalf("Execute error = %v, want MachineError", err)
	}
	if machineErr.Kind != machine.ErrorKindLiveAccessFailed {
		t.Errorf("Kind = %q, want %q", machineErr.Kind, machine.ErrorKindLiveAccessFailed)
	}
	if want := (machine.FailureDetail{Reason: "rate_limited", Status: 429, Retryable: true}); machineErr.Detail != want {
		t.Errorf("Detail = %+v, want %+v", machineErr.Detail, want)
	}
	if want := "resource read failed: rate_limited (HTTP 429)"; machineErr.Message != want {
		t.Errorf("Message = %q, want %q", machineErr.Message, want)
	}
}

func TestFailureDetailOfReadsMachineErrorDetail(t *testing.T) {
	t.Parallel()

	// Production code returns *MachineError; typed as error so vet accepts
	// wrapping a pointer to a value-receiver error.
	var cause error = &machine.MachineError{
		Kind:   machine.ErrorKindLiveAccessFailed,
		Detail: machine.FailureDetail{Reason: "server_error", Status: 502, Retryable: true},
	}
	err := fmt.Errorf("boundary: %w", cause)
	if got := machine.FailureDetailOf(err); got != (machine.FailureDetail{Reason: "server_error", Status: 502, Retryable: true}) {
		t.Errorf("FailureDetailOf(MachineError) = %+v", got)
	}
}

func TestFailureDetailOfReadsWrappedMachineErrorValue(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("boundary: %w", machine.MachineError{
		Kind:   machine.ErrorKindLiveAccessFailed,
		Detail: machine.FailureDetail{Reason: "forbidden", Status: 403},
	})
	if got := machine.FailureDetailOf(err); got != (machine.FailureDetail{Reason: "forbidden", Status: 403}) {
		t.Errorf("FailureDetailOf(MachineError value) = %+v", got)
	}
}
