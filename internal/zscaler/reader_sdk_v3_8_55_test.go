package zscaler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"zscalerctl/internal/redact"
	"zscalerctl/internal/resources"
)

// sdkV3855Case is one SDK v3.8.55 ZIA resource read through its registered
// handler: the endpoint it must hit, the record the fake API returns, and
// which canaries render in which modes.
type sdkV3855Case struct {
	name string
	path string
	body string
	// standardOnly render in standard and nowhere else; never render in no
	// mode; shared render in standard and share; everywhere render in every
	// mode.
	standardOnly []string
	never        []string
	shared       []string
	everywhere   []string
}

func sdkV3855Cases() []sdkV3855Case {
	ruleScope := `"users":[{"id":11,"name":"rule-user-canary"}],` +
		`"groups":[{"id":12,"name":"rule-group-canary"}],` +
		`"auditor":{"id":13,"name":"rule-auditor-canary"},` +
		`"lastModifiedBy":{"id":14,"name":"rule-admin-canary"},` +
		// Not email-shaped, so the share scrubber cannot mask a leak.
		`"externalAuditorEmail":"auditor-mailbox-canary",` +
		`"notificationTemplate":{"id":19,"name":"notify-template-canary"},` +
		`"receiver":{"id":20,"name":"receiver-name-canary","type":"INCIDENT"},` +
		`"description":"rule description canary",` +
		`"subRules":[{"id":2,"name":"sub-rule-canary"}]`
	return []sdkV3855Case{
		{
			name:         "dns-application-groups",
			path:         "/zia/api/v1/dnsApplicationGroups",
			body:         `[{"id":1,"name":"dns-group-name","dnsApplications":["DNS_APP_CANARY"],"description":"dns description canary"}]`,
			standardOnly: []string{"dns description canary"},
			shared:       []string{"dns-group-name", "DNS_APP_CANARY"},
		},
		{
			name: "endpoint-dlp-rules",
			path: "/zia/api/v1/endPointDlpRules",
			body: `[{"id":1,"name":"endpoint-rule-name","action":"BLOCK",` + ruleScope + `,` +
				`"eunTemplateId":987651,"ucTemplateId":987652,` +
				`"devices":[{"id":15,"name":"device-canary"}],` +
				`"endPointApplications":[{"resourceId":16,"applicationName":"endpoint-app-name",` +
				`"description":"app description canary","filename":"app-file-canary.exe",` +
				`"originalFileName":"app-original-canary.exe","bundleID":"com.example.bundle-canary",` +
				`"version":{"version":"7.7.7","bundleID":"com.example.version-canary"}}],` +
				`"endPointApplicationGroups":[{"groupId":22,"name":"app-group-name","description":"group description canary",` +
				`"endPointApplications":[{"resourceId":23,"applicationName":"nested-app-name","filename":"nested-file-canary.exe"}]}]}]`,
			standardOnly: []string{
				"rule-user-canary", "rule-group-canary", "rule-auditor-canary", "rule-admin-canary",
				"auditor-mailbox-canary", "notify-template-canary", "receiver-name-canary",
				"rule description canary", "device-canary", "987651", "987652",
				"app description canary", "app-file-canary.exe", "app-original-canary.exe",
				"com.example.bundle-canary", "com.example.version-canary", "nested-file-canary.exe",
				"group description canary",
			},
			never:  []string{"sub-rule-canary"},
			shared: []string{"endpoint-rule-name", "endpoint-app-name", "7.7.7", "app-group-name", "nested-app-name", `"groupId":22`, `"id":11`, `"id":15`},
		},
		{
			name: "email-dlp-rules",
			path: "/zia/api/v1/emailDlpRules",
			body: `[{"id":1,"name":"email-rule-name","customHeader":"X-Custom-Canary: yes",` + ruleScope + `,` +
				`"excludedUsers":[{"id":17,"name":"excluded-user-canary"}],` +
				`"emailTenants":[{"id":24,"name":"email-tenant-canary"}]}]`,
			standardOnly: []string{"rule-user-canary", "rule-group-canary", "rule-auditor-canary", "rule-admin-canary", "auditor-mailbox-canary", "notify-template-canary", "receiver-name-canary", "rule description canary", "X-Custom-Canary", "excluded-user-canary", "email-tenant-canary"},
			never:        []string{"sub-rule-canary"},
			shared:       []string{"email-rule-name", `"id":11`, `"id":17`, `"id":24`},
		},
		{
			name:         "alert-definitions",
			path:         "/zia/api/v1/alertDefinitions",
			body:         `[{"id":1,"alertName":"alert-name","scope":"USER","entity":{"id":18,"name":"alert-entity-canary"},"comments":"alert comment canary"}]`,
			standardOnly: []string{"alert-entity-canary", "alert comment canary"},
			shared:       []string{"alert-name", `"id":18`},
		},
		{
			name:         "http-header-profiles",
			path:         "/zia/api/v1/httpHeaderProfile",
			body:         `[{"id":1,"name":"header-profile-name","description":"header description canary","httpHeaderProfileCriteria":[{"id":2,"header":"ORIGIN","userAgent":"agent-canary"}]}]`,
			standardOnly: []string{"header description canary"},
			shared:       []string{"header-profile-name", "ORIGIN", "agent-canary"},
		},
		{
			name:   "http-header-action-profiles",
			path:   "/zia/api/v1/httpHeaderActionProfile",
			body:   `[{"id":1,"name":"header-action-name","httpHeaderActionProfileKeys":[{"id":2,"key":"x-allowed-tenants","value":"header-value-canary"}]}]`,
			never:  []string{"header-value-canary"},
			shared: []string{"header-action-name", "x-allowed-tenants"},
		},
		{
			name:       "adaptive-access-profiles",
			path:       "/zia/api/v1/adaptiveAccessProfiles",
			body:       `[{"id":1,"name":"adaptive-profile-name","type":"CUSTOM","iamAapId":"aap-canary"}]`,
			shared:     []string{"adaptive-profile-name"},
			everywhere: []string{"aap-canary"},
		},
		{
			name:         "dlp-global-options",
			path:         "/zia/api/v1/webDlpGlobalOptions",
			body:         `{"urls":["exempt-canary.example.com"],"applications":["APP_CANARY"],"exemptUrlEncodedData":true,"urlCategories":[{"id":21,"name":"url-category-canary"}]}`,
			standardOnly: []string{"exempt-canary.example.com", "url-category-canary"},
			shared:       []string{"APP_CANARY", `"exemptUrlEncodedData":true`, `"id":21`},
		},
	}
}

func TestSDKv3855HandlersReadTheirEndpointAndRedactPerMode(t *testing.T) {
	t.Parallel()

	cases := sdkV3855Cases()
	byPath := map[string]sdkV3855Case{}
	for _, c := range cases {
		byPath[c.path] = c
	}
	requested := map[string]int{}
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body := `{"access_token":"test-token","expires_in":60}`
		statusCode := http.StatusOK
		if c, ok := byPath[request.URL.Path]; ok {
			requested[request.URL.Path]++
			body = c.body
			// Paged reads stop at the first short page; page 2 is empty.
			if page := request.URL.Query().Get("page"); page != "" && page != "1" {
				body = `[]`
			}
		} else if request.URL.Path != "/oauth2/v1/token" {
			statusCode = http.StatusNotFound
			body = `{}`
		}
		return &http.Response{
			StatusCode: statusCode,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	})
	service := newZIATestService(t, transport)
	handlers := make(map[resourceKey]resourceHandler)
	addZIAHandlers(handlers, sdkClient{services: fixedService{cfg: validReaderConfig(), sdkService: service}})

	for _, c := range cases {
		handler := handlers[resourceKey{product: resources.ProductZIA, name: c.name}]
		if handler == nil {
			t.Fatalf("zia/%s handler = nil, want registered handler", c.name)
		}
		spec, ok := resources.FindSpec(resources.ProductZIA, c.name)
		if !ok {
			t.Fatalf("FindSpec(zia, %s) ok = false", c.name)
		}
		var records []resources.SourceRecord
		var err error
		if c.name == "dlp-global-options" {
			var record resources.SourceRecord
			record, err = handler.Show(context.Background())
			records = []resources.SourceRecord{record}
		} else {
			records, err = handler.List(context.Background())
		}
		if err != nil {
			t.Fatalf("zia/%s read error = %v", c.name, err)
		}
		if requested[c.path] == 0 {
			t.Fatalf("zia/%s did not request %s", c.name, c.path)
		}
		if len(records) != 1 {
			t.Fatalf("zia/%s records = %d, want 1", c.name, len(records))
		}

		for _, mode := range []redact.Mode{redact.ModeStandard, redact.ModeShare, redact.ModeParanoid} {
			projected, _, err := resources.ProjectRecords(spec, mode, records)
			if err != nil {
				t.Fatalf("ProjectRecords(zia/%s, %s) error = %v", c.name, mode, err)
			}
			fields := projected.Records()[0].Fields()
			if err := resources.AssertRenderedSubset(spec, mode, fields); err != nil {
				t.Errorf("AssertRenderedSubset(zia/%s, %s) error = %v", c.name, mode, err)
			}
			raw, err := json.Marshal(fields)
			if err != nil {
				t.Fatalf("json.Marshal(zia/%s, %s) error = %v", c.name, mode, err)
			}
			rendered := string(raw)
			check := func(canaries []string, want bool) {
				for _, canary := range canaries {
					if got := strings.Contains(rendered, canary); got != want {
						t.Errorf("zia/%s %s renders %q = %v, want %v: %s", c.name, mode, canary, got, want, rendered)
					}
				}
			}
			check(c.standardOnly, mode == redact.ModeStandard)
			check(c.never, false)
			check(c.shared, mode != redact.ModeParanoid)
			check(c.everywhere, true)
		}
	}
}
