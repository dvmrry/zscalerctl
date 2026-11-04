package zscaler

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	zpaadminsso "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/admin_sso_controller"
	zpaclientsettings "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/client_settings"
	zpacommon "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/common"
	zpacustomconfig "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/custom_config_controller"
	zpacustomer "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/customer_controller"
	zpadrtool "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/customer_dr_tool"
	zpamanagedbrowser "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/managed_browser"
	zpaoneidentity "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/one_identity_controller"

	"zscalerctl/internal/resources"
)

// reviewedSDKShapesZPAScout reviews the ZPA ledger-scout resources
// (catalog_zpa_scout.go).
func reviewedSDKShapesZPAScout() []sdkShapeReview {
	summary := func(name string) sdkShapeReview {
		return sdkShapeReview{
			name:          "zpaSummary/" + name,
			resource:      resources.ProductZPA,
			resourceName:  name,
			typ:           reflect.TypeFor[zpaSummary](),
			catalogFields: []string{"id", "name", "enabled"},
		}
	}
	return []sdkShapeReview{
		summary(resourceZPABranchConnectorGroups),
		summary(resourceZPAExtranetPartners),
		summary(resourceZPALocationSummaries),
		summary(resourceZPAWorkloadTagGroups),
		{
			name: "zpaDRToolVersion", resource: resources.ProductZPA, resourceName: resourceZPADRToolVersions,
			typ:           reflect.TypeFor[zpaDRToolVersion](),
			catalogFields: []string{"id", "name", "platform", "version", "latest", "customerId", "creationTime", "modifiedBy", "modifiedTime"},
		},
		{
			name: "zpaManagedBrowserProfile", resource: resources.ProductZPA, resourceName: resourceZPAManagedBrowserProfiles,
			typ: reflect.TypeFor[zpaManagedBrowserProfile](),
			catalogFields: []string{"id", "name", "description", "browserType", "customerId", "microtenantId", "microtenantName",
				"chromePostureProfile", "creationTime", "modifiedBy", "modifiedTime"},
		},
		{
			name: "zpaclientsettings.ClientSettings", resource: resources.ProductZPA, resourceName: resourceZPAClientSettings,
			typ: reflect.TypeFor[zpaclientsettings.ClientSettings](),
			catalogFields: []string{"id", "name", "clientCertificateType", "enrollmentCertId", "enrollmentCertName",
				"singningCertExpiryInEpochSec", "creationTime", "modifiedBy"},
		},
		{
			name: "zpaAdminSSOOptions", resource: resources.ProductZPA, resourceName: resourceZPAAdminSSOOptions,
			typ:           reflect.TypeFor[zpaAdminSSOOptions](),
			catalogFields: []string{"ssologinonly"},
		},
		{
			name: "zpaSessionTermination", resource: resources.ProductZPA, resourceName: resourceZPASessionTermination,
			typ:           reflect.TypeFor[zpaSessionTermination](),
			catalogFields: []string{"sessionTerminationOnReauth", "allowDisableSessionTerminationOnReauth"},
		},
		{
			name: "zpacustomer.AuthDomain", resource: resources.ProductZPA, resourceName: resourceZPAAuthDomains,
			typ:           reflect.TypeFor[zpacustomer.AuthDomain](),
			catalogFields: []string{"authDomains"},
		},
		{
			name: "zpaoneidentity.OneIdentityController", resource: resources.ProductZPA, resourceName: resourceZPAIAMIdPMapping,
			typ:           reflect.TypeFor[zpaoneidentity.OneIdentityController](),
			catalogFields: []string{"id", "sequence", "iamIdpIdToIamIdMapping"},
		},
		{
			name: "zpaStepUpAuthLevels", resource: resources.ProductZPA, resourceName: resourceZPAStepUpAuthLevels,
			typ:           reflect.TypeFor[zpaStepUpAuthLevels](),
			catalogFields: []string{"levels"},
		},
	}
}

// TestZPAScoutMirrorsMatchSDK keeps each local pointer-bool mirror's JSON
// field names equal to the SDK struct it replaces, so an SDK field added or
// renamed upstream fails here instead of silently going unread.
func TestZPAScoutMirrorsMatchSDK(t *testing.T) {
	t.Parallel()

	jsonNames := func(typ reflect.Type) []string {
		var names []string
		for field := range typ.Fields() {
			name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			names = append(names, name)
		}
		sort.Strings(names)
		return names
	}
	pairs := []struct{ local, sdk reflect.Type }{
		{reflect.TypeFor[zpaSummary](), reflect.TypeFor[zpacommon.CommonSummary]()},
		{reflect.TypeFor[zpaDRToolVersion](), reflect.TypeFor[zpadrtool.CustomerDrTool]()},
		{reflect.TypeFor[zpaManagedBrowserProfile](), reflect.TypeFor[zpamanagedbrowser.ManagedBrowserProfile]()},
		{reflect.TypeFor[zpaChromePostureProfile](), reflect.TypeFor[zpamanagedbrowser.ChromePostureProfile]()},
		{reflect.TypeFor[zpaAdminSSOOptions](), reflect.TypeFor[zpaadminsso.AdminSSOLoginOptions]()},
		{reflect.TypeFor[zpaSessionTermination](), reflect.TypeFor[zpacustomconfig.SessionTerminationOnReath]()},
	}
	for _, pair := range pairs {
		if got, want := jsonNames(pair.local), jsonNames(pair.sdk); strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s JSON fields = %v, want the SDK's %s fields %v", pair.local.Name(), got, pair.sdk, want)
		}
	}
}
