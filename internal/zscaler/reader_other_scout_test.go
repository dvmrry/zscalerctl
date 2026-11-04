package zscaler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	zsdk "github.com/zscaler/zscaler-sdk-go/v3/zscaler"

	"zscalerctl/internal/redact"
	"zscalerctl/internal/resources"
)

func otherScoutRoutes() map[string]string {
	return map[string]string{
		"/zia/api/v1/devices": `[{"id":1,"name":"device-name-canary","active":true,"rooted":false,"deleted":false,` +
			`"os":"WINDOWS","hostname":"host-canary","udid":"udid-canary","hardwareId":"hardware-canary",` +
			`"macAddress":"00:00:5E:00:53:01","model":"model-canary","user":{"id":77,"name":"device-user-canary"}}]`,
		"/zia/api/v1/integrationPartners":       `[{"id":2,"name":"partner-name","type":"CROWDSTRIKE","disabled":false}]`,
		"/zia/api/v1/smpcInstance":              `{"data":[{"id":3,"datacenterName":"DC-ONE","connectionStatus":"UP","nodes":[{"id":4,"nodeName":"node-one","adminIsolationEnabled":false}]}]}`,
		"/zcc/papi/public/v1/getWebPrivacyInfo": `{"id":"5","active":"1","collectUserInfo":"0","enablePacketCapture":"1"}`,
		"/ziam/admin/api/v1/api-clients": `{"results_total":1,"pageOffset":0,"pageSize":1,"next_link":"","records":[{"id":"6","name":"api-client-name",` +
			`"description":"client description canary","status":false,"accessTokenLifeTime":3600,` +
			`"clientAuthentication":{"authType":"PRIVATE_KEY_JWT","clientJWKsUrl":"https://jwks-canary.example.com",` +
			`"publicKeys":[{"keyName":"key-name-canary","keyValue":"PUBLIC-KEY-CANARY"}],"clientCertificates":[{"certContent":"CERT-CANARY"}]},` +
			`"clientResources":[{"id":"7","name":"resource-name","defaultApi":false,"selectedScopes":[{"id":"8","name":"scope-name"}]}]}]}`,
	}
}

// TestOtherScoutHandlersReadTheirRoutesAndRedactPerMode drives the ZIA, ZCC
// and Zidentity ledger-scout handlers through a fake transport: each reads its
// route, booleans keep false, hardware identifiers and owners are standard
// only, and API client key material never renders.
func TestOtherScoutHandlersReadTheirRoutesAndRedactPerMode(t *testing.T) {
	t.Parallel()

	routes := otherScoutRoutes()
	requested := map[string]int{}
	handlers := newOtherScoutTestHandlers(t, routes, requested)
	var err error

	cases := []struct {
		product      resources.Product
		name         string
		show         bool
		standardOnly []string
		shared       []string
		everywhere   []string
		never        []string
	}{
		{product: resources.ProductZIA, name: "registered-devices",
			standardOnly: []string{"device-name-canary", "host-canary", "udid-canary", "hardware-canary", "00:00:5E:00:53:01", "model-canary", "device-user-canary", `"id":77`},
			shared:       []string{"WINDOWS"},
			everywhere:   []string{`"rooted":false`, `"active":true`}},
		{product: resources.ProductZIA, name: "integration-partners", shared: []string{"partner-name", "CROWDSTRIKE"}, everywhere: []string{`"disabled":false`}},
		{product: resources.ProductZIA, name: "smpc-instances", everywhere: []string{"DC-ONE", "node-one", `"adminIsolationEnabled":false`}},
		{product: resources.ProductZCC, name: "web-privacy", show: true, shared: []string{`"collectUserInfo":"0"`, `"enablePacketCapture":"1"`}},
		{product: resources.ProductZidentity, name: "api-clients",
			standardOnly: []string{"client description canary", "jwks-canary.example.com", "key-name-canary"},
			shared:       []string{"api-client-name", "PRIVATE_KEY_JWT", "resource-name", "scope-name", `"defaultApi":false`},
			everywhere:   []string{`"status":false`, `"accessTokenLifeTime":3600`},
			never:        []string{"PUBLIC-KEY-CANARY", "CERT-CANARY"}},
	}
	for _, c := range cases {
		handler := handlers[resourceKey{product: c.product, name: c.name}]
		if handler == nil {
			t.Fatalf("%s/%s handler = nil, want registered handler", c.product, c.name)
		}
		spec, ok := resources.FindSpec(c.product, c.name)
		if !ok {
			t.Fatalf("FindSpec(%s, %s) ok = false", c.product, c.name)
		}
		var records []resources.SourceRecord
		if c.show {
			var record resources.SourceRecord
			record, err = handler.Show(context.Background())
			records = []resources.SourceRecord{record}
		} else {
			records, err = handler.List(context.Background())
		}
		if err != nil {
			t.Fatalf("%s/%s read error = %v", c.product, c.name, err)
		}
		if len(records) != 1 {
			t.Fatalf("%s/%s records = %d, want 1", c.product, c.name, len(records))
		}
		for _, mode := range []redact.Mode{redact.ModeStandard, redact.ModeShare, redact.ModeParanoid} {
			projected, _, err := resources.ProjectRecords(spec, mode, records)
			if err != nil {
				t.Fatalf("ProjectRecords(%s/%s, %s) error = %v", c.product, c.name, mode, err)
			}
			fields := projected.Records()[0].Fields()
			if err := resources.AssertRenderedSubset(spec, mode, fields); err != nil {
				t.Errorf("AssertRenderedSubset(%s/%s, %s) error = %v", c.product, c.name, mode, err)
			}
			raw, err := json.Marshal(fields)
			if err != nil {
				t.Fatalf("json.Marshal(%s/%s, %s) error = %v", c.product, c.name, mode, err)
			}
			rendered := string(raw)
			check := func(canaries []string, want bool) {
				for _, canary := range canaries {
					if got := strings.Contains(rendered, canary); got != want {
						t.Errorf("%s/%s %s renders %q = %v, want %v: %s", c.product, c.name, mode, canary, got, want, rendered)
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
}

func newOtherScoutTestHandlers(t *testing.T, routes map[string]string, requested map[string]int) map[resourceKey]resourceHandler {
	t.Helper()
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, status := `{"access_token":"test-token","expires_in":60}`, http.StatusOK
		if routeBody, ok := routes[request.URL.Path]; ok {
			if requested != nil {
				requested[request.URL.Path]++
			}
			body = routeBody
			// Paged ZIA reads stop at the first short page; page 2 is empty.
			if page := request.URL.Query().Get("page"); page != "" && page != "1" {
				body = `[]`
			}
		} else if request.URL.Path != "/oauth2/v1/token" {
			if requested != nil {
				requested["unexpected:"+request.URL.Path]++
			}
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
	addZIAHandlers(handlers, sdkClient{services: fixedService{cfg: validReaderConfig(), sdkService: service}})
	return handlers
}

// TestOtherScoutGetsReadTheirRecord: get reads the record at /<route>/<id>.
func TestOtherScoutGetsReadTheirRecord(t *testing.T) {
	t.Parallel()

	routes := otherScoutRoutes()
	routes["/zia/api/v1/devices/1"] = `{"id":1,"os":"get-device-os"}`
	routes["/ziam/admin/api/v1/api-clients/6"] = `{"id":"6","name":"get-client-name"}`
	requested := map[string]int{}
	handlers := newOtherScoutTestHandlers(t, routes, requested)
	for _, c := range []struct {
		product      resources.Product
		name, id     string
		path, canary string
	}{
		{resources.ProductZIA, "registered-devices", "1", "/zia/api/v1/devices/1", "get-device-os"},
		{resources.ProductZidentity, "api-clients", "6", "/ziam/admin/api/v1/api-clients/6", "get-client-name"},
	} {
		record, err := handlers[resourceKey{product: c.product, name: c.name}].Get(context.Background(), c.id)
		if err != nil {
			t.Fatalf("%s/%s Get(%s) error = %v", c.product, c.name, c.id, err)
		}
		spec, _ := resources.FindSpec(c.product, c.name)
		projected, _, err := resources.ProjectRecords(spec, redact.ModeStandard, []resources.SourceRecord{record})
		if err != nil {
			t.Fatalf("ProjectRecords(%s/%s) error = %v", c.product, c.name, err)
		}
		raw, _ := json.Marshal(projected.Records()[0].Fields())
		if requested[c.path] != 1 || !strings.Contains(string(raw), c.canary) {
			t.Errorf("%s/%s Get(%s) = %s after %d requests to %s, want the record there", c.product, c.name, c.id, raw, requested[c.path], c.path)
		}
	}
}

// TestOtherScoutListsFailClosed: a null partners array or an SMPC body
// without its data list fails rather than reading as none.
func TestOtherScoutListsFailClosed(t *testing.T) {
	t.Parallel()

	for _, c := range []struct{ name, route, body string }{
		{"integration-partners", "/zia/api/v1/integrationPartners", "null"},
		{"smpc-instances", "/zia/api/v1/smpcInstance", "{}"},
		{"smpc-instances", "/zia/api/v1/smpcInstance", `{"data":null}`},
	} {
		routes := otherScoutRoutes()
		routes[c.route] = c.body
		handlers := newOtherScoutTestHandlers(t, routes, nil)
		if records, err := handlers[resourceKey{product: resources.ProductZIA, name: c.name}].List(context.Background()); err == nil {
			t.Errorf("zia/%s List() with body %s = %d records, want an error", c.name, c.body, len(records))
		}
	}
}

// TestStringIDGetsRefuseOtherRoutes: an id that is not one path segment is
// refused before any request, so "x/secrets" cannot reach the API client
// secrets route.
func TestStringIDGetsRefuseOtherRoutes(t *testing.T) {
	t.Parallel()

	requested := map[string]int{}
	handlers := newOtherScoutTestHandlers(t, otherScoutRoutes(), requested)
	for _, id := range []string{"6/secrets", "../6", "..", ".", "a..b", "6?x=1", "6#x", "6%2Fsecrets", `6\secrets`} {
		_, err := handlers[resourceKey{product: resources.ProductZidentity, name: "api-clients"}].Get(context.Background(), id)
		if !errors.Is(err, ErrInvalidResourceID) {
			t.Errorf("api-clients Get(%q) error = %v, want ErrInvalidResourceID", id, err)
		}
	}
	for path, n := range requested {
		if strings.HasPrefix(path, "unexpected:") || strings.Contains(path, "secrets") {
			t.Errorf("request to %s (%d), want none", path, n)
		}
	}
	for _, id := range []string{"6", " 9007199254740993 ", "abc-DEF_1.2"} {
		if _, err := sdkPathSegmentID(id); err != nil {
			t.Errorf("sdkPathSegmentID(%q) error = %v, want nil", id, err)
		}
	}
}
