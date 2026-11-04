package zscaler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	zsdk "github.com/zscaler/zscaler-sdk-go/v3/zscaler"

	"zscalerctl/internal/redact"
	"zscalerctl/internal/resources"
)

const zpaPolicyRulesPathPrefix = "/zpa/mgmtconfig/v1/admin/customers/zscalerctl-zpa-customer-id/policySet/rules/policyType/"

// Canaries in the fake policy rule: operand values that name people, free
// text, and nested shapes the catalog does not render.
const (
	policyRuleSCIMCanary     = "finance-department-canary"
	policyRuleOperandCanary  = "operand-name-canary"
	policyRuleFreeTextCanary = "psk=policy-rule-canary-value"
	policyRuleServerCanary   = "192.0.2.77"
	policyRuleDesktopCanary  = "desktop-image-canary"
	policyRulePostCanary     = "post-action-canary"
)

func fakePolicyRule(policyType string) string {
	return `{"id":"rule-1","name":"rule","policyType":"` + policyType + `",` +
		`"description":"` + policyRuleFreeTextCanary + `","customMsg":"` + policyRuleFreeTextCanary + `",` +
		`"action":"ALLOW","ruleOrder":"1","operator":"AND",` +
		`"conditions":[{"id":"c-1","operator":"OR","negated":false,"operands":[` +
		`{"id":"o-1","objectType":"SCIM","lhs":"scim-attr-1","idpId":"idp-1","rhs":"` + policyRuleSCIMCanary + `","name":"` + policyRuleOperandCanary + `"},` +
		`{"id":"o-2","objectType":"APP","lhs":"id","rhs":"app-1"}]}],` +
		`"appServerGroups":[{"id":"sg-1","name":"server-group","servers":[{"id":"s-1","address":"` + policyRuleServerCanary + `"}],"applications":[{"id":"app-1","name":"app"}]}],` +
		`"desktopPolicyMappings":[{"id":"d-1","imageName":"` + policyRuleDesktopCanary + `"}],` +
		`"postActions":{"note":"` + policyRulePostCanary + `"}}`
}

// TestZPAPolicyRulesReadEachTypeAndRenderOperandValuesStandardOnly drives the
// real policy-rule handlers through a fake transport: every resource GETs its
// own policy type with the service-scoped microtenant, and operand values
// (which can be SCIM or SAML user attributes) render only in standard mode.
func TestZPAPolicyRulesReadEachTypeAndRenderOperandValuesStandardOnly(t *testing.T) {
	t.Parallel()

	var requests []*http.Request
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, status := `{"access_token":"test-token","expires_in":60}`, http.StatusOK
		switch path := request.URL.Path; {
		case strings.HasPrefix(path, zpaPolicyRulesPathPrefix):
			requests = append(requests, request.Clone(request.Context()))
			body = `{"totalPages":1,"list":[` + fakePolicyRule(strings.TrimPrefix(path, zpaPolicyRulesPathPrefix)) + `]}`
		case path != "/oauth2/v1/token":
			status, body = http.StatusNotFound, `{}`
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	})
	cfg := newSDKConfiguration(context.Background(), validReaderConfig())
	for _, client := range []*http.Client{cfg.HTTPClient, cfg.ZIAHTTPClient, cfg.ZPAHTTPClient} {
		client.Transport = transport
	}
	service, err := zsdk.NewOneAPIClient(cfg)
	if err != nil {
		t.Fatalf("NewOneAPIClient() error = %v", err)
	}
	t.Cleanup(service.Client.Close)
	handlers := make(map[resourceKey]resourceHandler)
	addZPAHandlers(handlers, sdkClient{services: fixedService{cfg: validReaderConfig(), sdkService: service.WithMicroTenant("service-scoped-microtenant")}})

	// Written out rather than read from zpaPolicyRuleTypes, so a wrong or
	// swapped policy type there fails here.
	want := map[string]string{
		"access-policy-rules":                        "ACCESS_POLICY",
		"timeout-policy-rules":                       "TIMEOUT_POLICY",
		"client-forwarding-policy-rules":             "CLIENT_FORWARDING_POLICY",
		"inspection-policy-rules":                    "INSPECTION_POLICY",
		"isolation-policy-rules":                     "ISOLATION_POLICY",
		"redirection-policy-rules":                   "REDIRECTION_POLICY",
		"capabilities-policy-rules":                  "CAPABILITIES_POLICY",
		"credential-policy-rules":                    "CREDENTIAL_POLICY",
		"clientless-session-protection-policy-rules": "CLIENTLESS_SESSION_PROTECTION_POLICY",
		"siem-policy-rules":                          "SIEM_POLICY",
	}
	if len(zpaPolicyRuleTypes) != len(want) {
		t.Errorf("zpaPolicyRuleTypes has %d types, want %d", len(zpaPolicyRuleTypes), len(want))
	}
	for name, policyType := range want {
		requests = nil
		records, err := handlers[resourceKey{product: resources.ProductZPA, name: name}].List(context.Background())
		if err != nil {
			t.Fatalf("zpa/%s List() error = %v", name, err)
		}
		if len(requests) != 1 {
			t.Fatalf("zpa/%s List() requests = %d, want 1", name, len(requests))
		}
		request := requests[0]
		if got, want := request.URL.Path, zpaPolicyRulesPathPrefix+policyType; request.Method != http.MethodGet || got != want {
			t.Errorf("zpa/%s request = %s %s, want GET %s", name, request.Method, got, want)
		}
		if got := request.URL.Query().Get("microtenantId"); got != "service-scoped-microtenant" {
			t.Errorf("zpa/%s request microtenantId = %q, want the service-scoped microtenant", name, got)
		}
		if len(records) != 1 {
			t.Fatalf("zpa/%s records = %d, want 1", name, len(records))
		}

		spec, ok := resources.FindSpec(resources.ProductZPA, name)
		if !ok {
			t.Fatalf("FindSpec(zpa, %s) ok = false", name)
		}
		for _, mode := range []redact.Mode{redact.ModeStandard, redact.ModeShare, redact.ModeParanoid} {
			projected, _, err := resources.ProjectRecords(spec, mode, records)
			if err != nil {
				t.Fatalf("ProjectRecords(zpa/%s, %s) error = %v", name, mode, err)
			}
			fields := projected.Records()[0].Fields()
			if err := resources.AssertRenderedSubset(spec, mode, fields); err != nil {
				t.Errorf("AssertRenderedSubset(zpa/%s, %s) error = %v", name, mode, err)
			}
			raw, err := json.Marshal(fields)
			if err != nil {
				t.Fatalf("json.Marshal(zpa/%s, %s) error = %v", name, mode, err)
			}
			rendered := string(raw)
			if fields["policyType"] != policyType {
				t.Errorf("zpa/%s %s policyType = %v, want %s", name, mode, fields["policyType"], policyType)
			}

			for _, canary := range []string{policyRuleSCIMCanary, policyRuleOperandCanary} {
				if got, want := strings.Contains(rendered, canary), mode == redact.ModeStandard; got != want {
					t.Errorf("zpa/%s %s renders operand value %q = %v, want %v", name, mode, canary, got, want)
				}
			}
			for _, canary := range []string{"policy-rule-canary-value", policyRuleServerCanary, policyRuleDesktopCanary, policyRulePostCanary} {
				if strings.Contains(rendered, canary) {
					t.Errorf("zpa/%s %s renders %q: %s", name, mode, canary, rendered)
				}
			}
			for _, want := range []string{`"objectType":"SCIM"`, `"lhs":"scim-attr-1"`, `"idpId":"idp-1"`, `"operator":"OR"`, `"id":"sg-1"`} {
				if !strings.Contains(rendered, want) {
					t.Errorf("zpa/%s %s missing %s: %s", name, mode, want, rendered)
				}
			}
			for _, absent := range []string{`"desktopPolicyMappings"`, `"postActions"`, `"servers"`, `"applications"`} {
				if strings.Contains(rendered, absent) {
					t.Errorf("zpa/%s %s renders %s: %s", name, mode, absent, rendered)
				}
			}
		}
	}
}
