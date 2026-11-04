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

const (
	zpaIdentityMgmtV1 = "/zpa/mgmtconfig/v1/admin/customers/zscalerctl-zpa-customer-id"
	zpaIdentityMgmtV2 = "/zpa/mgmtconfig/v2/admin/customers/zscalerctl-zpa-customer-id"
	zpaIdentityUserV1 = "/zpa/userconfig/v1/customers/zscalerctl-zpa-customer-id"
)

// zpaIdentityRoutes answers each identity route by escaped path. The second
// IdP's id holds a slash, so a per-IdP route that does not escape it misses.
func zpaIdentityRoutes() map[string]string {
	paged := func(records string) string { return `{"totalPages":"1","list":[` + records + `]}` }
	return map[string]string{
		// Administrators answer one bare array too.
		zpaIdentityMgmtV1 + "/administrators": `[{"id":"11","username":"admin-user-canary@example.com","displayName":"admin display canary",` +
			`"email":"admin-email-canary@example.com","phoneNumber":"+1-555-0100","comments":"admin comments canary",` +
			`"password":"` + "admin-pw-" + `canary","tmpPassword":"` + "admin-tmp-" + `canary","tokenId":"admin-token-canary",` +
			`"isEnabled":true,"isLocked":false,"forcePwdChange":false,"roleId":"1","role":{"id":"1"},` +
			`"microtenantName":"admin-microtenant-name","timezone":"America/Example"}]`,
		// Roles answer one bare array, not the paged envelope.
		zpaIdentityMgmtV1 + "/roles": `[{"id":"1","name":"role-name","description":"role description canary",` +
			`"users":"role-users-canary","apiKeys":"role-keys-canary","customRole":true,"hiddenUI":false,` +
			`"classPermissionGroups":[{"id":"2","name":"permission-group-name","classPermissions":[{"id":"3",` +
			`"permission":{"type":"VIEW_ONLY","mask":"1"},"classType":{"id":"4","aclClass":"ACL_CLASS","customerId":"role-customer-canary"}}]}]}]`,
		zpaIdentityMgmtV2 + "/idp": paged(`{"id":"9007199254740993","name":"idp-one-name","description":"idp description canary",` +
			`"domainList":["idp-domain-canary.example.com"],"idpEntityId":"idp-entity-canary","loginUrl":"https://idp-login-canary.example.com",` +
			`"scimServiceProviderEndpoint":"https://scim-endpoint-canary.example.com","scimSharedSecretExists":true,"ssoType":["USER"],` +
			`"adminMetadata":{"spEntityId":"sp-entity-canary","spPostUrl":"https://sp-post-canary.example.com"}},` +
			`{"id":"8/9","name":"idp-two-name"}`),
		zpaIdentityMgmtV2 + "/samlAttribute":                    paged(`{"id":"21","name":"saml-attr-name","samlName":"saml-claim-name","idpId":"9007199254740993","idpName":"idp-one-name"}`),
		zpaIdentityUserV1 + "/scimgroup/idpId/9007199254740993": paged(`{"id":31,"name":"scim-group-one","idpId":9007199254740993,"idpGroupId":"external-group-canary","internalId":"internal-group-canary"}`),
		zpaIdentityUserV1 + "/scimgroup/idpId/8%2F9":            paged(`{"id":32,"name":"scim-group-two","idpId":89}`),
		zpaIdentityMgmtV1 + "/idp/9007199254740993/scimattribute": paged(`{"id":"41","name":"scim-attr-one","idpId":"9007199254740993","dataType":"STRING",` +
			`"canonicalValues":["canonical-value-canary"],"description":"scim attr description canary","schemaURI":"urn:example:schema-canary"}`),
		zpaIdentityMgmtV1 + "/idp/8%2F9/scimattribute": paged(`{"id":"42","name":"scim-attr-two","idpId":"8/9"}`),
	}
}

// TestZPAIdentityHandlersReadTheirRoutesAndRedactPerMode drives the real
// handlers through a fake transport: roles read the bare array once, SCIM
// groups and attributes fan out over both IdPs in order at escaped routes,
// and identifiers render only where the catalog allows.
func TestZPAIdentityHandlersReadTheirRoutesAndRedactPerMode(t *testing.T) {
	t.Parallel()

	routes := zpaIdentityRoutes()
	requested := map[string]int{}
	var roleQueries, adminQueries []string
	handlers := newZPAIdentityTestHandlers(t, routes, func(request *http.Request) {
		path := request.URL.EscapedPath()
		requested[path]++
		switch path {
		case zpaIdentityMgmtV1 + "/roles":
			roleQueries = append(roleQueries, request.URL.RawQuery)
		case zpaIdentityMgmtV1 + "/administrators":
			adminQueries = append(adminQueries, request.URL.RawQuery)
		}
	})

	cases := []struct {
		name         string
		wantNames    []string
		standardOnly []string
		shared       []string
		everywhere   []string
		never        []string
	}{
		{
			name:      "admins",
			wantNames: []string{"admin-user-canary@example.com"},
			standardOnly: []string{"admin-user-canary@example.com", "admin display canary", "admin-email-canary@example.com",
				"+1-555-0100", "admin comments canary", "America/Example"},
			shared:     []string{"admin-microtenant-name"},
			everywhere: []string{`"isEnabled":true`, `"isLocked":false`, `"roleId":"1"`},
			never:      []string{"admin-pw-canary", "admin-tmp-canary", "admin-token-canary", "twoFactorAuth"},
		},
		{
			name:         "roles",
			wantNames:    []string{"role-name"},
			standardOnly: []string{"role description canary", "role-users-canary", "role-keys-canary", "role-customer-canary"},
			shared:       []string{"role-name", "permission-group-name", "VIEW_ONLY", "ACL_CLASS"},
			everywhere:   []string{`"customRole":true`},
		},
		{
			name:      "idps",
			wantNames: []string{"idp-one-name", "idp-two-name"},
			standardOnly: []string{"idp description canary", "idp-domain-canary.example.com", "idp-entity-canary",
				"idp-login-canary.example.com", "scim-endpoint-canary.example.com", "sp-entity-canary", "sp-post-canary.example.com"},
			shared:     []string{"idp-one-name", `"USER"`},
			everywhere: []string{`"scimSharedSecretExists":true`},
		},
		{
			name:         "saml-attributes",
			wantNames:    []string{"saml-attr-name"},
			standardOnly: []string{"saml-claim-name"},
			shared:       []string{"saml-attr-name", "idp-one-name"},
		},
		{
			name:         "scim-groups",
			wantNames:    []string{"scim-group-one", "scim-group-two"},
			standardOnly: []string{"external-group-canary", "internal-group-canary"},
			shared:       []string{"scim-group-one"},
			// 2^53+1: a float64 round trip would render ...992.
			everywhere: []string{`"idpId":9007199254740993`},
		},
		{
			name:         "scim-attributes",
			wantNames:    []string{"scim-attr-one", "scim-attr-two"},
			standardOnly: []string{"canonical-value-canary", "scim attr description canary", "urn:example:schema-canary"},
			shared:       []string{"scim-attr-one", "STRING"},
		},
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
		records, err := handler.List(context.Background())
		if err != nil {
			t.Fatalf("zpa/%s List() error = %v", c.name, err)
		}
		if len(records) != len(c.wantNames) {
			t.Fatalf("zpa/%s records = %d, want %d", c.name, len(records), len(c.wantNames))
		}
		for _, mode := range []redact.Mode{redact.ModeStandard, redact.ModeShare, redact.ModeParanoid} {
			projected, _, err := resources.ProjectRecords(spec, mode, records)
			if err != nil {
				t.Fatalf("ProjectRecords(zpa/%s, %s) error = %v", c.name, mode, err)
			}
			var rendered strings.Builder
			for i, record := range projected.Records() {
				fields := record.Fields()
				if err := resources.AssertRenderedSubset(spec, mode, fields); err != nil {
					t.Errorf("AssertRenderedSubset(zpa/%s, %s) error = %v", c.name, mode, err)
				}
				raw, err := json.Marshal(fields)
				if err != nil {
					t.Fatalf("json.Marshal(zpa/%s, %s) error = %v", c.name, mode, err)
				}
				if mode == redact.ModeStandard && !strings.Contains(string(raw), c.wantNames[i]) {
					t.Errorf("zpa/%s record %d = %s, want %q (IdP order)", c.name, i, raw, c.wantNames[i])
				}
				rendered.Write(raw)
			}
			check := func(canaries []string, want bool) {
				for _, canary := range canaries {
					if got := strings.Contains(rendered.String(), canary); got != want {
						t.Errorf("zpa/%s %s renders %q = %v, want %v: %s", c.name, mode, canary, got, want, rendered.String())
					}
				}
			}
			check(c.standardOnly, mode == redact.ModeStandard)
			check(c.shared, mode != redact.ModeParanoid)
			check(c.everywhere, true)
			check(c.never, false)
		}
	}

	for path := range routes {
		if requested[path] == 0 {
			t.Errorf("no request to %s", path)
		}
	}
	if requested[zpaIdentityMgmtV1+"/roles"] != 1 {
		t.Errorf("roles requests = %d, want 1 (unpaged)", requested[zpaIdentityMgmtV1+"/roles"])
	}
	for _, query := range roleQueries {
		if strings.Contains(query, "page") || !strings.Contains(query, "microtenantId=service-scoped-microtenant") {
			t.Errorf("roles query = %q, want no paging and the service-scoped microtenant", query)
		}
	}
	if requested[zpaIdentityMgmtV1+"/administrators"] != 1 {
		t.Errorf("administrators requests = %d, want 1 (unpaged)", requested[zpaIdentityMgmtV1+"/administrators"])
	}
	// The SDK's GetAll scopes administrators to the service's microtenant.
	for _, query := range adminQueries {
		if strings.Contains(query, "page") || !strings.Contains(query, "microtenantId=service-scoped-microtenant") {
			t.Errorf("administrators query = %q, want no paging and the service-scoped microtenant", query)
		}
	}
}

// newZPAIdentityTestHandlers registers the real ZPA handlers against a fake
// transport answering routes by escaped path; a route absent from routes
// answers 404. Each matched request is passed to seen, if set.
func newZPAIdentityTestHandlers(t *testing.T, routes map[string]string, seen func(*http.Request)) map[resourceKey]resourceHandler {
	t.Helper()
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, status := `{"access_token":"test-token","expires_in":60}`, http.StatusOK
		path := request.URL.EscapedPath()
		if routeBody, ok := routes[path]; ok {
			if seen != nil {
				seen(request)
			}
			body = routeBody
		} else if path != "/oauth2/v1/token" {
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
	return handlers
}

// TestZPAIdentityReadsFailClosed: an unreadable body, a failing IdP or an
// IdP without an id fails the whole list with no records, never a short
// list that looks complete.
func TestZPAIdentityReadsFailClosed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		resource string
		mutate   func(routes map[string]string)
	}{
		{"admins blank body", "admins", func(r map[string]string) { r[zpaIdentityMgmtV1+"/administrators"] = "" }},
		{"admins null body", "admins", func(r map[string]string) { r[zpaIdentityMgmtV1+"/administrators"] = "null" }},
		{"admins object body", "admins", func(r map[string]string) { r[zpaIdentityMgmtV1+"/administrators"] = "{}" }},
		{"roles blank body", "roles", func(r map[string]string) { r[zpaIdentityMgmtV1+"/roles"] = "" }},
		{"roles null body", "roles", func(r map[string]string) { r[zpaIdentityMgmtV1+"/roles"] = "null" }},
		{"roles object body", "roles", func(r map[string]string) { r[zpaIdentityMgmtV1+"/roles"] = "{}" }},
		{"scim groups second idp fails", "scim-groups", func(r map[string]string) { delete(r, zpaIdentityUserV1+"/scimgroup/idpId/8%2F9") }},
		{"scim attributes second idp fails", "scim-attributes", func(r map[string]string) { delete(r, zpaIdentityMgmtV1+"/idp/8%2F9/scimattribute") }},
		{"scim groups idp without id", "scim-groups", func(r map[string]string) {
			r[zpaIdentityMgmtV2+"/idp"] = `{"totalPages":"1","list":[{"id":"9007199254740993","name":"idp-one-name"},{"name":"idp-no-id"}]}`
			r[zpaIdentityUserV1+"/scimgroup/idpId/"] = `{"totalPages":"1","list":[{"id":99,"name":"every-group"}]}`
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			routes := zpaIdentityRoutes()
			c.mutate(routes)
			handlers := newZPAIdentityTestHandlers(t, routes, nil)
			records, err := handlers[resourceKey{product: resources.ProductZPA, name: c.resource}].List(context.Background())
			if err == nil || len(records) != 0 {
				t.Fatalf("zpa/%s List() = %d records, err %v; want an error and no records", c.resource, len(records), err)
			}
		})
	}
}

// TestZPAAdminGetReadsOneAdminAndFailsClosed: get reads the admin's own route
// in the service microtenant and renders like the list; a null, {} or other
// admin's body is an error, never a zero record that reads as disabled.
func TestZPAAdminGetReadsOneAdminAndFailsClosed(t *testing.T) {
	t.Parallel()

	spec, ok := resources.FindSpec(resources.ProductZPA, "admins")
	if !ok {
		t.Fatal("FindSpec(zpa, admins) ok = false")
	}
	adminRoutes := func() map[string]string {
		routes := zpaIdentityRoutes()
		routes[zpaIdentityMgmtV1+"/administrators/11"] = `{"id":"11","username":"admin-user-canary@example.com",` +
			`"password":"` + "admin-pw-" + `canary","isEnabled":true,"role":{"id":"1"}}`
		return routes
	}
	var queries []string
	handlers := newZPAIdentityTestHandlers(t, adminRoutes(), func(request *http.Request) {
		if request.URL.EscapedPath() == zpaIdentityMgmtV1+"/administrators/11" {
			queries = append(queries, request.URL.RawQuery)
		}
	})
	record, err := getResource(context.Background(), handlers, resources.ProductZPA, "admins", "11")
	if err != nil {
		t.Fatalf("zpa/admins Get(11) error = %v", err)
	}
	if len(queries) != 1 || !strings.Contains(queries[0], "microtenantId=service-scoped-microtenant") {
		t.Errorf("admin get queries = %q, want one in the service-scoped microtenant", queries)
	}
	for _, mode := range []redact.Mode{redact.ModeStandard, redact.ModeShare, redact.ModeParanoid} {
		projected, _, err := resources.ProjectRecords(spec, mode, []resources.SourceRecord{record})
		if err != nil {
			t.Fatalf("ProjectRecords(zpa/admins, %s) error = %v", mode, err)
		}
		raw, err := json.Marshal(projected.Records()[0].Fields())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "admin-pw-canary") {
			t.Errorf("zpa/admins get %s renders the password: %s", mode, raw)
		}
		if got := strings.Contains(string(raw), "admin-user-canary@example.com"); got != (mode == redact.ModeStandard) {
			t.Errorf("zpa/admins get %s renders the username = %v: %s", mode, got, raw)
		}
	}

	for name, body := range map[string]string{"null": "null", "empty object": "{}", "other admin": `{"id":"12","isEnabled":false}`} {
		routes := adminRoutes()
		routes[zpaIdentityMgmtV1+"/administrators/11"] = body
		handlers := newZPAIdentityTestHandlers(t, routes, nil)
		if record, err := getResource(context.Background(), handlers, resources.ProductZPA, "admins", "11"); err == nil {
			t.Errorf("zpa/admins Get(11) with a %s body = %+v, want an error", name, record)
		}
	}
}
