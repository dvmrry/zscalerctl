package zscaler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	zsdk "github.com/zscaler/zscaler-sdk-go/v3/zscaler"
)

const (
	zpaPageSize = 500
	zpaMaxPages = 1000

	zpaAPIV1 = "v1"
	zpaAPIV2 = "v2"
)

type zpaPage[T any] struct {
	records    []T
	totalPages int
	response   *http.Response
}

type zpaPaginationQuery struct {
	PageSize      int     `json:"pagesize,omitempty" url:"pagesize,omitempty"`
	Page          int     `json:"page,omitempty" url:"page,omitempty"`
	MicroTenantID *string `json:"microtenantId,omitempty" url:"microtenantId,omitempty"`
}

type zpaPageEnvelope[T any] struct {
	TotalPages json.RawMessage `json:"totalPages"`
	List       []T             `json:"list"`
}

func parseZPATotalPages(raw json.RawMessage) (int, error) {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		return 0, paginationFailure("zpa pagination response is missing totalPages")
	}
	if strings.HasPrefix(value, `"`) {
		var quoted string
		if err := json.Unmarshal(raw, &quoted); err != nil {
			return 0, fmt.Errorf("decode zpa totalPages string: %w", err)
		}
		value = strings.TrimSpace(quoted)
	}

	totalPages, err := strconv.Atoi(value)
	if err == nil && totalPages >= 0 {
		return totalPages, nil
	}

	// The SDK decoded unquoted JSON numbers through float64 before formatting
	// them, so an integral value such as 1.0 was accepted as "1". Preserve that
	// envelope compatibility while still rejecting fractional and negative
	// values. Quoted "1.0" remains invalid, matching the SDK's string path.
	if strings.ContainsAny(value, ".eE") && !strings.HasPrefix(strings.TrimSpace(string(raw)), `"`) {
		number, floatErr := strconv.ParseFloat(value, 64)
		if floatErr == nil {
			totalPages, intErr := strconv.Atoi(fmt.Sprintf("%v", number))
			if intErr == nil && totalPages >= 0 {
				return totalPages, nil
			}
		}
	}
	return 0, paginationFailure("zpa pagination response has invalid totalPages")
}

// zpaPaginate follows the page count declared by the first response and fails
// closed if later responses contradict it. The vendored SDK silently treats a
// missing or malformed totalPages value as zero, rewrites the loop bound from
// every subsequent page, and has no ceiling.
func zpaPaginate[T any](
	ctx context.Context,
	fetchPage func(context.Context, int, int) (zpaPage[T], error),
) ([]T, *http.Response, error) {
	var (
		all              []T
		expectedPages    = -1
		lastResponse     *http.Response
		pageFingerprints = make(map[pageFingerprint]struct{})
	)

	for pageNumber := 1; ; pageNumber++ {
		if pageNumber > zpaMaxPages {
			return nil, lastResponse, paginationFailure(
				"zpa pagination exceeded the ceiling of %d pages (%d records)",
				zpaMaxPages,
				len(all),
			)
		}

		page, err := fetchPage(ctx, pageNumber, zpaPageSize)
		if err != nil {
			return nil, page.response, err
		}
		lastResponse = page.response

		if pageNumber == 1 {
			expectedPages = page.totalPages
			if expectedPages > zpaMaxPages {
				return nil, lastResponse, paginationFailure(
					"zpa pagination declared %d pages, exceeding the ceiling of %d",
					expectedPages,
					zpaMaxPages,
				)
			}
			if expectedPages == 0 {
				if len(page.records) != 0 {
					return nil, lastResponse, paginationFailure(
						"zpa pagination declared zero pages but returned %d records",
						len(page.records),
					)
				}
				return all, lastResponse, nil
			}
		} else if page.totalPages != expectedPages {
			return nil, lastResponse, paginationFailure(
				"zpa pagination totalPages changed from %d to %d on page %d",
				expectedPages,
				page.totalPages,
				pageNumber,
			)
		}

		if len(page.records) == 0 && expectedPages > 1 {
			return nil, lastResponse, paginationFailure(
				"zpa pagination returned an empty page %d of %d",
				pageNumber,
				expectedPages,
			)
		}
		// Exact whole-page fingerprints catch repeated pages and non-adjacent
		// cycles. They do not attempt to identify partial overlap between pages.
		fingerprint, err := fingerprintPage(page.records)
		if err != nil {
			return nil, lastResponse, fmt.Errorf(
				"failed to fingerprint zpa page %d: %w",
				pageNumber,
				err,
			)
		}
		if _, seen := pageFingerprints[fingerprint]; seen {
			return nil, lastResponse, paginationFailure(
				"zpa pagination repeated page content at page %d while %d pages were declared",
				pageNumber,
				expectedPages,
			)
		}
		pageFingerprints[fingerprint] = struct{}{}

		all = append(all, page.records...)
		if pageNumber >= expectedPages {
			return all, lastResponse, nil
		}
	}
}

func getZPAAllPages[T any](
	ctx context.Context,
	service *zsdk.Service,
	apiVersion string,
	endpointSuffix string,
) ([]T, *http.Response, error) {
	return getZPAAllPagesWithMicrotenant[T](
		ctx,
		service,
		apiVersion,
		endpointSuffix,
		nil,
	)
}

func getZPAAllPagesForMicrotenant[T any](
	ctx context.Context,
	service *zsdk.Service,
	apiVersion string,
	endpointSuffix string,
) ([]T, *http.Response, error) {
	return getZPAAllPagesWithMicrotenant[T](
		ctx,
		service,
		apiVersion,
		endpointSuffix,
		service.MicroTenantID(),
	)
}

func getZPAAllPagesWithMicrotenant[T any](
	ctx context.Context,
	service *zsdk.Service,
	apiVersion string,
	endpointSuffix string,
	microTenantID *string,
) ([]T, *http.Response, error) {
	endpoint := "/zpa/mgmtconfig/" + apiVersion + "/admin/customers/" +
		service.Client.GetCustomerID() + endpointSuffix
	return getZPAEndpointAllPages[T](ctx, service, endpoint, endpointSuffix, microTenantID)
}

// getZPAUserConfigAllPages reads a paged list under the userconfig API
// (SCIM groups), which sits beside mgmtconfig rather than under it.
func getZPAUserConfigAllPages[T any](
	ctx context.Context,
	service *zsdk.Service,
	endpointSuffix string,
) ([]T, *http.Response, error) {
	endpoint := "/zpa/userconfig/v1/customers/" + service.Client.GetCustomerID() + endpointSuffix
	return getZPAEndpointAllPages[T](ctx, service, endpoint, endpointSuffix, nil)
}

func getZPAEndpointAllPages[T any](
	ctx context.Context,
	service *zsdk.Service,
	endpoint string,
	endpointSuffix string,
	microTenantID *string,
) ([]T, *http.Response, error) {
	items, response, err := zpaPaginate(ctx, func(ctx context.Context, pageNumber, pageSize int) (zpaPage[T], error) {
		// NewRequestDo injects the client-level ZPA microtenant when this field
		// is nil. Replaced SDK methods that accepted service.WithMicroTenant
		// pass that service-scoped override explicitly instead.
		query := zpaPaginationQuery{
			PageSize:      pageSize,
			Page:          pageNumber,
			MicroTenantID: microTenantID,
		}

		var envelope zpaPageEnvelope[T]
		response, err := zpaGetJSON(ctx, service, endpoint, query, &envelope)
		if err != nil {
			return zpaPage[T]{response: response}, err
		}
		totalPages, err := parseZPATotalPages(envelope.TotalPages)
		if err != nil {
			return zpaPage[T]{response: response}, fmt.Errorf(
				"read zpa %s page %d metadata: %w",
				endpointSuffix,
				pageNumber,
				err,
			)
		}
		return zpaPage[T]{
			records:    envelope.List,
			totalPages: totalPages,
			response:   response,
		}, nil
	})
	if err != nil {
		return nil, response, err
	}

	items, err = zsdk.ApplyJMESPathFromContext(ctx, items)
	if err != nil {
		return nil, response, fmt.Errorf("apply zpa list filter: %w", err)
	}
	return items, response, nil
}

// getZPAUnpagedList reads a mgmtconfig route that answers one bare JSON
// array instead of the paged envelope (roles do, though the SDK's GetAll
// pages them and fails to decode). It is one request; there is no page to
// follow, so completeness rests on the API returning the whole list.
func getZPAUnpagedList[T any](
	ctx context.Context,
	service *zsdk.Service,
	apiVersion string,
	endpointSuffix string,
	microTenantID *string,
) ([]T, *http.Response, error) {
	endpoint := "/zpa/mgmtconfig/" + apiVersion + "/admin/customers/" +
		service.Client.GetCustomerID() + endpointSuffix
	var items []T
	// microTenantID is the service-scoped override where the SDK sends one;
	// nil leaves only the client-level microtenant.
	query := zpaPaginationQuery{MicroTenantID: microTenantID}
	response, err := zpaGetJSON(ctx, service, endpoint, query, &items)
	if err != nil {
		return nil, response, err
	}
	// A JSON null decodes to a nil slice; only an explicit [] means none.
	if items == nil {
		return nil, response, errors.New("read zpa list: response is not an array")
	}
	return items, response, nil
}

// zpaGetJSON GETs a ZPA route and decodes the body into out itself. Passing
// a decode target to NewRequestDo would run the SDK's unescapeHTML, which
// round-trips an object response through map[string]interface{} and so
// rounds every integer above 2^53 (SCIM groups carry int64 IdP ids that
// large). With a nil target the SDK returns the raw body untouched.
func zpaGetJSON(ctx context.Context, service *zsdk.Service, endpoint string, query any, out any) (*http.Response, error) {
	response, err := service.Client.NewRequestDo(ctx, http.MethodGet, endpoint, query, nil, nil)
	if err != nil {
		return response, err
	}
	if response == nil || response.Body == nil {
		return response, errors.New("read zpa response: no body")
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return response, fmt.Errorf("read zpa response body: %w", err)
	}
	// A blank 200 is never a valid read; the SDK's decode failed on it too.
	if len(bytes.TrimSpace(body)) == 0 {
		return response, errors.New("read zpa response: empty body")
	}
	if err := json.Unmarshal(body, out); err != nil {
		return response, err
	}
	return response, nil
}

// getZPASingleton reads one settings object from a mgmtconfig v1 route. A
// JSON null or a non-object body fails rather than reading as empty settings.
func getZPASingleton[T any](ctx context.Context, service *zsdk.Service, endpointSuffix string) (*T, *http.Response, error) {
	endpoint := "/zpa/mgmtconfig/v1/admin/customers/" + service.Client.GetCustomerID() + endpointSuffix
	var item *T
	// The SDK's settings reads pass no query; the client-level microtenant
	// still applies.
	response, err := zpaGetJSON(ctx, service, endpoint, nil, &item)
	if err != nil {
		return nil, response, err
	}
	if item == nil {
		return nil, response, errors.New("read zpa settings: response is not an object")
	}
	return item, response, nil
}
