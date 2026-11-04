package machine

import (
	"context"
	"errors"
	"fmt"
)

// Failure reasons are a closed, value-free vocabulary describing why a live
// read failed. They let a caller choose between retrying, fixing the API role,
// or giving up, without exposing the backend message (which can carry tenant
// values). This package owns the vocabulary: the live reader classifies into
// it, and the CLI error envelope and dump error records carry it. A test keeps
// the published schemas equal to FailureReasons.
const (
	// FailureReasonCredentialsRejected is a sign-in the API refused (HTTP
	// 400, 401 or 403 from the token or session endpoint): the configured
	// credentials or account need fixing, unlike unauthorized or forbidden on
	// a read, which concern the role.
	FailureReasonCredentialsRejected  = "credentials_rejected"
	FailureReasonUnauthorized         = "unauthorized"
	FailureReasonForbidden            = "forbidden"
	FailureReasonNotFound             = "not_found"
	FailureReasonRateLimited          = "rate_limited"
	FailureReasonServerError          = "server_error"
	FailureReasonClientError          = "client_error"
	FailureReasonTransport            = "transport"
	FailureReasonTimeout              = "timeout"
	FailureReasonPaginationIncomplete = "pagination_incomplete"
	FailureReasonDecodeFailed         = "decode_failed"
	FailureReasonNotSubscribed        = "not_subscribed"
	FailureReasonUnknown              = "unknown"
)

// FailureReasons returns the whole vocabulary.
func FailureReasons() []string {
	return []string{
		FailureReasonCredentialsRejected, FailureReasonUnauthorized, FailureReasonForbidden, FailureReasonNotFound, FailureReasonRateLimited,
		FailureReasonServerError, FailureReasonClientError, FailureReasonTransport, FailureReasonTimeout,
		FailureReasonPaginationIncomplete, FailureReasonDecodeFailed, FailureReasonNotSubscribed, FailureReasonUnknown,
	}
}

// FailureDetail says why a live read failed in value-free terms: a reason from
// a closed vocabulary, the HTTP status when there was a response, and whether
// a retry could plausibly succeed. The zero value means "not known".
type FailureDetail struct {
	Reason    string
	Status    int
	Retryable bool
	// Note names the check behind a pagination_incomplete failure, in counts
	// and offsets only. It is shown in the message, not as its own field.
	Note string
}

// failureDetailer is implemented by live-read errors that classified their own
// failure. It is an interface so this package does not depend on the reader.
type failureDetailer interface {
	FailureDetail() (reason string, status int, retryable bool)
}

// failureNoter is implemented by live-read errors that can name the check
// behind their failure.
type failureNoter interface {
	FailureNote() string
}

// FailureDetailOf extracts the failure detail from an error chain. A deadline
// is a timeout even when the reader wrapped it, because a longer --timeout or
// a retry can succeed.
func FailureDetailOf(err error) FailureDetail {
	if err == nil {
		return FailureDetail{}
	}
	var machineErr *MachineError
	if errors.As(err, &machineErr) && machineErr.Detail.Reason != "" {
		return machineErr.Detail
	}
	var machineValue MachineError
	if errors.As(err, &machineValue) && machineValue.Detail.Reason != "" {
		return machineValue.Detail
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return FailureDetail{Reason: FailureReasonTimeout, Retryable: true}
	}
	var detailer failureDetailer
	if errors.As(err, &detailer) {
		reason, status, retryable := detailer.FailureDetail()
		detail := FailureDetail{Reason: reason, Status: status, Retryable: retryable}
		if noter, ok := detailer.(failureNoter); ok {
			detail.Note = noter.FailureNote()
		}
		return detail
	}
	return FailureDetail{}
}

// AttachFailure records why a live read failed on a machine error built from
// it. Every site that turns a reader error into a live_access_failed or
// deadline_exceeded MachineError calls this while it still holds the original
// error, because the error it builds keeps only a sanitized sentinel cause.
// Other kinds are left unchanged.
func (e *MachineError) AttachFailure(err error) {
	if e.Kind != ErrorKindLiveAccessFailed && e.Kind != ErrorKindDeadlineExceeded {
		return
	}
	if e.Detail.Reason != "" {
		return
	}
	e.Detail = FailureDetailOf(err)
	if e.Kind == ErrorKindLiveAccessFailed {
		e.Message = e.Detail.Describe(e.Message)
	}
}

// Describe appends the reason, status and note to a fixed message, for
// example "resource read failed: forbidden (HTTP 403)".
func (d FailureDetail) Describe(message string) string {
	switch {
	case d.Reason == "":
		return message
	case d.Status > 0:
		message = fmt.Sprintf("%s: %s (HTTP %d)", message, d.Reason, d.Status)
	default:
		message = fmt.Sprintf("%s: %s", message, d.Reason)
	}
	if d.Note != "" {
		message += ": " + d.Note
	}
	return message
}
