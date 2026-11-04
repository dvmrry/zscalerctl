package zscaler

import (
	"reflect"

	zccwebprivacy "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zcc/services/web_privacy"
	ziadevices "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/devices"
	ziapartners "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/partner_integrations"
	ziasmpc "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/smpc_instance"
	zidapiclients "github.com/zscaler/zscaler-sdk-go/v3/zscaler/ziam/services/api_clients"

	"zscalerctl/internal/resources"
)

// reviewedSDKShapesOtherScout reviews the ZIA, ZCC and Zidentity ledger-scout
// resources (catalog_other_scout.go).
func reviewedSDKShapesOtherScout() []sdkShapeReview {
	return []sdkShapeReview{
		{
			name: "ziadevices.Devices", resource: resources.ProductZIA, resourceName: resourceZIARegisteredDevices,
			typ: reflect.TypeFor[ziadevices.Devices](),
			catalogFields: []string{"id", "name", "active", "deleted", "rooted", "os", "version", "vendor", "model", "locale",
				"hostname", "udid", "hardwareId", "macAddress", "user", "firstRegistrationTimestamp",
				"lastRegistrationTimestamp", "unRegistrationTimestamp"},
		},
		{
			name: "ziapartners.PartnerIntegrations", resource: resources.ProductZIA, resourceName: resourceZIAIntegrationPartners,
			typ:           reflect.TypeFor[ziapartners.PartnerIntegrations](),
			catalogFields: []string{"id", "name", "type", "disabled"},
		},
		{
			name: "ziasmpc.Data", resource: resources.ProductZIA, resourceName: resourceZIASMPCInstances,
			typ: reflect.TypeFor[ziasmpc.Data](),
			catalogFields: []string{"id", "datacenterName", "city", "countryCode", "zscmIsolationEnabled",
				"zscmIsolationEnabledApiFlag", "connectionStatus", "dcIsolationStatus", "nodes"},
		},
		{
			name: "zccwebprivacy.WebPrivacyInfo", resource: resources.ProductZCC, resourceName: resourceZCCWebPrivacy,
			typ: reflect.TypeFor[zccwebprivacy.WebPrivacyInfo](),
			catalogFields: []string{"id", "active", "collectUserInfo", "collectMachineHostname", "collectZdxLocation",
				"enablePacketCapture", "disableCrashlytics", "overrideT2ProtocolSetting", "restrictRemotePacketCapture",
				"grantAccessToZscalerLogFolder", "exportLogsForNonAdmin", "enableAutoLogSnippet", "enforceSecurePacUrls",
				"enableFQDNMatchForVpnBypasses"},
		},
		{
			name: "zidapiclients.APIClients", resource: resources.ProductZidentity, resourceName: resourceZidentityAPIClients,
			typ: reflect.TypeFor[zidapiclients.APIClients](),
			catalogFields: []string{"id", "name", "description", "status", "accessTokenLifeTime",
				"clientAuthentication", "clientResources"},
		},
	}
}
