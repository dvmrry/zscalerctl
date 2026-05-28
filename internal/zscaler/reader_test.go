package zscaler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	ziacommon "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/common"
	"github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/location/locationmanagement"
	rulelabels "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/rule_labels"

	"github.com/dvmrry/zscalerctl/internal/redact"
	"github.com/dvmrry/zscalerctl/internal/resources"
	"github.com/dvmrry/zscalerctl/internal/secret"
)

func TestNewReaderRequiresExplicitZscalerctlCredentials(t *testing.T) {
	t.Parallel()

	if _, err := NewReader(ReaderConfig{}); !errors.Is(err, ErrMissingCredentials) {
		t.Fatalf("NewReader(empty) error = %v, want ErrMissingCredentials", err)
	}
}

func TestNewReaderIgnoresSDKEnvironmentNames(t *testing.T) {
	t.Setenv("ZSCALER_CLIENT_ID", "sdk-client-id")
	t.Setenv("ZSCALER_CLIENT_SECRET", "sdk-client-secret")
	t.Setenv("ZSCALER_VANITY_DOMAIN", "sdk-vanity")

	if _, err := NewReader(ReaderConfig{}); !errors.Is(err, ErrMissingCredentials) {
		t.Fatalf("NewReader(empty with SDK env) error = %v, want ErrMissingCredentials", err)
	}
}

func TestNewSDKConfigurationDoesNotUseSDKDiscoveryOrLogging(t *testing.T) {
	t.Setenv("ZSCALER_CLIENT_ID", "sdk-client-id")
	t.Setenv("ZSCALER_CLIENT_SECRET", "sdk-client-secret")
	t.Setenv("ZSCALER_VANITY_DOMAIN", "sdk-vanity")
	t.Setenv("ZSCALER_CLOUD", "sdk-cloud")
	t.Setenv("ZSCALER_CLIENT_PROXY_HOST", "sdk-proxy.example.invalid")
	t.Setenv("ZSCALER_CLIENT_CACHE_ENABLED", "true")
	t.Setenv("ZSCALER_SDK_LOG", "true")
	t.Setenv("ZSCALER_SDK_VERBOSE", "true")
	t.Setenv("HTTPS_PROXY", "http://standard-proxy.example.invalid:8080")

	cfg := newSDKConfiguration(context.Background(), validReaderConfig())
	if got := cfg.Zscaler.Client.ClientID; got != "zscalerctl-client-id" {
		t.Errorf("newSDKConfiguration().ClientID = %q, want zscalerctl-client-id", got)
	}
	if got := cfg.Zscaler.Client.ClientSecret; got != "zscalerctl-client-secret" {
		t.Errorf("newSDKConfiguration().ClientSecret = %q, want zscalerctl-client-secret", got)
	}
	if got := cfg.Zscaler.Client.VanityDomain; got != "zscalerctl-vanity" {
		t.Errorf("newSDKConfiguration().VanityDomain = %q, want zscalerctl-vanity", got)
	}
	if got := cfg.Zscaler.Client.Cloud; got != "" {
		t.Errorf("newSDKConfiguration().Cloud = %q, want empty", got)
	}
	if got := cfg.Zscaler.Client.Proxy.Host; got != "" {
		t.Errorf("newSDKConfiguration().Proxy.Host = %q, want empty", got)
	}
	if cfg.Zscaler.Client.Cache.Enabled {
		t.Errorf("newSDKConfiguration().Cache.Enabled = true, want false")
	}
	if cfg.Debug {
		t.Errorf("newSDKConfiguration().Debug = true, want false")
	}
	transport, ok := cfg.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("newSDKConfiguration().HTTPClient.Transport = %T, want *http.Transport", cfg.HTTPClient.Transport)
	}
	if transport.Proxy != nil {
		t.Errorf("newSDKConfiguration().HTTPClient.Transport.Proxy is non-nil, want nil")
	}
}

func TestReaderListLocationsProjectsSDKShapeThroughAllowList(t *testing.T) {
	t.Parallel()

	const (
		psk               = "plain-raw-sdk-psk-canary"
		bareFreeTextToken = "A7b9C2d4E6f8G1h3J5k7L9m2N4p6Q8r0S2t4U6v"
	)
	reader := &SDKReader{
		cfg: validReaderConfig(),
		handlers: map[resourceKey]resourceHandler{
			{product: resources.ProductZIA, name: resourceLocations}: ziaLocationsHandler{
				client: fakeZIALocationClient{
					locations: []locationmanagement.Locations{
						{
							ID:          123,
							Name:        "HQ",
							IPAddresses: []string{"192.0.2.10"},
							Description: "temporary psk=" + psk + " " + bareFreeTextToken,
							VPNCredentials: []locationmanagement.VPNCredentials{
								{
									ID:           456,
									Type:         "UFQDN",
									FQDN:         "hq@example.invalid",
									PreSharedKey: psk,
								},
							},
						},
					},
				},
			},
		},
	}

	records, err := reader.List(context.Background(), resources.ProductZIA, "locations")
	if err != nil {
		t.Fatalf("SDKReader.List(zia, locations) error = %v, want nil", err)
	}
	spec, ok := resources.FindSpec(resources.ProductZIA, "locations")
	if !ok {
		t.Fatal("FindSpec(zia, locations) ok = false, want true")
	}
	projected, _, err := resources.ProjectRecords(spec, redact.ModeStandard, records)
	if err != nil {
		t.Fatalf("ProjectRecords(zia locations) error = %v, want nil", err)
	}
	got := projected.Records()[0].Fields()
	if strings.Contains(toString(got["description"]), psk) {
		t.Errorf("projected description = %v, want no %q", got["description"], psk)
	}
	if strings.Contains(toString(got["description"]), bareFreeTextToken) {
		t.Errorf("projected description = %v, want no bare token", got["description"])
	}
	if _, ok := got["vpnCredentials"]; ok {
		t.Errorf("projected record = %#v, want no vpnCredentials", got)
	}
	if err := resources.AssertRenderedSubset(spec, redact.ModeStandard, got); err != nil {
		t.Errorf("AssertRenderedSubset(projected SDK shape) error = %v, want nil", err)
	}
}

func TestReaderListRuleLabelsProjectsSDKShapeThroughAllowList(t *testing.T) {
	t.Parallel()

	const (
		canary            = "rule-label-psk-canary"
		adminCanary       = "rule-label-admin-canary"
		bareFreeTextToken = "A7b9C2d4E6f8G1h3J5k7L9m2N4p6Q8r0S2t4U6v"
	)
	reader := &SDKReader{
		cfg: validReaderConfig(),
		handlers: map[resourceKey]resourceHandler{
			{product: resources.ProductZIA, name: resourceRuleLabels}: ziaRuleLabelsHandler{
				client: fakeZIARuleLabelsClient{
					labels: []rulelabels.RuleLabels{
						{
							ID:                  789,
							Name:                "Outbound psk=" + canary,
							Description:         "temporary psk=" + canary + " " + bareFreeTextToken,
							LastModifiedTime:    1712345678,
							ReferencedRuleCount: 3,
							CreatedBy: &ziacommon.IDNameExtensions{
								ID:   1001,
								Name: adminCanary,
							},
							LastModifiedBy: &ziacommon.IDNameExtensions{
								ID:   1002,
								Name: adminCanary,
							},
						},
					},
				},
			},
		},
	}

	records, err := reader.List(context.Background(), resources.ProductZIA, "rule-labels")
	if err != nil {
		t.Fatalf("SDKReader.List(zia, rule-labels) error = %v, want nil", err)
	}
	spec, ok := resources.FindSpec(resources.ProductZIA, "rule-labels")
	if !ok {
		t.Fatal("FindSpec(zia, rule-labels) ok = false, want true")
	}
	projected, _, err := resources.ProjectRecords(spec, redact.ModeStandard, records)
	if err != nil {
		t.Fatalf("ProjectRecords(zia rule-labels) error = %v, want nil", err)
	}
	got := projected.Records()[0].Fields()
	for _, field := range []string{"name", "description"} {
		value := toString(got[field])
		if strings.Contains(value, canary) {
			t.Errorf("projected rule-labels %s = %v, want no %q", field, got[field], canary)
		}
		if field == "description" && strings.Contains(value, bareFreeTextToken) {
			t.Errorf("projected rule-labels %s = %v, want no bare token", field, got[field])
		}
		if !strings.Contains(value, "<REDACTED:SECRET>") {
			t.Errorf("projected rule-labels %s = %v, want typed redaction marker", field, got[field])
		}
	}
	for _, field := range []string{"createdBy", "lastModifiedBy"} {
		if _, ok := got[field]; ok {
			t.Errorf("projected rule-labels = %#v, want no %s", got, field)
		}
	}
	if strings.Contains(fmt.Sprint(got), adminCanary) {
		t.Errorf("projected rule-labels = %#v, want no %q", got, adminCanary)
	}
	if err := resources.AssertRenderedSubset(spec, redact.ModeStandard, got); err != nil {
		t.Errorf("AssertRenderedSubset(projected rule-labels SDK shape) error = %v, want nil", err)
	}
}

func TestReaderGetLocationRejectsNonNumericID(t *testing.T) {
	t.Parallel()

	reader := &SDKReader{
		cfg: validReaderConfig(),
		handlers: map[resourceKey]resourceHandler{
			{product: resources.ProductZIA, name: resourceLocations}: ziaLocationsHandler{
				client: fakeZIALocationClient{},
			},
		},
	}

	_, err := reader.Get(context.Background(), resources.ProductZIA, "locations", "not-a-number")
	if !errors.Is(err, ErrInvalidResourceID) {
		t.Fatalf("SDKReader.Get(non-numeric id) error = %v, want ErrInvalidResourceID", err)
	}
}

func TestReaderGetRuleLabelDispatchesByResource(t *testing.T) {
	t.Parallel()

	reader := &SDKReader{
		cfg: validReaderConfig(),
		handlers: map[resourceKey]resourceHandler{
			{product: resources.ProductZIA, name: resourceRuleLabels}: ziaRuleLabelsHandler{
				client: fakeZIARuleLabelsClient{
					label: &rulelabels.RuleLabels{
						ID:                  789,
						Name:                "Outbound",
						ReferencedRuleCount: 3,
					},
				},
			},
		},
	}

	record, err := reader.Get(context.Background(), resources.ProductZIA, "rule-labels", "789")
	if err != nil {
		t.Fatalf("SDKReader.Get(zia, rule-labels, 789) error = %v, want nil", err)
	}
	spec, ok := resources.FindSpec(resources.ProductZIA, "rule-labels")
	if !ok {
		t.Fatal("FindSpec(zia, rule-labels) ok = false, want true")
	}
	projected, _, err := resources.ProjectRecord(spec, redact.ModeStandard, record)
	if err != nil {
		t.Fatalf("ProjectRecord(zia rule-labels) error = %v, want nil", err)
	}
	got := projected.Fields()
	if got["id"] != 789 {
		t.Errorf("projected rule-label id = %v, want 789", got["id"])
	}
	if got["name"] != "Outbound" {
		t.Errorf("projected rule-label name = %v, want Outbound", got["name"])
	}
	if err := resources.AssertRenderedSubset(spec, redact.ModeStandard, got); err != nil {
		t.Errorf("AssertRenderedSubset(projected rule label) error = %v, want nil", err)
	}
}

func TestReaderUnsupportedResourceFailsClosed(t *testing.T) {
	t.Parallel()

	reader := &SDKReader{cfg: validReaderConfig()}

	_, err := reader.List(context.Background(), resources.ProductZPA, "applications")
	if !errors.Is(err, ErrUnsupportedResource) {
		t.Fatalf("SDKReader.List(zpa, applications) error = %v, want ErrUnsupportedResource", err)
	}
}

func TestReaderNormalizesSDKErrors(t *testing.T) {
	t.Parallel()

	const leaked = "sdk-client-secret"
	reader := &SDKReader{
		cfg: validReaderConfig(),
		handlers: map[resourceKey]resourceHandler{
			{product: resources.ProductZIA, name: resourceLocations}: ziaLocationsHandler{
				client: fakeZIALocationClient{
					err: errors.New("raw SDK error containing " + leaked),
				},
			},
		},
	}

	_, err := reader.List(context.Background(), resources.ProductZIA, "locations")
	if !errors.Is(err, ErrLiveAccessFailed) {
		t.Fatalf("SDKReader.List(error) error = %v, want ErrLiveAccessFailed", err)
	}
	if strings.Contains(err.Error(), leaked) {
		t.Errorf("SDKReader.List(error) error = %q, want no leaked SDK error content", err.Error())
	}
}

func validReaderConfig() ReaderConfig {
	return ReaderConfig{
		ClientID:     secret.New("zscalerctl-client-id"),
		ClientSecret: secret.New("zscalerctl-client-secret"),
		VanityDomain: "zscalerctl-vanity",
		Timeout:      time.Second,
	}
}

type fakeZIALocationClient struct {
	locations []locationmanagement.Locations
	location  *locationmanagement.Locations
	err       error
}

func (f fakeZIALocationClient) ListLocations(context.Context) ([]locationmanagement.Locations, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.locations, nil
}

func (f fakeZIALocationClient) GetLocation(context.Context, int) (*locationmanagement.Locations, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.location, nil
}

type fakeZIARuleLabelsClient struct {
	labels []rulelabels.RuleLabels
	label  *rulelabels.RuleLabels
	err    error
}

func (f fakeZIARuleLabelsClient) ListRuleLabels(context.Context) ([]rulelabels.RuleLabels, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.labels, nil
}

func (f fakeZIARuleLabelsClient) GetRuleLabel(context.Context, int) (*rulelabels.RuleLabels, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.label, nil
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	text, _ := value.(string)
	return text
}
