package zscaler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"zscalerctl/internal/redact"
	"zscalerctl/internal/resources"
)

func zpaScoutRoutes() map[string]string {
	paged := func(records string) string { return `{"totalPages":"1","list":[` + records + `]}` }
	summary := func(name string) string {
		return paged(`{"id":"1","name":"` + name + `","enabled":true},{"id":"2","name":"` + name + `-off","enabled":false},{"id":"3","name":"` + name + `-unset"}`)
	}
	return map[string]string{
		zpaIdentityMgmtV1 + "/branchConnectorGroup/summary": summary("branch-group-name"),
		zpaIdentityMgmtV1 + "/extranetResource/partner":     summary("extranet-partner-name"),
		zpaIdentityMgmtV1 + "/location/summary":             summary("location-summary-name"),
		zpaIdentityMgmtV1 + "/workloadTagGroup/summary":     summary("workload-tag-name"),
		zpaIdentityMgmtV1 + "/customerDRToolVersion": paged(`{"id":"2","name":"dr-build-name","platform":"LINUX",` +
			`"version":"9.9.9","latest":true,"customerId":"dr-customer-canary"},{"id":"9","name":"dr-old-build","latest":false}`),
		zpaIdentityMgmtV1 + "/managedBrowserProfile/search": paged(`{"id":"3","name":"browser-profile-name",` +
			`"description":"browser description canary","customerId":"browser-customer-canary","browserType":"CHROME",` +
			`"chromePostureProfile":{"id":"4","crowdStrikeAgent":true}},{"id":"10","name":"browser-profile-two","chromePostureProfile":{"id":"11","crowdStrikeAgent":false}}`),
		// Client settings answer an array, not the one object the SDK decodes.
		zpaIdentityMgmtV1 + "/clientSetting/all": `[{"id":"5","name":"client-setting-name","clientCertificateType":"ZAPP_CLIENT",` +
			`"enrollmentCertId":"6","enrollmentCertName":"enrollment-cert-name"}]`,
		zpaIdentityMgmtV1 + "/v2/ssoLoginOptions":                `{"ssologinonly":false}`,
		zpaIdentityMgmtV1 + "/config/sessionTerminationOnReauth": `{"sessionTerminationOnReauth":true,"allowDisableSessionTerminationOnReauth":false}`,
		zpaIdentityMgmtV1 + "/authDomains":                       `{"authDomains":["auth-domain-canary.example.com"]}`,
		zpaIdentityMgmtV1 + "/iamidpmapping": `{"id":"7","sequence":"1","iamIdpIdToIamIdMapping":{"deliveryTag":3,` +
			`"orgId":"org-id-canary","mappings":[{"iamIdpId":"iam-idp-canary","idpId":"8","idpName":"mapped-idp-name"}]}}`,
		zpaIdentityMgmtV1 + "/stepupauthlevel": `["STEP_UP_LEVEL_ONE","STEP_UP_LEVEL_TWO"]`,
	}
}

// TestZPAScoutHandlersReadTheirRoutesAndRedactPerMode drives the ledger-scout
// handlers through a fake transport: each reads its route, client settings
// and step-up levels decode bare arrays, and identifiers render only where
// the catalog allows.
func TestZPAScoutHandlersReadTheirRoutesAndRedactPerMode(t *testing.T) {
	t.Parallel()

	routes := zpaScoutRoutes()
	requested := map[string]int{}
	handlers := newZPAIdentityTestHandlers(t, routes, func(request *http.Request) { requested[request.URL.EscapedPath()]++ })

	cases := []struct {
		name         string
		show         bool
		records      int
		standardOnly []string
		shared       []string
		everywhere   []string
	}{
		// A false and an absent enabled must stay distinguishable.
		{name: "branch-connector-groups", records: 3, shared: []string{"branch-group-name"},
			everywhere: []string{`"enabled":true`, `"enabled":false`, `"enabled":null`}},
		{name: "extranet-partners", records: 3, standardOnly: []string{"extranet-partner-name"}},
		{name: "location-summaries", records: 3, shared: []string{"location-summary-name"}},
		{name: "workload-tag-groups", records: 3, shared: []string{"workload-tag-name"}},
		{name: "dr-tool-versions", records: 2, standardOnly: []string{"dr-customer-canary"}, shared: []string{"dr-build-name"}, everywhere: []string{"9.9.9", "LINUX", `"latest":true`, `"latest":false`}},
		{name: "managed-browser-profiles", records: 2, standardOnly: []string{"browser description canary", "browser-customer-canary"},
			shared: []string{"browser-profile-name", `"crowdStrikeAgent":true`, `"crowdStrikeAgent":false`}, everywhere: []string{"CHROME"}},
		{name: "client-settings", shared: []string{"client-setting-name", "ZAPP_CLIENT", "enrollment-cert-name"}},
		{name: "admin-sso-options", show: true, shared: []string{`"ssologinonly":false`}},
		{name: "session-termination-settings", show: true, shared: []string{`"sessionTerminationOnReauth":true`, `"allowDisableSessionTerminationOnReauth":false`}},
		{name: "auth-domains", show: true, standardOnly: []string{"auth-domain-canary.example.com"}},
		{name: "iam-idp-mapping", show: true, standardOnly: []string{"org-id-canary", "iam-idp-canary"}, shared: []string{"mapped-idp-name"}, everywhere: []string{`"sequence":"1"`}},
		{name: "step-up-auth-levels", show: true, shared: []string{"STEP_UP_LEVEL_ONE", "STEP_UP_LEVEL_TWO"}},
	}
	for _, c := range cases {
		handler := handlers[resourceKey{product: resources.ProductZPA, name: c.name}]
		if handler == nil {
			t.Fatalf("zpa/%s handler = nil, want registered handler", c.name)
		}
		spec, ok := resources.FindSpec(resources.ProductZPA, c.name)
		if !ok {
			t.Fatalf("FindSpec(zpa, %s) ok = false", c.name)
		}
		var records []resources.SourceRecord
		var err error
		if c.show {
			var record resources.SourceRecord
			record, err = handler.Show(context.Background())
			records = []resources.SourceRecord{record}
		} else {
			records, err = handler.List(context.Background())
		}
		if err != nil {
			t.Fatalf("zpa/%s read error = %v", c.name, err)
		}
		if want := max(c.records, 1); len(records) != want {
			t.Fatalf("zpa/%s records = %d, want %d", c.name, len(records), want)
		}
		for _, mode := range []redact.Mode{redact.ModeStandard, redact.ModeShare, redact.ModeParanoid} {
			projected, _, err := resources.ProjectRecords(spec, mode, records)
			if err != nil {
				t.Fatalf("ProjectRecords(zpa/%s, %s) error = %v", c.name, mode, err)
			}
			var all []map[string]any
			for _, record := range projected.Records() {
				fields := record.Fields()
				if err := resources.AssertRenderedSubset(spec, mode, fields); err != nil {
					t.Errorf("AssertRenderedSubset(zpa/%s, %s) error = %v", c.name, mode, err)
				}
				all = append(all, fields)
			}
			raw, err := json.Marshal(all)
			if err != nil {
				t.Fatalf("json.Marshal(zpa/%s, %s) error = %v", c.name, mode, err)
			}
			rendered := string(raw)
			check := func(canaries []string, want bool) {
				for _, canary := range canaries {
					if got := strings.Contains(rendered, canary); got != want {
						t.Errorf("zpa/%s %s renders %q = %v, want %v: %s", c.name, mode, canary, got, want, rendered)
					}
				}
			}
			check(c.standardOnly, mode == redact.ModeStandard)
			check(c.shared, mode != redact.ModeParanoid)
			check(c.everywhere, true)
		}
	}
	for path := range routes {
		if requested[path] == 0 {
			t.Errorf("no request to %s", path)
		}
	}
}

// TestZPAScoutSingletonsFailClosed: a null, blank or non-object settings body
// fails the read instead of rendering empty settings.
func TestZPAScoutSingletonsFailClosed(t *testing.T) {
	t.Parallel()

	for _, body := range []string{"null", "", "[]"} {
		routes := zpaScoutRoutes()
		routes[zpaIdentityMgmtV1+"/authDomains"] = body
		handlers := newZPAIdentityTestHandlers(t, routes, nil)
		if _, err := handlers[resourceKey{product: resources.ProductZPA, name: "auth-domains"}].Show(context.Background()); err == nil {
			t.Errorf("zpa/auth-domains Show() with body %q error = nil, want an error", body)
		}
	}
}

// TestZPAScoutSettingsMissingFieldFail: a settings object without the field
// it exists to report fails rather than rendering as unset.
func TestZPAScoutSettingsMissingFieldFail(t *testing.T) {
	t.Parallel()

	for _, c := range []struct{ resource, route string }{
		{"admin-sso-options", "/v2/ssoLoginOptions"},
		{"session-termination-settings", "/config/sessionTerminationOnReauth"},
	} {
		routes := zpaScoutRoutes()
		routes[zpaIdentityMgmtV1+c.route] = `{}`
		handlers := newZPAIdentityTestHandlers(t, routes, nil)
		if _, err := handlers[resourceKey{product: resources.ProductZPA, name: c.resource}].Show(context.Background()); err == nil {
			t.Errorf("zpa/%s Show() with body {} error = nil, want an error", c.resource)
		}
	}
}

// TestZPAScoutMicrotenantScopeMatchesSDK pins which routes send the
// service-scoped microtenant: the ones the SDK scopes, and no others.
func TestZPAScoutMicrotenantScopeMatchesSDK(t *testing.T) {
	t.Parallel()

	scoped := map[string]bool{
		"branch-connector-groups": true, "workload-tag-groups": true, "managed-browser-profiles": true, "step-up-auth-levels": true,
		"extranet-partners": false, "location-summaries": false, "dr-tool-versions": false, "client-settings": false,
		"admin-sso-options": false, "session-termination-settings": false, "auth-domains": false, "iam-idp-mapping": false,
	}
	routeOf := map[string]string{
		"branch-connector-groups": "/branchConnectorGroup/summary", "workload-tag-groups": "/workloadTagGroup/summary",
		"managed-browser-profiles": "/managedBrowserProfile/search", "step-up-auth-levels": "/stepupauthlevel",
		"extranet-partners": "/extranetResource/partner", "location-summaries": "/location/summary",
		"dr-tool-versions": "/customerDRToolVersion", "client-settings": "/clientSetting/all",
		"admin-sso-options": "/v2/ssoLoginOptions", "session-termination-settings": "/config/sessionTerminationOnReauth",
		"auth-domains": "/authDomains", "iam-idp-mapping": "/iamidpmapping",
	}
	queries := map[string]string{}
	handlers := newZPAIdentityTestHandlers(t, zpaScoutRoutes(), func(request *http.Request) {
		queries[request.URL.EscapedPath()] = request.URL.RawQuery
	})
	for name, wantScoped := range scoped {
		handler := handlers[resourceKey{product: resources.ProductZPA, name: name}]
		var err error
		if spec, _ := resources.FindSpec(resources.ProductZPA, name); spec.Shape == resources.ShapeSingleton {
			_, err = handler.Show(context.Background())
		} else {
			_, err = handler.List(context.Background())
		}
		if err != nil {
			t.Fatalf("zpa/%s read error = %v", name, err)
		}
		got := strings.Contains(queries[zpaIdentityMgmtV1+routeOf[name]], "microtenantId=service-scoped-microtenant")
		if got != wantScoped {
			t.Errorf("zpa/%s sends the service microtenant = %v, want %v (the SDK's scoping)", name, got, wantScoped)
		}
	}
}
