package zscaler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	zsdk "github.com/zscaler/zscaler-sdk-go/v3/zscaler"
	activation "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/activation"
	adaptiveaccess "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/adaptive_access"
	ziaadminusers "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/adminuserrolemgmt/admins"
	ziaadminroles "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/adminuserrolemgmt/roles"
	advancedsettings "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/advanced_settings"
	advancedthreatsettings "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/advancedthreatsettings"
	"github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/alerts"
	authsettings "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/auth_settings"
	bandwidthclasses "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/bandwidth_control/bandwidth_classes"
	bandwidthcontrolrules "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/bandwidth_control/bandwidth_control_rules"
	browsercontrolsettings "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/browser_control_settings"
	browserisolation "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/browser_isolation"
	c2cincidentreceiver "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/c2c_incident_receiver"
	cloudappinstances "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/cloud_app_instances"
	cloudappcontrol "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/cloudappcontrol"
	cloudapplications "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/cloudapplications/cloudapplications"
	riskprofiles "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/cloudapplications/risk_profiles"
	cloudnss "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/cloudnss/cloudnss"
	nssservers "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/cloudnss/nss_servers"
	ziacommon "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/common"
	"github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/devicegroups"
	dlpengines "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/dlp/dlp_engines"
	dlpexactdatamatch "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/dlp/dlp_exact_data_match"
	dlpedmlite "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/dlp/dlp_exact_data_match_lite"
	dlpglobaloptions "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/dlp/dlp_global_options"
	dlpicapservers "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/dlp/dlp_icap_servers"
	dlpidmprofilelite "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/dlp/dlp_idm_profile_lite"
	dlpidmprofiles "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/dlp/dlp_idm_profiles"
	dlpincidentreceivers "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/dlp/dlp_incident_receiver_servers"
	dlpnotificationtemplates "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/dlp/dlp_notification_templates"
	dlpwebrules "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/dlp/dlp_web_rules"
	dlpdictionaries "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/dlp/dlpdictionaries"
	emailprofiles "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/email_profiles"
	endusernotification "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/end_user_notification"
	endpointdlprules "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/endpoint_dlp/endpoint_dlp_rules"
	emaildlprules "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/endpoint_dlp/outbound_email_dlp"
	filetypecontrol "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/filetypecontrol"
	customfiletypes "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/filetypecontrol/custom_file_types"
	firewalldnscontrolpolicies "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/firewalldnscontrolpolicies"
	applicationservices "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/firewallpolicies/applicationservices"
	appservicegroups "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/firewallpolicies/appservicegroups"
	dnsappgroups "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/firewallpolicies/dns_application_groups"
	dnsgateways "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/firewallpolicies/dns_gateways"
	filteringrules "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/firewallpolicies/filteringrules"
	ipdestinationgroups "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/firewallpolicies/ipdestinationgroups"
	ipsourcegroups "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/firewallpolicies/ipsourcegroups"
	networkapplicationgroups "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/firewallpolicies/networkapplicationgroups"
	networkapplications "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/firewallpolicies/networkapplications"
	networkservicegroups "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/firewallpolicies/networkservicegroups"
	networkservices "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/firewallpolicies/networkservices"
	"github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/firewallpolicies/timewindow"
	forwardingrules "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/forwarding_control_policy/forwarding_rules"
	"github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/forwarding_control_policy/proxies"
	proxygateways "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/forwarding_control_policy/proxy_gateways"
	zpagateways "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/forwarding_control_policy/zpa_gateways"
	ftpcontrolpolicy "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/ftp_control_policy"
	httpheaderactionprofile "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/http_header_control/http_header_action_profile"
	httpheaderprofile "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/http_header_control/http_header_profile"
	intermediatecacertificates "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/intermediatecacertificates"
	ipspolicies "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/ips_control_policies/ips_policies"
	ipssignaturerules "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/ips_control_policies/ips_signature_rules"
	"github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/location/locationgroups"
	"github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/location/locationmanagement"
	malwareprotection "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/malware_protection"
	mobilethreatsettings "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/mobile_threat_settings"
	natcontrol "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/nat_control_policies"
	organizationdetails "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/organization_details"
	pacfiles "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/pacfiles"
	remoteassistance "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/remote_assistance"
	rulelabels "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/rule_labels"
	saassecurityapi "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/saas_security_api"
	casbdlprules "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/saas_security_api/casb_dlp_rules"
	casbmalwarerules "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/saas_security_api/casb_malware_rules"
	sandboxrules "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/sandbox/sandbox_rules"
	sandboxsettings "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/sandbox/sandbox_settings"
	securebrowsing "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/secure_browsing"
	securitypolicysettings "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/security_policy_settings"
	alertdefinitions "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/security_ueba_alerts/alert_definitions"
	"github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/sslinspection"
	tenancyrestriction "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/tenancy_restriction"
	timeintervals "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/time_intervals"
	dcexclusions "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/trafficforwarding/dc_exclusions"
	gretunnels "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/trafficforwarding/gretunnels"
	ipv6config "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/trafficforwarding/ipv6_config"
	staticips "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/trafficforwarding/staticips"
	subclouds "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/trafficforwarding/sub_clouds"
	"github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/urlcategories"
	"github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/urlfilteringpolicies"
	userauthsettings "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/user_authentication_settings"
	userdepartments "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/usermanagement/departments"
	usergroups "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/usermanagement/groups"
	ziausers "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/usermanagement/users"
	vzenclusters "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/vzen_clusters"
	vzennodes "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/vzen_nodes"
	"github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/workloadgroups"

	"zscalerctl/internal/resources"
)

// ziaMaxPages is a fail-closed ceiling on the number of pages the bounded ZIA
// paginators will fetch. Like the zcc/zidentity guards, it converts a
// pathological endpoint into a visible, descriptive error instead of an
// unbounded loop. Even when an endpoint clamps pages to 20 records, the ceiling
// still admits nearly 20,000 records before requiring a terminal short page.
const ziaMaxPages = 1000

// errPaginationValidation marks a list a page walker (ZIA, ZPA, ZCC, ZTW,
// ZIdentity, and the single-page network-applications read) refused because
// it could not prove the result complete; it classifies as
// pagination_incomplete. Callers that tolerate per-parent read errors must
// still propagate it.
var errPaginationValidation = errors.New("pagination validation failed")

// paginationError is a page walker's refusal. Its check names the failed
// check with counts, offsets and page numbers only, never record values, so
// the error envelope can show it.
type paginationError struct{ check string }

func (e paginationError) Error() string { return errPaginationValidation.Error() + ": " + e.check }

func (e paginationError) Is(target error) bool { return target == errPaginationValidation }

func paginationFailure(format string, args ...any) error {
	return paginationError{check: fmt.Sprintf(format, args...)}
}

// ziaWalkPages walks a ZIA list endpoint while enforcing the
// ziaMaxPages ceiling the vendored ReadAllPages lacks. A ZIA endpoint can
// silently clamp a requested page size (urlCategories returned 20 records for a
// 5,000-record request on one tenant), so a short first page does not prove
// completion. The first nonempty response establishes the effective page width;
// a later short page completes the walk. Repeated pages, repeated record
// identities, and width growth fail closed because each
// can indicate that the server ignored pagination or changed its pagination
// contract mid-read. One repeat can be resolved instead. When page 1 was
// shorter than requested and page 2 returns the same records, in any order and
// whether or not they carry ids, the endpoint ignored the page
// parameter. That alone does not prove page 1 was complete, because a server
// that caps each response (min(pageSize, cap)) and ignores the page would look
// the same. So the walker asks for page 1 again with a page size one smaller:
// an endpoint that still returns every record ignores paging entirely, and
// page 1 is everything it will return (observed live on a different tenant for
// urlCategories, about 25 ZIA list endpoints in all); one that returns fewer
// honors sizes and may be capping, so the walk fails closed. A fixed window
// that ignores both page and size is indistinguishable and accepted; no client
// can read past it.
// Records must match field for field, so a tenant change between requests
// still fails closed. visitPage may
// stop the walk after inspecting a validated page; point lookups use that path
// so an already-found record does not depend on a later confirmation request.
func ziaWalkPages[T any](
	ctx context.Context,
	pageSize int,
	fetchPage func(ctx context.Context, page, pageSize int) ([]T, error),
	visitPage func([]T) bool,
) error {
	if pageSize <= 0 {
		return paginationFailure("zia pagination page size must be positive: %d", pageSize)
	}

	var (
		effectivePageSize int
		recordsSeen       int
		pageFingerprints  = make(map[pageFingerprint]struct{})
		recordIdentities  = make(map[string]struct{})
		firstPage         pageSummary
	)
	for page := 1; ; page++ {
		if page > ziaMaxPages {
			return paginationFailure("zia pagination exceeded the ceiling of %d pages (%d records); the endpoint kept returning full pages, so completeness cannot be guaranteed", ziaMaxPages, recordsSeen)
		}
		items, err := fetchPage(ctx, page, pageSize)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			break
		}
		if effectivePageSize == 0 {
			effectivePageSize = len(items)
		} else if len(items) > effectivePageSize {
			return paginationFailure("zia pagination page width changed from %d to %d on page %d; completeness cannot be guaranteed", effectivePageSize, len(items), page)
		}

		fingerprint, err := fingerprintPage(items)
		if err != nil {
			return paginationFailure("failed to fingerprint zia page %d", page)
		}
		summary, err := summarizePage(items)
		if err != nil {
			return paginationFailure("failed to inspect zia records on page %d", page)
		}
		if page == 1 {
			firstPage = summary
		} else if page == 2 && firstPage.records < pageSize && summary.sameRecords(firstPage) {
			complete, err := ziaIgnoresPaging(ctx, firstPage, fetchPage)
			if err != nil {
				return err
			}
			if complete {
				return nil
			}
			return paginationFailure("zia endpoint ignored the page parameter but honored a smaller page size, so page 1 may be capped; completeness cannot be guaranteed")
		}
		if _, seen := pageFingerprints[fingerprint]; seen {
			return paginationFailure("zia pagination received repeated page content at page %d; completeness cannot be guaranteed", page)
		}
		pageFingerprints[fingerprint] = struct{}{}
		repeated := 0
		for _, item := range items {
			identity, hasIdentity, err := paginationRecordIdentity(item)
			if err != nil {
				return paginationFailure("failed to inspect zia record identity on page %d", page)
			}
			if !hasIdentity {
				continue
			}
			if _, seen := recordIdentities[identity]; seen {
				repeated++
				continue
			}
			recordIdentities[identity] = struct{}{}
		}
		if repeated > 0 {
			return paginationFailure("zia pagination received duplicate record identity on page %d: %d of its %d records repeat an id already read (page 1 had %d); completeness cannot be guaranteed", page, repeated, len(items), firstPage.records)
		}

		recordsSeen += len(items)
		if visitPage(items) {
			return nil
		}
		if page > 1 && len(items) < effectivePageSize {
			break
		}
	}
	return nil
}

// ziaIgnoresPaging re-requests page 1 with a page size one below what page 1
// returned. Every record coming back shows the endpoint ignores page and size
// alike, so page 1 is everything it will return. A one-record page cannot be
// probed smaller; a cap of one record is not a real server limit, so it is
// accepted. The probe is one request outside the ziaMaxPages ceiling.
func ziaIgnoresPaging[T any](
	ctx context.Context,
	firstPage pageSummary,
	fetchPage func(ctx context.Context, page, pageSize int) ([]T, error),
) (bool, error) {
	if firstPage.records <= 1 {
		return true, nil
	}
	probe, err := fetchPage(ctx, 1, firstPage.records-1)
	if err != nil {
		return false, err
	}
	summary, err := summarizePage(probe)
	if err != nil {
		return false, paginationFailure("failed to inspect zia page-size probe")
	}
	return summary.sameRecords(firstPage), nil
}

// ziaPaginate collects every validated page. It discards the partial aggregate
// whenever ziaWalkPages cannot prove completeness.
func ziaPaginate[T any](ctx context.Context, pageSize int, fetchPage func(ctx context.Context, page, pageSize int) ([]T, error)) ([]T, error) {
	var all []T
	err := ziaWalkPages(ctx, pageSize, fetchPage, func(items []T) bool {
		all = append(all, items...)
		return false
	})
	if err != nil {
		return nil, err
	}
	return all, nil
}

func getZIAAllPages[T any](
	ctx context.Context,
	service *zsdk.Service,
	endpoint string,
) ([]T, error) {
	return getZIAAllPagesWithSize[T](ctx, service, endpoint, ziacommon.GetPageSize())
}

func getZIAAllPagesWithSize[T any](
	ctx context.Context,
	service *zsdk.Service,
	endpoint string,
	pageSize int,
) ([]T, error) {
	items, err := ziaPaginate(ctx, pageSize, func(ctx context.Context, page, size int) ([]T, error) {
		var pageItems []T
		err := ziacommon.ReadPage(ctx, service.Client, endpoint, page, &pageItems, size)
		return pageItems, err
	})
	if err != nil {
		return nil, err
	}
	items, err = zsdk.ApplyJMESPathFromContext(ctx, items)
	if err != nil {
		return nil, fmt.Errorf("apply zia list filter: %w", err)
	}
	return items, nil
}

func ziaSortedListEndpoint(service *zsdk.Service, endpoint string) string {
	query := url.Values{}
	if service.SortBy != "" {
		query.Set("sortBy", string(service.SortBy))
	}
	if service.SortOrder != "" {
		query.Set("sortOrder", string(service.SortOrder))
	}
	if len(query) == 0 {
		return endpoint
	}
	return endpoint + "?" + query.Encode()
}

func getZIASublocationsForParentAllPages(
	ctx context.Context,
	service *zsdk.Service,
	parentID int,
) ([]locationmanagement.Locations, error) {
	endpoint := fmt.Sprintf("/zia/api/v1/locations/%d/sublocations", parentID)
	return getZIAAllPages[locationmanagement.Locations](ctx, service, endpoint)
}

func getZIASublocationForParentByID(
	ctx context.Context,
	service *zsdk.Service,
	parentID int,
	id int,
) (*locationmanagement.Locations, error) {
	endpoint := fmt.Sprintf("/zia/api/v1/locations/%d/sublocations", parentID)
	var found *locationmanagement.Locations
	err := ziaWalkPages(ctx, ziacommon.GetPageSize(), func(ctx context.Context, page, size int) ([]locationmanagement.Locations, error) {
		var items []locationmanagement.Locations
		err := ziacommon.ReadPage(ctx, service.Client, endpoint, page, &items, size)
		return items, err
	}, func(items []locationmanagement.Locations) bool {
		for index := range items {
			if items[index].ID == id {
				item := items[index]
				found = &item
				return true
			}
		}
		return false
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}

func getZIASublocationsAllPages(ctx context.Context, service *zsdk.Service) ([]locationmanagement.Locations, error) {
	parents, err := getZIALocationsAllPages(ctx, service)
	if err != nil {
		return nil, err
	}

	var sublocations []locationmanagement.Locations
	for _, parent := range parents {
		items, err := getZIASublocationsForParentAllPages(ctx, service, parent.ID)
		if err != nil {
			return nil, err
		}
		sublocations = append(sublocations, items...)
	}
	return sublocations, nil
}

func getZIASublocationByID(
	ctx context.Context,
	service *zsdk.Service,
	id int,
) (*locationmanagement.Locations, error) {
	var found *locationmanagement.Locations
	var searchErr, skipped error
	const pageSize = 1000
	err := ziaWalkPages(ctx, pageSize, func(ctx context.Context, page, size int) ([]locationmanagement.Locations, error) {
		var parents []locationmanagement.Locations
		err := ziacommon.ReadPage(ctx, service.Client, "/zia/api/v1/locations", page, &parents, size)
		return parents, err
	}, func(parents []locationmanagement.Locations) bool {
		for _, parent := range parents {
			sublocation, err := getZIASublocationForParentByID(ctx, service, parent.ID, id)
			if err != nil {
				// Preserve the SDK get helper's tolerance for an inaccessible
				// parent, but propagate pagination validation errors and never
				// turn caller cancellation into a false not-found.
				if ctxErr := ctx.Err(); ctxErr != nil {
					searchErr = ctxErr
					return true
				}
				if errors.Is(err, errPaginationValidation) {
					searchErr = err
					return true
				}
				// Keep one skip to report on a miss; a parent that 404s (deleted
				// mid-scan) must not hide a later one that may hold the ID.
				if skipped == nil || sdkStatusCode(skipped) == http.StatusNotFound {
					skipped = err
				}
				continue
			}
			if sublocation != nil {
				found = sublocation
				return true
			}
		}
		return false
	})
	if searchErr != nil {
		return nil, searchErr
	}
	if err != nil {
		return nil, err
	}
	if found != nil {
		return found, nil
	}
	// A miss is not-found only when every parent was read; a parent skipped
	// above may hold the ID, so its error is the answer (exit 5, not 4). A
	// complete scan has no HTTP 404 to classify downstream, so it carries the
	// not-found sentinel itself.
	if skipped != nil {
		return nil, skipped
	}
	return nil, fmt.Errorf("%w: %w", ErrResourceNotFound, resources.ErrRecordNotFound)
}

func getZIABrowserIsolationAllPages(
	ctx context.Context,
	service *zsdk.Service,
) ([]browserisolation.CBIProfile, error) {
	items, err := getZIAAllPages[browserisolation.CBIProfile](
		ctx,
		service,
		"/zia/api/v1/browserIsolation/profiles",
	)
	if err == nil {
		return items, nil
	}
	return nil, normalizeZIABrowserIsolationListError(err)
}

func normalizeZIABrowserIsolationListError(err error) error {
	if err == nil {
		return nil
	}
	const subscriptionMessage = "Cloud Browser Isolation subscription is required"
	// Only a non-retryable 4xx means the licence is missing; a 5xx, 408 or
	// 429 that happens to carry the text keeps its retryable classification.
	if status := sdkStatusCode(err); status >= 400 && status < 500 &&
		status != http.StatusTooManyRequests && status != http.StatusRequestTimeout &&
		strings.Contains(err.Error(), subscriptionMessage) {
		// Match the SDK wrapper's intentionally status-free operator message.
		return fmt.Errorf("%w: %s", errNotSubscribed, subscriptionMessage)
	}
	return err
}

// getZIANetworkServices reads /zia/api/v1/networkServices in one request. The
// endpoint is not paginated and some clouds reject a pageSize with HTTP 400
// ("Invalid Page Size"), so it gets no page or pageSize; the SDK's
// GetAllNetworkServices does exactly that since v3.8.57.
func getZIANetworkServices(ctx context.Context, service *zsdk.Service) ([]networkservices.NetworkServices, error) {
	return networkservices.GetAllNetworkServices(ctx, service, nil, nil)
}

// getZIAUsersAllPages reads /zia/api/v1/users with a bounded page walk. It
// replaces ziausers.GetAllUsers, whose vendored ReadAllPages loop has no page
// ceiling; the endpoint, service-level sort options, 10000-record page size,
// short-page termination, and post-aggregation JMESPath filtering are
// preserved.
func getZIAUsersAllPages(ctx context.Context, service *zsdk.Service) ([]ziausers.Users, error) {
	const pageSize = 10000
	return getZIAAllPagesWithSize[ziausers.Users](
		ctx,
		service,
		ziaSortedListEndpoint(service, "/zia/api/v1/users"),
		pageSize,
	)
}

// getZIALocationsAllPages reads /zia/api/v1/locations with a bounded page walk,
// replacing locationmanagement.GetAll's unbounded loop. Endpoint, 1000-record
// page size, and short-page termination match the SDK.
func getZIALocationsAllPages(ctx context.Context, service *zsdk.Service) ([]locationmanagement.Locations, error) {
	const pageSize = 1000
	return ziaPaginate(ctx, pageSize, func(ctx context.Context, page, size int) ([]locationmanagement.Locations, error) {
		var items []locationmanagement.Locations
		err := ziacommon.ReadPage(ctx, service.Client, "/zia/api/v1/locations", page, &items, size)
		return items, err
	})
}

// getZIALocationGroupsAllPages reads /zia/api/v1/locations/groups with the
// fetchLocations option enabled so list and dump records include associated
// parent locations and sublocations. It preserves the SDK's documented
// 1000-record page size while replacing its unbounded ReadAllPages loop with
// ziaPaginate's fail-closed page ceiling.
func getZIALocationGroupsAllPages(ctx context.Context, service *zsdk.Service) ([]locationgroups.LocationGroup, error) {
	const pageSize = 1000
	return getZIAAllPagesWithSize[locationgroups.LocationGroup](
		ctx,
		service,
		"/zia/api/v1/locations/groups?fetchLocations=true",
		pageSize,
	)
}

// getZIAURLFilteringRulesAllPages reads /zia/api/v1/urlFilteringRules with an
// explicit bounded page walk. A read with no paging parameters gets only the
// API's 100-record default page, which truncated rule lists in live use (100
// returned, more in the console).
// Keeping the explicit page size at 100 matches Zscaler's published
// pagination example and the observed default boundary. It also avoids relying
// on a larger, endpoint-specific maximum that the API reference does not state.
// The vendored SDK's GetAll now pages too, but asks for 1,000 records a page,
// has no page ceiling, and stops at the first shorter page, so it depends on
// the server honoring that size.
func getZIAURLFilteringRulesAllPages(ctx context.Context, service *zsdk.Service) ([]urlfilteringpolicies.URLFilteringRule, error) {
	const pageSize = 100
	return ziaPaginate(ctx, pageSize, func(ctx context.Context, page, size int) ([]urlfilteringpolicies.URLFilteringRule, error) {
		var items []urlfilteringpolicies.URLFilteringRule
		err := ziacommon.ReadPage(ctx, service.Client, "/zia/api/v1/urlFilteringRules", page, &items, size)
		return items, err
	})
}

// getZIAURLFilteringRule reads one URL filtering rule directly and enriches
// ISOLATE rules whose detail response has the known omitted cbiProfile object.
// The SDK's Get helper performs this enrichment through its own GetAll call;
// keeping the workaround here reuses the bounded list adapter instead.
func getZIAURLFilteringRule(
	ctx context.Context,
	service *zsdk.Service,
	id int,
) (*urlfilteringpolicies.URLFilteringRule, error) {
	var rule urlfilteringpolicies.URLFilteringRule
	if err := service.Client.Read(ctx, fmt.Sprintf("/zia/api/v1/urlFilteringRules/%d", id), &rule); err != nil {
		return nil, err
	}

	// The API omission is specific to ISOLATE rules that identify a profile.
	// Leave all other detail responses on the direct-read path, including an
	// ISOLATE response that legitimately has no profile ID to enrich.
	if rule.Action != "ISOLATE" || rule.CBIProfile != nil || rule.CBIProfileID == 0 {
		return &rule, nil
	}

	rules, err := getZIAURLFilteringRulesAllPages(ctx, service)
	if err != nil {
		return nil, fmt.Errorf("enrich zia url-filtering rule %d cbiProfile: %w", id, err)
	}
	for index := range rules {
		if rules[index].ID != id {
			continue
		}
		if rules[index].CBIProfile == nil {
			return nil, fmt.Errorf(
				"enrich zia url-filtering rule %d cbiProfile: list response omitted profile ID %d",
				id,
				rule.CBIProfileID,
			)
		}
		profileID, parseErr := strconv.Atoi(strings.TrimSpace(rules[index].CBIProfile.ID))
		if parseErr != nil || profileID != rule.CBIProfileID {
			return nil, fmt.Errorf(
				"enrich zia url-filtering rule %d cbiProfile: list profile ID does not match detail cbiProfileId %d",
				id,
				rule.CBIProfileID,
			)
		}
		rule.CBIProfile = rules[index].CBIProfile
		return &rule, nil
	}

	return nil, fmt.Errorf(
		"enrich zia url-filtering rule %d cbiProfile: list response omitted rule",
		id,
	)
}

// getZIAURLCategoriesAll reads every /zia/api/v1/urlCategories page. The SDK's
// GetAll issues a single Read, which is complete only on some tenants. One
// tenant returned 20 records for a 5,000-record request (2026-07-29), and a
// single read with a ceiling check cannot detect that truncation. Another
// returned the whole collection on every request, whatever page and pageSize
// said (2026-10-14). What decides which a tenant does is not known, so the
// adaptive bounded paginator learns the response width and continues, and
// ziaWalkPages resolves the ignored-paging case.
// includeOnlyUrlKeywordCounts=true preserves the prior GetAll(customOnly=false,
// includeOnlyUrlKeywordCounts=true) payload shape. type=ALL is required to
// include TLD_CATEGORY records; omitting the type returns only predefined and
// custom URL_CATEGORY records.
func getZIAURLCategoriesAll(ctx context.Context, service *zsdk.Service) ([]urlcategories.URLCategory, error) {
	const requestedPageSize = 5000
	return ziaPaginate(ctx, requestedPageSize, func(ctx context.Context, page, size int) ([]urlcategories.URLCategory, error) {
		var categories []urlcategories.URLCategory
		err := ziacommon.ReadPage(
			ctx,
			service.Client,
			"/zia/api/v1/urlCategories?includeOnlyUrlKeywordCounts=true&type=ALL",
			page,
			&categories,
			size,
		)
		return categories, err
	})
}

func addZIAHandlers(m map[resourceKey]resourceHandler, client sdkClient) {
	entries := map[resourceKey]resourceHandler{
		{product: resources.ProductZIA, name: resourceLocations}: newListGetHandler(
			resourceLocations,
			ziaSDKList(client, getZIALocationsAllPages),
			ziaSDKGet(client, locationmanagement.GetLocation),
			locationSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceLocationGroups}: newListGetHandler(
			resourceLocationGroups,
			ziaSDKList(client, getZIALocationGroupsAllPages),
			ziaSDKGet(client, locationgroups.GetLocationGroup),
			locationGroupSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceRuleLabels}: newListGetHandler(
			resourceRuleLabels,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]rulelabels.RuleLabels, error) {
				return getZIAAllPages[rulelabels.RuleLabels](ctx, service, "/zia/api/v1/ruleLabels")
			}),
			ziaSDKGet(client, rulelabels.Get),
			ruleLabelSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceAuthSettings}: newSingletonHandler(
			resourceAuthSettings,
			ziaSDKShow(client, authsettings.Get),
			authSettingsSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceStaticIPs}: newListGetHandler(
			resourceStaticIPs,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]staticips.StaticIP, error) {
				return getZIAAllPages[staticips.StaticIP](ctx, service, "/zia/api/v1/staticIP")
			}),
			ziaSDKGet(client, staticips.Get),
			staticIPSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceGRETunnels}: newListGetHandler(
			resourceGRETunnels,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]gretunnels.GreTunnels, error) {
				return getZIAAllPages[gretunnels.GreTunnels](ctx, service, "/zia/api/v1/greTunnels")
			}),
			ziaSDKGet(client, gretunnels.GetGreTunnels),
			greTunnelSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceSublocations}: newListGetHandler(
			resourceSublocations,
			ziaSDKList(client, getZIASublocationsAllPages),
			ziaSDKGet(client, getZIASublocationByID),
			sublocationSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceSSLRules}: newListGetHandler(
			resourceSSLRules,
			ziaSDKList(client, sslinspection.GetAll),
			ziaSDKGet(client, sslinspection.Get),
			sslInspectionRuleSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceURLCategories}: newListGetHandler(
			resourceURLCategories,
			ziaSDKList(client, getZIAURLCategoriesAll),
			ziaSDKStringGet(client, urlcategories.Get),
			urlCategorySourceRecord,
		),
		{product: resources.ProductZIA, name: resourceURLRules}: newListGetHandler(
			resourceURLRules,
			ziaSDKList(client, getZIAURLFilteringRulesAllPages),
			ziaSDKGet(client, getZIAURLFilteringRule),
			urlFilteringRuleSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceFirewallRules}: newListGetHandler(
			resourceFirewallRules,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]filteringrules.FirewallFilteringRules, error) {
				return getZIAAllPagesWithSize[filteringrules.FirewallFilteringRules](
					ctx,
					service,
					"/zia/api/v1/firewallFilteringRules",
					5000,
				)
			}),
			ziaSDKGet(client, filteringrules.Get),
			firewallFilteringRuleSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceForwardingRules}: newListGetHandler(
			resourceForwardingRules,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]forwardingrules.ForwardingRules, error) {
				return getZIAAllPages[forwardingrules.ForwardingRules](ctx, service, "/zia/api/v1/forwardingRules")
			}),
			ziaSDKGet(client, forwardingrules.Get),
			forwardingRuleSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceIPSourceGroups}: newListGetHandler(
			resourceIPSourceGroups,
			ziaSDKList(client, ipsourcegroups.GetAll),
			ziaSDKGet(client, ipsourcegroups.Get),
			ipSourceGroupSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceIPDestGroups}: newListGetHandler(
			resourceIPDestGroups,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]ipdestinationgroups.IPDestinationGroups, error) {
				return ipdestinationgroups.GetAll(ctx, service, "")
			}),
			ziaSDKGet(client, ipdestinationgroups.Get),
			ipDestinationGroupSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceNetworkServices}: newListGetHandler(
			resourceNetworkServices,
			ziaSDKList(client, getZIANetworkServices),
			ziaSDKGet(client, networkservices.Get),
			networkServiceSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceNetworkSvcGroups}: newListGetHandler(
			resourceNetworkSvcGroups,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]networkservicegroups.NetworkServiceGroups, error) {
				return getZIAAllPages[networkservicegroups.NetworkServiceGroups](ctx, service, "/zia/api/v1/networkServiceGroups")
			}),
			ziaSDKGet(client, networkservicegroups.GetNetworkServiceGroups),
			networkServiceGroupSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceNetworkApps}: newListGetHandler(
			resourceNetworkApps,
			ziaSDKList(client, getNetworkApplicationsPage),
			ziaSDKStringGet(client, func(ctx context.Context, service *zsdk.Service, id string) (*networkapplications.NetworkApplications, error) {
				return networkapplications.GetNetworkApplication(ctx, service, id, "")
			}),
			networkApplicationSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceAppServices}: newListGetHandler(
			resourceAppServices,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]applicationservices.ApplicationServicesLite, error) {
				return getZIAAllPages[applicationservices.ApplicationServicesLite](ctx, service, "/zia/api/v1/appServices/lite")
			}),
			ziaSDKListGetByIntID(
				client,
				func(ctx context.Context, service *zsdk.Service) ([]applicationservices.ApplicationServicesLite, error) {
					return getZIAAllPages[applicationservices.ApplicationServicesLite](ctx, service, "/zia/api/v1/appServices/lite")
				},
				func(item applicationservices.ApplicationServicesLite) int { return item.ID },
			),
			applicationServiceSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceAppServiceGroups}: newListGetHandler(
			resourceAppServiceGroups,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]appservicegroups.ApplicationServicesGroupLite, error) {
				return getZIAAllPages[appservicegroups.ApplicationServicesGroupLite](ctx, service, "/zia/api/v1/appServiceGroups/lite")
			}),
			ziaSDKListGetByIntID(
				client,
				func(ctx context.Context, service *zsdk.Service) ([]appservicegroups.ApplicationServicesGroupLite, error) {
					return getZIAAllPages[appservicegroups.ApplicationServicesGroupLite](ctx, service, "/zia/api/v1/appServiceGroups/lite")
				},
				func(item appservicegroups.ApplicationServicesGroupLite) int { return item.ID },
			),
			applicationServiceGroupSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceNetworkAppGroups}: newListGetHandler(
			resourceNetworkAppGroups,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]networkapplicationgroups.NetworkApplicationGroups, error) {
				return getZIAAllPages[networkapplicationgroups.NetworkApplicationGroups](ctx, service, "/zia/api/v1/networkApplicationGroups")
			}),
			ziaSDKGet(client, networkapplicationgroups.GetNetworkApplicationGroups),
			networkApplicationGroupSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceTimeWindows}: newListGetHandler(
			resourceTimeWindows,
			ziaSDKList(client, timewindow.GetAll),
			ziaSDKListGetByIntID(
				client,
				timewindow.GetAll,
				func(item timewindow.TimeWindow) int { return item.ID },
			),
			timeWindowSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceProxies}: newListGetHandler(
			resourceProxies,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]proxies.Proxies, error) {
				return getZIAAllPages[proxies.Proxies](ctx, service, "/zia/api/v1/proxies")
			}),
			ziaSDKGet(client, proxies.Get),
			proxySourceRecord,
		),
		{product: resources.ProductZIA, name: resourceProxyGateways}: newListGetHandler(
			resourceProxyGateways,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]proxygateways.ProxyGateways, error) {
				return getZIAAllPages[proxygateways.ProxyGateways](ctx, service, "/zia/api/v1/proxyGateways")
			}),
			ziaSDKListGetByIntID(
				client,
				func(ctx context.Context, service *zsdk.Service) ([]proxygateways.ProxyGateways, error) {
					return getZIAAllPages[proxygateways.ProxyGateways](ctx, service, "/zia/api/v1/proxyGateways")
				},
				func(item proxygateways.ProxyGateways) int { return item.ID },
			),
			proxyGatewaySourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDedicatedIPGWs}: newListGetHandler(
			resourceDedicatedIPGWs,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]proxies.DedicatedIPGateways, error) {
				return getZIAAllPages[proxies.DedicatedIPGateways](ctx, service, "/zia/api/v1/dedicatedIPGateways/lite")
			}),
			ziaSDKListGetByIntID(
				client,
				func(ctx context.Context, service *zsdk.Service) ([]proxies.DedicatedIPGateways, error) {
					return getZIAAllPages[proxies.DedicatedIPGateways](ctx, service, "/zia/api/v1/dedicatedIPGateways/lite")
				},
				func(item proxies.DedicatedIPGateways) int { return item.Id },
			),
			dedicatedIPGatewaySourceRecord,
		),
		{product: resources.ProductZIA, name: resourceTimeIntervals}: newListGetHandler(
			resourceTimeIntervals,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]timeintervals.TimeInterval, error) {
				return getZIAAllPages[timeintervals.TimeInterval](ctx, service, "/zia/api/v1/timeIntervals")
			}),
			ziaSDKGet(client, timeintervals.Get),
			timeIntervalSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceBandwidthClasses}: newListGetHandler(
			resourceBandwidthClasses,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]bandwidthclasses.BandwidthClasses, error) {
				return getZIAAllPages[bandwidthclasses.BandwidthClasses](ctx, service, "/zia/api/v1/bandwidthClasses")
			}),
			ziaSDKGet(client, bandwidthclasses.Get),
			bandwidthClassSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceBandwidthRules}: newListGetHandler(
			resourceBandwidthRules,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]bandwidthcontrolrules.BandwidthControlRules, error) {
				return getZIAAllPages[bandwidthcontrolrules.BandwidthControlRules](ctx, service, "/zia/api/v1/bandwidthControlRules")
			}),
			ziaSDKGet(client, bandwidthcontrolrules.Get),
			bandwidthControlRuleSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceIPSSignatureRules}: newListGetHandler(
			resourceIPSSignatureRules,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]ipssignaturerules.IPSSignatureRules, error) {
				return getZIAAllPages[ipssignaturerules.IPSSignatureRules](ctx, service, "/zia/api/v1/ipsSignatureRules")
			}),
			ziaSDKGet(client, ipssignaturerules.Get),
			ipsSignatureRuleSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceIPSPolicies}: newListGetHandler(
			resourceIPSPolicies,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]ipspolicies.FirewallIPSRules, error) {
				return getZIAAllPages[ipspolicies.FirewallIPSRules](ctx, service, "/zia/api/v1/firewallIpsRules")
			}),
			ziaSDKGet(client, ipspolicies.Get),
			ipsPolicySourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDNSGateways}: newListGetHandler(
			resourceDNSGateways,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]dnsgateways.DNSGateways, error) {
				return getZIAAllPages[dnsgateways.DNSGateways](ctx, service, "/zia/api/v1/dnsGateways")
			}),
			ziaSDKGet(client, dnsgateways.Get),
			dnsGatewaySourceRecord,
		),
		{product: resources.ProductZIA, name: resourceNATRules}: newListGetHandler(
			resourceNATRules,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]natcontrol.NatControlPolicies, error) {
				return getZIAAllPages[natcontrol.NatControlPolicies](ctx, service, "/zia/api/v1/dnatRules")
			}),
			ziaSDKGet(client, natcontrol.Get),
			natControlRuleSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceGroups}: newListGetHandler(
			resourceGroups,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]usergroups.Groups, error) {
				return getZIAAllPages[usergroups.Groups](
					ctx,
					service,
					ziaSortedListEndpoint(service, "/zia/api/v1/groups"),
				)
			}),
			ziaSDKGet(client, usergroups.GetGroups),
			groupSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDepartments}: newListGetHandler(
			resourceDepartments,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]userdepartments.Department, error) {
				return getZIAAllPages[userdepartments.Department](
					ctx,
					service,
					ziaSortedListEndpoint(service, "/zia/api/v1/departments"),
				)
			}),
			ziaSDKGet(client, userdepartments.GetDepartments),
			departmentSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceUsers}: newListGetHandler(
			resourceUsers,
			ziaSDKList(client, getZIAUsersAllPages),
			ziaSDKGet(client, ziausers.Get),
			userSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDeviceGroups}: newListGetHandler(
			resourceDeviceGroups,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]devicegroups.DeviceGroups, error) {
				return getZIAAllPages[devicegroups.DeviceGroups](ctx, service, "/zia/api/v1/deviceGroups")
			}),
			ziaSDKListGetByIntID(
				client,
				func(ctx context.Context, service *zsdk.Service) ([]devicegroups.DeviceGroups, error) {
					return getZIAAllPages[devicegroups.DeviceGroups](ctx, service, "/zia/api/v1/deviceGroups")
				},
				func(item devicegroups.DeviceGroups) int { return item.ID },
			),
			deviceGroupSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDevices}: newListGetHandler(
			resourceDevices,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]devicegroups.Devices, error) {
				return getZIAAllPages[devicegroups.Devices](ctx, service, "/zia/api/v1/deviceGroups/devices")
			}),
			ziaSDKListGetByIntID(
				client,
				func(ctx context.Context, service *zsdk.Service) ([]devicegroups.Devices, error) {
					return getZIAAllPages[devicegroups.Devices](ctx, service, "/zia/api/v1/deviceGroups/devices")
				},
				func(device devicegroups.Devices) int { return device.ID },
			),
			deviceSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceWorkloadGroups}: newListGetHandler(
			resourceWorkloadGroups,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]workloadgroups.WorkloadGroup, error) {
				return getZIAAllPages[workloadgroups.WorkloadGroup](ctx, service, "/zia/api/v1/workloadGroups")
			}),
			ziaSDKGet(client, workloadgroups.Get),
			workloadGroupSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceAlertSubs}: newListGetHandler(
			resourceAlertSubs,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]alerts.AlertSubscriptions, error) {
				return getZIAAllPages[alerts.AlertSubscriptions](ctx, service, "/zia/api/v1/alertSubscriptions")
			}),
			ziaSDKGet(client, alerts.Get),
			alertSubscriptionSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceActivationStatus}: newSingletonHandler(
			resourceActivationStatus,
			ziaSDKShow(client, activation.GetActivationStatus),
			structSourceRecord[activation.Activation],
		),
		{product: resources.ProductZIA, name: resourceEUSAStatus}: newSingletonHandler(
			resourceEUSAStatus,
			ziaSDKShow(client, activation.GetEusaStatus),
			eusaStatusSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceAuthExemptedURLs}: newSingletonHandler(
			resourceAuthExemptedURLs,
			ziaSDKShow(client, userauthsettings.Get),
			structSourceRecord[userauthsettings.ExemptedUrls],
		),
		{product: resources.ProductZIA, name: resourceIntermediateCAs}: newListGetHandler(
			resourceIntermediateCAs,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]intermediatecacertificates.IntermediateCACertificate, error) {
				return getZIAAllPages[intermediatecacertificates.IntermediateCACertificate](ctx, service, "/zia/api/v1/intermediateCaCertificate")
			}),
			ziaSDKGet(client, intermediatecacertificates.GetCertificate),
			intermediateCACertificateSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceCloudAppInsts}: newListGetHandler(
			resourceCloudAppInsts,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]cloudappinstances.CloudApplicationInstances, error) {
				return getZIAAllPages[cloudappinstances.CloudApplicationInstances](ctx, service, "/zia/api/v1/cloudApplicationInstances")
			}),
			ziaSDKGet(client, cloudappinstances.Get),
			cloudAppInstanceSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceTenancyProfiles}: newListGetHandler(
			resourceTenancyProfiles,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]tenancyrestriction.TenancyRestrictionProfile, error) {
				return getZIAAllPages[tenancyrestriction.TenancyRestrictionProfile](ctx, service, "/zia/api/v1/tenancyRestrictionProfile")
			}),
			ziaSDKGet(client, tenancyrestriction.Get),
			tenancyRestrictionProfileSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceVZENClusters}: newListGetHandler(
			resourceVZENClusters,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]vzenclusters.VZENClusters, error) {
				return getZIAAllPages[vzenclusters.VZENClusters](ctx, service, "/zia/api/v1/virtualZenClusters")
			}),
			ziaSDKGet(client, vzenclusters.Get),
			vzenClusterSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceVZENNodes}: newListGetHandler(
			resourceVZENNodes,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]vzennodes.VZENNodes, error) {
				return getZIAAllPages[vzennodes.VZENNodes](ctx, service, "/zia/api/v1/virtualZenNodes")
			}),
			ziaSDKGet(client, vzennodes.Get),
			vzenNodeSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceLocationsLite}: newListOnlyHandler(
			resourceLocationsLite,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]ziaLiteLocation, error) {
				return getZIAAllPages[ziaLiteLocation](ctx, service, "/zia/api/v1/locations/lite")
			}),
			liteLocationSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceBrowserIsolation}: newListOnlyHandler(
			resourceBrowserIsolation,
			ziaSDKList(client, getZIABrowserIsolationAllPages),
			browserIsolationProfileSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDLPEngines}: newListGetHandler(
			resourceDLPEngines,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]dlpengines.DLPEngines, error) {
				return getZIAAllPages[dlpengines.DLPEngines](ctx, service, "/zia/api/v1/dlpEngines")
			}),
			ziaSDKGet(client, dlpengines.Get),
			dlpEngineSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDLPDictionaries}: newListGetHandler(
			resourceDLPDictionaries,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]dlpdictionaries.DlpDictionary, error) {
				return getZIAAllPages[dlpdictionaries.DlpDictionary](ctx, service, "/zia/api/v1/dlpDictionaries")
			}),
			ziaSDKGet(client, dlpdictionaries.Get),
			dlpDictionarySourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDLPEDMSchemas}: newListGetHandler(
			resourceDLPEDMSchemas,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]dlpexactdatamatch.DLPEDMSchema, error) {
				return getZIAAllPages[dlpexactdatamatch.DLPEDMSchema](ctx, service, "/zia/api/v1/dlpExactDataMatchSchemas")
			}),
			ziaSDKGet(client, dlpexactdatamatch.GetDLPEDMSchemaID),
			dlpEDMSchemaSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDLPEDMLite}: newListOnlyHandler(
			resourceDLPEDMLite,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]dlpedmlite.DLPEDMLite, error) {
				return getZIAAllPages[dlpedmlite.DLPEDMLite](ctx, service, "/zia/api/v1/dlpExactDataMatchSchemas/lite")
			}),
			dlpEDMLiteSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDLPIDMLite}: newListGetHandler(
			resourceDLPIDMLite,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]dlpidmprofilelite.DLPIDMProfileLite, error) {
				return getZIAAllPages[dlpidmprofilelite.DLPIDMProfileLite](ctx, service, "/zia/api/v1/idmprofile/lite")
			}),
			ziaSDKListGetByIntID(
				client,
				func(ctx context.Context, service *zsdk.Service) ([]dlpidmprofilelite.DLPIDMProfileLite, error) {
					return getZIAAllPages[dlpidmprofilelite.DLPIDMProfileLite](ctx, service, "/zia/api/v1/idmprofile/lite")
				},
				func(profile dlpidmprofilelite.DLPIDMProfileLite) int { return profile.ProfileID },
			),
			dlpIDMProfileLiteSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDLPIDMProfiles}: newListGetHandler(
			resourceDLPIDMProfiles,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]dlpidmprofiles.DLPIDMProfile, error) {
				return getZIAAllPages[dlpidmprofiles.DLPIDMProfile](ctx, service, "/zia/api/v1/idmprofile")
			}),
			ziaSDKGet(client, dlpidmprofiles.Get),
			dlpIDMProfileSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDLPWebRules}: newListGetHandler(
			resourceDLPWebRules,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]dlpwebrules.WebDLPRules, error) {
				return getZIAAllPages[dlpwebrules.WebDLPRules](ctx, service, "/zia/api/v1/webDlpRules")
			}),
			ziaSDKGet(client, dlpwebrules.Get),
			dlpWebRuleSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDLPICAPServers}: newListGetHandler(
			resourceDLPICAPServers,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]dlpicapservers.DLPICAPServers, error) {
				return getZIAAllPages[dlpicapservers.DLPICAPServers](ctx, service, "/zia/api/v1/icapServers")
			}),
			ziaSDKGet(client, dlpicapservers.Get),
			dlpICAPServerSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDLPIncidentRcvs}: newListGetHandler(
			resourceDLPIncidentRcvs,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]dlpincidentreceivers.IncidentReceiverServers, error) {
				return getZIAAllPages[dlpincidentreceivers.IncidentReceiverServers](ctx, service, "/zia/api/v1/incidentReceiverServers")
			}),
			ziaSDKGet(client, dlpincidentreceivers.Get),
			dlpIncidentReceiverSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDLPNotifyTmpls}: newListGetHandler(
			resourceDLPNotifyTmpls,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]dlpnotificationtemplates.DlpNotificationTemplates, error) {
				return getZIAAllPages[dlpnotificationtemplates.DlpNotificationTemplates](ctx, service, "/zia/api/v1/dlpNotificationTemplates")
			}),
			ziaSDKGet(client, dlpnotificationtemplates.Get),
			dlpNotificationTemplateSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceC2CIncidentRcvs}: newListGetHandler(
			resourceC2CIncidentRcvs,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]c2cincidentreceiver.C2CIncidentReceiver, error) {
				return getZIAAllPages[c2cincidentreceiver.C2CIncidentReceiver](ctx, service, "/zia/api/v1/cloudToCloudIR")
			}),
			ziaSDKGet(client, c2cincidentreceiver.Get),
			c2cIncidentReceiverSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceRiskProfiles}: newListGetHandler(
			resourceRiskProfiles,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]riskprofiles.RiskProfiles, error) {
				return getZIAAllPages[riskprofiles.RiskProfiles](ctx, service, "/zia/api/v1/riskProfiles")
			}),
			ziaSDKGet(client, riskprofiles.Get),
			riskProfileSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceNSSServers}: newListGetHandler(
			resourceNSSServers,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]nssservers.NSSServers, error) {
				return nssservers.GetAll(ctx, service, nil)
			}),
			ziaSDKGet(client, nssservers.Get),
			nssServerSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceNSSFeeds}: newListGetHandler(
			resourceNSSFeeds,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]cloudnss.NSSFeed, error) {
				return getZIAAllPages[cloudnss.NSSFeed](ctx, service, "/zia/api/v1/nssFeeds")
			}),
			ziaSDKGet(client, cloudnss.Get),
			nssFeedSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceFileTypeRules}: newListGetHandler(
			resourceFileTypeRules,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]filetypecontrol.FileTypeRules, error) {
				return getZIAAllPages[filetypecontrol.FileTypeRules](ctx, service, "/zia/api/v1/fileTypeRules")
			}),
			ziaSDKGet(client, filetypecontrol.Get),
			fileTypeRuleSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceSandboxRules}: newListGetHandler(
			resourceSandboxRules,
			ziaSDKList(client, sandboxrules.GetAll),
			ziaSDKGet(client, sandboxrules.Get),
			sandboxRuleSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceFirewallDNSRules}: newListGetHandler(
			resourceFirewallDNSRules,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]firewalldnscontrolpolicies.FirewallDNSRules, error) {
				return getZIAAllPages[firewalldnscontrolpolicies.FirewallDNSRules](ctx, service, "/zia/api/v1/firewallDnsRules")
			}),
			ziaSDKGet(client, firewalldnscontrolpolicies.Get),
			firewallDNSRuleSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceCustomFileTypes}: newListGetHandler(
			resourceCustomFileTypes,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]customfiletypes.CustomFileTypes, error) {
				return getZIAAllPages[customfiletypes.CustomFileTypes](ctx, service, "/zia/api/v1/customFileTypes")
			}),
			ziaSDKGet(client, customfiletypes.Get),
			customFileTypeSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceZPAGateways}: newListGetHandler(
			resourceZPAGateways,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpagateways.ZPAGateways, error) {
				return getZIAAllPages[zpagateways.ZPAGateways](ctx, service, "/zia/api/v1/zpaGateways")
			}),
			ziaSDKGet(client, zpagateways.Get),
			zpaGatewaySourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDCExclusions}: newListOnlyHandler(
			resourceDCExclusions,
			ziaSDKList(client, dcexclusions.GetAll),
			dcExclusionSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceSubClouds}: newListOnlyHandler(
			resourceSubClouds,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]subclouds.SubClouds, error) {
				return getZIAAllPages[subclouds.SubClouds](ctx, service, "/zia/api/v1/subclouds")
			}),
			subCloudSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceIPv6Config}: newSingletonHandler(
			resourceIPv6Config,
			ziaSDKShow(client, ipv6config.GetIPv6Config),
			ipv6ConfigSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceIPv6DNS64Prefix}: newListOnlyHandler(
			resourceIPv6DNS64Prefix,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]ipv6config.IPv6ConfigPrefix, error) {
				return getZIAAllPages[ipv6config.IPv6ConfigPrefix](ctx, service, "/zia/api/v1/ipv6config/dns64prefix")
			}),
			ipv6ConfigPrefixSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceIPv6NAT64Prefix}: newListOnlyHandler(
			resourceIPv6NAT64Prefix,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]ipv6config.IPv6ConfigPrefix, error) {
				return getZIAAllPages[ipv6config.IPv6ConfigPrefix](ctx, service, "/zia/api/v1/ipv6config/nat64prefix")
			}),
			ipv6ConfigPrefixSourceRecord,
		),
		{product: resources.ProductZIA, name: resourcePACFiles}: newListOnlyHandler(
			resourcePACFiles,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]pacfiles.PACFileConfig, error) {
				return getZIAAllPages[pacfiles.PACFileConfig](ctx, service, "/zia/api/v1/pacFiles")
			}),
			pacFileSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceCloudAppControl}: newListOnlyHandler(
			resourceCloudAppControl,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]cloudappcontrol.WebApplicationRules, error) {
				// Cloud App Control has no flat list endpoint; enumerate the rule
				// types and concatenate each type's rules (sorted for determinism).
				mapping, err := cloudappcontrol.GetRuleTypeMapping(ctx, service)
				if err != nil {
					return nil, err
				}
				// The mapping's direction (rule-type code vs display name) is not
				// guaranteed, so probe both keys and values; invalid entries are
				// skipped by the tolerant GetByRuleType loop below.
				seen := map[string]struct{}{}
				ruleTypes := make([]string, 0, len(mapping)*2)
				for k, v := range mapping {
					for _, candidate := range []string{k, v} {
						if candidate == "" {
							continue
						}
						if _, ok := seen[candidate]; ok {
							continue
						}
						seen[candidate] = struct{}{}
						ruleTypes = append(ruleTypes, candidate)
					}
				}
				if len(ruleTypes) == 0 {
					ruleTypes = cloudAppControlRuleTypes
				}
				sort.Strings(ruleTypes)
				var all []cloudappcontrol.WebApplicationRules
				for _, ruleType := range ruleTypes {
					rules, err := cloudappcontrol.GetByRuleType(ctx, service, ruleType)
					if err != nil {
						// A rule type can be unavailable or not entitled in a given
						// tenant; skip it rather than failing the whole list.
						continue
					}
					all = append(all, rules...)
				}
				return all, nil
			}),
			cloudAppControlSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceCloudAppPolicy}: newListOnlyHandler(
			resourceCloudAppPolicy,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]cloudapplications.CloudApplications, error) {
				return getZIAAllPages[cloudapplications.CloudApplications](ctx, service, "/zia/api/v1/cloudApplications/policy")
			}),
			cloudApplicationPolicySourceRecord,
		),
		{product: resources.ProductZIA, name: resourceCloudAppSSLPol}: newListOnlyHandler(
			resourceCloudAppSSLPol,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]cloudapplications.CloudApplications, error) {
				return getZIAAllPages[cloudapplications.CloudApplications](ctx, service, "/zia/api/v1/cloudApplications/sslPolicy")
			}),
			cloudApplicationPolicySourceRecord,
		),
		{product: resources.ProductZIA, name: resourceDomainProfiles}: newListOnlyHandler(
			resourceDomainProfiles,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]saassecurityapi.DomainProfiles, error) {
				return getZIAAllPages[saassecurityapi.DomainProfiles](ctx, service, "/zia/api/v1/domainProfiles")
			}),
			domainProfileSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceCASBTombstones}: newListOnlyHandler(
			resourceCASBTombstones,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]saassecurityapi.QuarantineTombstoneLite, error) {
				return getZIAAllPages[saassecurityapi.QuarantineTombstoneLite](ctx, service, "/zia/api/v1/quarantineTombstoneTemplate/lite")
			}),
			casbTombstoneTemplateSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceCASBEmailLabels}: newListOnlyHandler(
			resourceCASBEmailLabels,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]saassecurityapi.CasbEmailLabel, error) {
				return getZIAAllPages[saassecurityapi.CasbEmailLabel](ctx, service, "/zia/api/v1/casbEmailLabel/lite")
			}),
			casbEmailLabelSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceCASBTenants}: newListOnlyHandler(
			resourceCASBTenants,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]saassecurityapi.CasbTenants, error) {
				return getZIAAllPages[saassecurityapi.CasbTenants](ctx, service, "/zia/api/v1/casbTenant/lite")
			}),
			casbTenantSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceCASBDLPRules}: newListOnlyHandler(
			resourceCASBDLPRules,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]casbdlprules.CasbDLPRules, error) {
				return getZIAAllPages[casbdlprules.CasbDLPRules](ctx, service, "/zia/api/v1/casbDlpRules/all")
			}),
			casbDLPRuleSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceCASBMalwareRules}: newListOnlyHandler(
			resourceCASBMalwareRules,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]casbmalwarerules.CasbMalwareRules, error) {
				return getZIAAllPages[casbmalwarerules.CasbMalwareRules](ctx, service, "/zia/api/v1/casbMalwareRules/all")
			}),
			casbMalwareRuleSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceBrowserControl}: newSingletonHandler(
			resourceBrowserControl,
			ziaSDKShow(client, browsercontrolsettings.GetBrowserControlSettings),
			browserControlSettingsSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceSupportedBrowsers}: newListOnlyHandler(
			resourceSupportedBrowsers,
			ziaSDKList(client, securebrowsing.GetSupportedBrowserVersions),
			structSourceRecord[securebrowsing.SupportedBrowserVersion],
		),
		{product: resources.ProductZIA, name: resourceFTPControl}: newSingletonHandler(
			resourceFTPControl,
			ziaSDKShow(client, ftpcontrolpolicy.GetFTPControlPolicy),
			structSourceRecord[ftpcontrolpolicy.FTPControlPolicy],
		),
		{product: resources.ProductZIA, name: resourceRemoteAssistance}: newSingletonHandler(
			resourceRemoteAssistance,
			ziaSDKShow(client, remoteassistance.GetRemoteAssistance),
			structSourceRecord[remoteassistance.RemoteAssistance],
		),
		{product: resources.ProductZIA, name: resourceAdminUsers}: newListGetHandler(
			resourceAdminUsers,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]ziaadminusers.AdminUsers, error) {
				return getZIAAllPages[ziaadminusers.AdminUsers](
					ctx,
					service,
					"/zia/api/v1/adminUsers?includeAuditorUsers=true&includeAdminUsers=true",
				)
			}),
			ziaSDKGet(client, ziaadminusers.GetAdminUsers),
			ziaAdminUserSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceAdminRoles}: newListGetHandler(
			resourceAdminRoles,
			ziaSDKList(client, ziaadminroles.GetAllAdminRoles),
			ziaSDKGet(client, ziaadminroles.Get),
			ziaAdminRoleSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceEmailProfiles}: newListGetHandler(
			resourceEmailProfiles,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]emailprofiles.EmailProfiles, error) {
				return getZIAAllPages[emailprofiles.EmailProfiles](ctx, service, "/zia/api/v1/emailRecipientProfile")
			}),
			ziaSDKGet(client, emailprofiles.Get),
			emailProfileSourceRecord,
		),
		{product: resources.ProductZIA, name: resourceAdvancedSettings}: newSingletonHandler(
			resourceAdvancedSettings,
			ziaSDKShow(client, advancedsettings.GetAdvancedSettings),
			structSourceRecord[advancedsettings.AdvancedSettings],
		),
		{product: resources.ProductZIA, name: resourceAdvancedThreatSettings}: newSingletonHandler(
			resourceAdvancedThreatSettings,
			ziaSDKShow(client, advancedthreatsettings.GetAdvancedThreatSettings),
			structSourceRecord[advancedthreatsettings.AdvancedThreatSettings],
		),
		{product: resources.ProductZIA, name: resourceMobileThreatSettings}: newSingletonHandler(
			resourceMobileThreatSettings,
			ziaSDKShow(client, mobilethreatsettings.GetMobileThreatSettings),
			structSourceRecord[mobilethreatsettings.MobileAdvanceThreatSettings],
		),
		{product: resources.ProductZIA, name: resourceSandboxSettings}: newSingletonHandler(
			resourceSandboxSettings,
			ziaSDKShow(client, sandboxsettings.Get),
			structSourceRecord[sandboxsettings.BaAdvancedSettings],
		),
		{product: resources.ProductZIA, name: resourceEndUserNotification}: newSingletonHandler(
			resourceEndUserNotification,
			ziaSDKShow(client, endusernotification.GetUserNotificationSettings),
			structSourceRecord[endusernotification.UserNotificationSettings],
		),
		{product: resources.ProductZIA, name: resourceOrgInformation}: newSingletonHandler(
			resourceOrgInformation,
			ziaSDKShow(client, organizationdetails.GetOrgInformation),
			structSourceRecord[organizationdetails.Organization],
		),
		{product: resources.ProductZIA, name: resourceATPMalwarePolicy}: newSingletonHandler(
			resourceATPMalwarePolicy,
			ziaSDKShow(client, malwareprotection.GetATPMalwarePolicy),
			structSourceRecord[malwareprotection.MalwarePolicy],
		),
		{product: resources.ProductZIA, name: resourceATPMalwareSettings}: newSingletonHandler(
			resourceATPMalwareSettings,
			ziaSDKShow(client, malwareprotection.GetATPMalwareSettings),
			structSourceRecord[malwareprotection.MalwareSettings],
		),
		{product: resources.ProductZIA, name: resourceATPMalwareInspection}: newSingletonHandler(
			resourceATPMalwareInspection,
			ziaSDKShow(client, malwareprotection.GetATPMalwareInspection),
			structSourceRecord[malwareprotection.ATPMalwareInspection],
		),
		{product: resources.ProductZIA, name: resourceATPMalwareProtocols}: newSingletonHandler(
			resourceATPMalwareProtocols,
			ziaSDKShow(client, malwareprotection.GetATPMalwareProtocols),
			structSourceRecord[malwareprotection.ATPMalwareProtocols],
		),
		{product: resources.ProductZIA, name: resourceMaliciousURLs}: newSingletonHandler(
			resourceMaliciousURLs,
			ziaSDKShow(client, advancedthreatsettings.GetMaliciousURLs),
			structSourceRecord[advancedthreatsettings.MaliciousURLs],
		),
		{product: resources.ProductZIA, name: resourceSecurityExceptions}: newSingletonHandler(
			resourceSecurityExceptions,
			ziaSDKShow(client, advancedthreatsettings.GetSecurityExceptions),
			structSourceRecord[advancedthreatsettings.SecurityExceptions],
		),
		{product: resources.ProductZIA, name: resourceSecurityPolicyURLAllowlist}: newSingletonHandler(
			resourceSecurityPolicyURLAllowlist,
			ziaSDKShow(client, securitypolicysettings.GetWhiteListUrls),
			structSourceRecord[securitypolicysettings.ListUrls],
		),
		{product: resources.ProductZIA, name: resourceSecurityPolicyURLDenylist}: newSingletonHandler(
			resourceSecurityPolicyURLDenylist,
			ziaSDKShow(client, securitypolicysettings.GetBlackListUrls),
			structSourceRecord[securitypolicysettings.ListUrls],
		),
		{product: resources.ProductZIA, name: resourceDNSAppGroups}: newListGetHandler(
			resourceDNSAppGroups,
			ziaSDKList(client, dnsappgroups.GetAll),
			ziaSDKGet(client, dnsappgroups.Get),
			structSourceRecord[dnsappgroups.DnsApplicationGroup],
		),
		{product: resources.ProductZIA, name: resourceEndpointDLPRules}: newListGetHandler(
			resourceEndpointDLPRules,
			ziaSDKList(client, endpointdlprules.GetAll),
			ziaSDKGet(client, endpointdlprules.Get),
			structSourceRecord[endpointdlprules.EndpointDlpRules],
		),
		{product: resources.ProductZIA, name: resourceEmailDLPRules}: newListGetHandler(
			resourceEmailDLPRules,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]emaildlprules.OutboundEmailDlp, error) {
				return emaildlprules.GetAll(ctx, service, nil)
			}),
			ziaSDKGet(client, emaildlprules.Get),
			structSourceRecord[emaildlprules.OutboundEmailDlp],
		),
		{product: resources.ProductZIA, name: resourceAlertDefinitions}: newListGetHandler(
			resourceAlertDefinitions,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]alertdefinitions.AlertDefinitions, error) {
				return getZIAAllPages[alertdefinitions.AlertDefinitions](ctx, service, "/zia/api/v1/alertDefinitions")
			}),
			ziaSDKGet(client, alertdefinitions.Get),
			structSourceRecord[alertdefinitions.AlertDefinitions],
		),
		{product: resources.ProductZIA, name: resourceHTTPHeaderProfiles}: newListOnlyHandler(
			resourceHTTPHeaderProfiles,
			ziaSDKList(client, httpheaderprofile.GetAll),
			structSourceRecord[httpheaderprofile.HttpHeaderProfile],
		),
		{product: resources.ProductZIA, name: resourceHTTPHeaderActions}: newListOnlyHandler(
			resourceHTTPHeaderActions,
			ziaSDKList(client, httpheaderactionprofile.GetAll),
			structSourceRecord[httpheaderactionprofile.HttpHeaderActionProfile],
		),
		{product: resources.ProductZIA, name: resourceAdaptiveAccess}: newListOnlyHandler(
			resourceAdaptiveAccess,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]adaptiveaccess.AdaptiveAccess, error) {
				return getZIAAllPages[adaptiveaccess.AdaptiveAccess](ctx, service, "/zia/api/v1/adaptiveAccessProfiles")
			}),
			structSourceRecord[adaptiveaccess.AdaptiveAccess],
		),
		{product: resources.ProductZIA, name: resourceDLPGlobalOptions}: newSingletonHandler(
			resourceDLPGlobalOptions,
			ziaSDKShow(client, dlpglobaloptions.GetDLPGlobalOptions),
			structSourceRecord[dlpglobaloptions.WebDlpGlobal],
		),
	}
	for k, v := range entries {
		addHandler(m, k, v)
	}
	addOtherScoutHandlers(m, client)
}

func ipsSignatureRuleSourceRecord(rule ipssignaturerules.IPSSignatureRules) resources.SourceRecord {
	fields := map[string]any{
		"id":                         rule.ID,
		"name":                       rule.Name,
		"description":                rule.Description,
		"enabled":                    rule.Enabled,
		"deleted":                    rule.Deleted,
		"promoteTime":                rule.PromoteTime,
		"ruleTextModTime":            rule.RuleTextModTime,
		"dynamicValidationSubmitted": rule.DynamicValidationSubmitted,
		"dynamicValidationRejected":  rule.DynamicValidationRejected,
		"dynamicValidationSucceeded": rule.DynamicValidationSucceeded,
		"disabledFromZSCM":           rule.DisabledFromZSCM,
		"dynamicValRejectCode":       rule.DynamicValRejectCode,
	}
	// ruleText is deliberately never mapped: the Suricata/Snort signature body
	// embeds tenant network indicators, so it is not in the catalog either.
	if rule.Category != nil {
		fields["category"] = map[string]any{
			"id":            rule.Category.ID,
			"name":          rule.Category.Name,
			"isNameL10nTag": rule.Category.IsNameL10nTag,
		}
	}
	return resources.NewSourceRecord(fields)
}

// cloudAppControlRuleTypes is a fallback list of Cloud App Control rule-type
// codes used only when the live ruleTypeMapping endpoint returns nothing.
var cloudAppControlRuleTypes = []string{
	"AI_ML",
	"BUSINESS_PRODUCTIVITY",
	"CONSUMER",
	"CUSTOM_CAPP",
	"DNS_OVER_HTTPS",
	"ENTERPRISE_COLLABORATION",
	"FILE_SHARE",
	"FINANCE",
	"HEALTH_CARE",
	"HOSTING_PROVIDER",
	"HUMAN_RESOURCES",
	"INSTANT_MESSAGING",
	"IT_SERVICES",
	"LEGAL",
	"SALES_AND_MARKETING",
	"SOCIAL_NETWORKING",
	"STREAMING_MEDIA",
	"SYSTEM_AND_DEVELOPMENT",
	"WEBMAIL",
	"WEB_CONFERENCING",
}

func cloudAppControlSourceRecord(rule cloudappcontrol.WebApplicationRules) resources.SourceRecord {
	fields := map[string]any{
		"id":                   rule.ID,
		"name":                 rule.Name,
		"description":          rule.Description,
		"state":                rule.State,
		"rank":                 rule.Rank,
		"type":                 rule.Type,
		"order":                rule.Order,
		"timeQuota":            rule.TimeQuota,
		"sizeQuota":            rule.SizeQuota,
		"cascadingEnabled":     rule.CascadingEnabled,
		"accessControl":        rule.AccessControl,
		"numberOfApplications": rule.NumberOfApplications,
		"promptCaptureEnabled": rule.PromptCaptureEnabled,
		"eunEnabled":           rule.EunEnabled,
		"eunTemplateId":        rule.EunTemplateID,
		"browserEunTemplateId": rule.BrowserEunTemplateID,
		"predefined":           rule.Predefined,
		"validityStartTime":    rule.ValidityStartTime,
		"validityEndTime":      rule.ValidityEndTime,
		"validityTimeZoneId":   rule.ValidityTimeZoneID,
		"lastModifiedTime":     rule.LastModifiedTime,
		"enforceTimeValidity":  rule.EnforceTimeValidity,
	}
	addStringSlice(fields, "actions", rule.Actions)
	addStringSlice(fields, "applications", rule.Applications)
	addStringSlice(fields, "userAgentTypes", rule.UserAgentTypes)
	addStringSlice(fields, "deviceTrustLevels", rule.DeviceTrustLevels)
	addStringSlice(fields, "userRiskScoreLevels", rule.UserRiskScoreLevels)
	addIDNameExtensionsSlice(fields, "labels", rule.Labels)
	addIDNameExtensionsSlice(fields, "timeWindows", rule.TimeWindows)
	addIDNameExtensionsSlice(fields, "locations", rule.Locations)
	addIDNameExtensionsSlice(fields, "locationGroups", rule.LocationGroups)
	addIDNameExtensionsSlice(fields, "tenancyProfileIds", rule.TenancyProfileIDs)
	addIDNameExtensionsSlice(fields, "departments", rule.Departments)
	addIDNameExtensionsSlice(fields, "groups", rule.Groups)
	addIDNameExtensionsSlice(fields, "users", rule.Users)
	addIDNameExtensionsSlice(fields, "deviceGroups", rule.DeviceGroups)
	addIDNameExtensionsSlice(fields, "devices", rule.Devices)
	addIDCustomPtr(fields, "cloudAppRiskProfile", rule.CloudAppRiskProfile)
	if len(rule.CloudAppInstances) > 0 {
		fields["cloudAppInstances"] = cloudAppControlInstancesSource(rule.CloudAppInstances)
	}
	if rule.CBIProfile != (cloudappcontrol.CBIProfile{}) {
		fields["cbiProfile"] = cloudAppControlCBIProfileSource(rule.CBIProfile)
	}
	return resources.NewSourceRecord(fields)
}

// cloudAppControlCBIProfileSource maps the cloudappcontrol-local CBIProfile
// struct (a superset of ziacommon.CBIProfile with defaultProfile/sandboxMode).
// The url value still reaches the source record but the catalog classifies it
// as a secret, so projection drops it in every mode.
func cloudAppControlCBIProfileSource(value cloudappcontrol.CBIProfile) map[string]any {
	return map[string]any{
		"id":             value.ID,
		"name":           value.Name,
		"url":            value.URL,
		"profileSeq":     value.ProfileSeq,
		"defaultProfile": value.DefaultProfile,
		"sandboxMode":    value.SandboxMode,
	}
}

func cloudAppControlInstancesSource(values []cloudappcontrol.CloudAppInstances) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, map[string]any{
			"id":   value.ID,
			"name": value.Name,
			"type": value.Type,
		})
	}
	return out
}

// ziaLiteLocation is one /locations/lite object, modeled on the keys a live
// read returned. The SDK's locationlite.LocationLite is not used: its plain
// booleans would turn omitted keys into false, and it lacks keys the API
// returns (ecLocation, digestAuthEnabled, kerberosAuth, iotDiscoveryEnabled).
// Keys it has that no live read returned (ipv6Enabled, other6SubLocation,
// subLocScope*, subLocAccIds) are not read. Booleans are pointers so a key
// the API omits stays absent.
type ziaLiteLocation struct {
	ID                                  int64  `json:"id"`
	Name                                string `json:"name,omitempty"`
	ParentID                            *int64 `json:"parentId,omitempty"`
	TZ                                  string `json:"tz,omitempty"`
	ECLocation                          *bool  `json:"ecLocation,omitempty"`
	OtherSubLocation                    *bool  `json:"otherSubLocation,omitempty"`
	AUPBlockInternetUntilAccepted       *bool  `json:"aupBlockInternetUntilAccepted,omitempty"`
	AUPEnabled                          *bool  `json:"aupEnabled,omitempty"`
	AUPForceSSLInspection               *bool  `json:"aupForceSslInspection,omitempty"`
	CautionEnabled                      *bool  `json:"cautionEnabled,omitempty"`
	DigestAuthEnabled                   *bool  `json:"digestAuthEnabled,omitempty"`
	IOTDiscoveryEnabled                 *bool  `json:"iotDiscoveryEnabled,omitempty"`
	IPSControl                          *bool  `json:"ipsControl,omitempty"`
	KerberosAuth                        *bool  `json:"kerberosAuth,omitempty"`
	OFWEnabled                          *bool  `json:"ofwEnabled,omitempty"`
	SurrogateIP                         *bool  `json:"surrogateIP,omitempty"`
	SurrogateIPEnforcedForKnownBrowsers *bool  `json:"surrogateIPEnforcedForKnownBrowsers,omitempty"`
	XFFForwardEnabled                   *bool  `json:"xffForwardEnabled,omitempty"`
	ZappSSLScanEnabled                  *bool  `json:"zappSslScanEnabled,omitempty"`
}

func liteLocationSourceRecord(location ziaLiteLocation) resources.SourceRecord {
	fields := map[string]any{"id": location.ID}
	if location.Name != "" {
		fields["name"] = location.Name
	}
	if location.TZ != "" {
		fields["tz"] = location.TZ
	}
	if location.ParentID != nil {
		fields["parentId"] = *location.ParentID
	}
	for key, value := range map[string]*bool{
		"ecLocation":                          location.ECLocation,
		"otherSubLocation":                    location.OtherSubLocation,
		"aupBlockInternetUntilAccepted":       location.AUPBlockInternetUntilAccepted,
		"aupEnabled":                          location.AUPEnabled,
		"aupForceSslInspection":               location.AUPForceSSLInspection,
		"cautionEnabled":                      location.CautionEnabled,
		"digestAuthEnabled":                   location.DigestAuthEnabled,
		"iotDiscoveryEnabled":                 location.IOTDiscoveryEnabled,
		"ipsControl":                          location.IPSControl,
		"kerberosAuth":                        location.KerberosAuth,
		"ofwEnabled":                          location.OFWEnabled,
		"surrogateIP":                         location.SurrogateIP,
		"surrogateIPEnforcedForKnownBrowsers": location.SurrogateIPEnforcedForKnownBrowsers,
		"xffForwardEnabled":                   location.XFFForwardEnabled,
		"zappSslScanEnabled":                  location.ZappSSLScanEnabled,
	} {
		if value != nil {
			fields[key] = *value
		}
	}
	return resources.NewSourceRecord(fields)
}
