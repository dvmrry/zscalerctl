package zscaler

import (
	"context"
	"net/http"

	zsdk "github.com/zscaler/zscaler-sdk-go/v3/zscaler"
	zpaappconnectorcontroller "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/appconnectorcontroller"
	zpaappconnectorgroup "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/appconnectorgroup"
	zpaapplicationsegment "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/applicationsegment"
	zpaappsegmentba "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/applicationsegmentbrowseraccess"
	zpaappsegmentinspection "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/applicationsegmentinspection"
	zpaappsegmentpra "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/applicationsegmentpra"
	zpaappservercontroller "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/appservercontroller"
	zpabranchconnector "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/branch_connector"
	zpac2cipranges "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/c2c_ip_ranges"
	zpaclienttypes "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/clienttypes"
	zpacloudconnector "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/cloud_connector"
	zpacloudconnectorgroup "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/cloud_connector_group"
	zpacbizpaprofile "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/cloudbrowserisolation/cbizpaprofile"
	zpaisolationprofile "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/cloudbrowserisolation/isolationprofile"
	zpaconfigoverride "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/config_override"
	zpaversionprofile "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/customerversionprofile"
	zpamachinegroup "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/machinegroup"
	zpamicrotenants "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/microtenants"
	zpaplatforms "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/platforms"
	zpapolicyset "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/policysetcontroller"
	zpapostureprofile "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/postureprofile"
	zpasegmentgroup "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/segmentgroup"
	zpaservergroup "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/servergroup"
	zpaserviceedgecontroller "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/serviceedgecontroller"
	zpaserviceedgegroup "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/serviceedgegroup"
	zpatrustednetwork "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/trustednetwork"
	zpauserportalaup "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/userportal/aup"
	zpauserportal "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/userportal/portal_controller"
	zpauserportallink "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/userportal/portal_link"

	"zscalerctl/internal/resources"
)

func addZPAHandlers(m map[resourceKey]resourceHandler, client sdkClient) {
	entries := map[resourceKey]resourceHandler{
		{product: resources.ProductZPA, name: resourceZPAServerGroups}: newListGetHandler(
			resourceZPAServerGroups,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaservergroup.ServerGroup, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpaservergroup.ServerGroup](ctx, service, zpaAPIV1, "/serverGroup")
			}),
			zpaSDKStringGet(client, zpaservergroup.Get),
			jsonSourceRecord[zpaservergroup.ServerGroup],
		),
		{product: resources.ProductZPA, name: resourceZPAMicrotenants}: newListGetHandler(
			resourceZPAMicrotenants,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpamicrotenants.MicroTenant, *http.Response, error) {
				return getZPAAllPages[zpamicrotenants.MicroTenant](ctx, service, zpaAPIV1, "/microtenants")
			}),
			zpaSDKStringGet(client, zpamicrotenants.Get),
			jsonSourceRecord[zpamicrotenants.MicroTenant],
		),
		{product: resources.ProductZPA, name: resourceZPAVersionProfiles}: newListOnlyHandler(
			resourceZPAVersionProfiles,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaversionprofile.CustomerVersionProfile, *http.Response, error) {
				return getZPAAllPages[zpaversionprofile.CustomerVersionProfile](ctx, service, zpaAPIV1, "/visible/versionProfiles")
			}),
			jsonSourceRecord[zpaversionprofile.CustomerVersionProfile],
		),
		{product: resources.ProductZPA, name: resourceZPAClientTypes}: newSingletonHandler(
			resourceZPAClientTypes,
			zpaSDKShow(client, zpaclienttypes.GetAllClientTypes),
			jsonSourceRecord[zpaclienttypes.ClientTypes],
		),
		{product: resources.ProductZPA, name: resourceZPAPlatforms}: newSingletonHandler(
			resourceZPAPlatforms,
			zpaSDKShow(client, zpaplatforms.GetAllPlatforms),
			jsonSourceRecord[zpaplatforms.Platforms],
		),
		{product: resources.ProductZPA, name: resourceZPAIsolationProfiles}: newListOnlyHandler(
			resourceZPAIsolationProfiles,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaisolationprofile.IsolationProfile, *http.Response, error) {
				return getZPAAllPages[zpaisolationprofile.IsolationProfile](ctx, service, zpaAPIV1, "/isolation/profiles")
			}),
			jsonSourceRecord[zpaisolationprofile.IsolationProfile],
		),
		{product: resources.ProductZPA, name: resourceZPABranchConnectors}: newListOnlyHandler(
			resourceZPABranchConnectors,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpabranchconnector.BranchConnector, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpabranchconnector.BranchConnector](ctx, service, zpaAPIV1, "/branchConnector")
			}),
			jsonSourceRecord[zpabranchconnector.BranchConnector],
		),
		{product: resources.ProductZPA, name: resourceZPAUserPortals}: newListGetHandler(
			resourceZPAUserPortals,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpauserportal.UserPortalController, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpauserportal.UserPortalController](ctx, service, zpaAPIV1, "/userPortal")
			}),
			zpaSDKStringGet(client, zpauserportal.Get),
			jsonSourceRecord[zpauserportal.UserPortalController],
		),
		{product: resources.ProductZPA, name: resourceZPAUserPortalAups}: newListGetHandler(
			resourceZPAUserPortalAups,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpauserportalaup.UserPortalAup, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpauserportalaup.UserPortalAup](ctx, service, zpaAPIV1, "/userportal/aup")
			}),
			zpaSDKStringGet(client, zpauserportalaup.Get),
			jsonSourceRecord[zpauserportalaup.UserPortalAup],
		),
		{product: resources.ProductZPA, name: resourceZPAUserPortalLinks}: newListGetHandler(
			resourceZPAUserPortalLinks,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpauserportallink.UserPortalLink, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpauserportallink.UserPortalLink](ctx, service, zpaAPIV1, "/userPortalLink")
			}),
			zpaSDKStringGet(client, zpauserportallink.Get),
			jsonSourceRecord[zpauserportallink.UserPortalLink],
		),
		{product: resources.ProductZPA, name: resourceZPABrowserAccess}: newListGetHandler(
			resourceZPABrowserAccess,
			zpaSDKList(client, zpaAppSegmentsWhere(func(item zpaappsegmentba.BrowserAccess) bool { return len(item.ClientlessApps) > 0 })),
			zpaSDKStringGet(client, zpaappsegmentba.Get),
			jsonSourceRecord[zpaappsegmentba.BrowserAccess],
		),
		{product: resources.ProductZPA, name: resourceZPAInspectionAppSegments}: newListGetHandler(
			resourceZPAInspectionAppSegments,
			zpaSDKList(client, zpaAppSegmentsWhere(func(item zpaappsegmentinspection.AppSegmentInspection) bool { return len(item.InspectionAppDto) > 0 })),
			zpaSDKStringGet(client, zpaappsegmentinspection.Get),
			jsonSourceRecord[zpaappsegmentinspection.AppSegmentInspection],
		),
		{product: resources.ProductZPA, name: resourceZPAPRAAppSegments}: newListGetHandler(
			resourceZPAPRAAppSegments,
			zpaSDKList(client, zpaAppSegmentsWhere(func(item zpaappsegmentpra.AppSegmentPRA) bool { return len(item.PRAApps) > 0 })),
			zpaSDKStringGet(client, zpaappsegmentpra.Get),
			jsonSourceRecord[zpaappsegmentpra.AppSegmentPRA],
		),
		{product: resources.ProductZPA, name: resourceZPASegmentGroups}: newListGetHandler(
			resourceZPASegmentGroups,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpasegmentgroup.SegmentGroup, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpasegmentgroup.SegmentGroup](ctx, service, zpaAPIV1, "/segmentGroup")
			}),
			zpaSDKStringGet(client, zpasegmentgroup.Get),
			jsonSourceRecord[zpasegmentgroup.SegmentGroup],
		),
		{product: resources.ProductZPA, name: resourceZPAAppSegments}: newListGetHandler(
			resourceZPAAppSegments,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaapplicationsegment.ApplicationSegmentResource, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpaapplicationsegment.ApplicationSegmentResource](ctx, service, zpaAPIV1, "/application")
			}),
			zpaSDKStringGet(client, zpaapplicationsegment.Get),
			applicationSegmentSourceRecord,
		),
		{product: resources.ProductZPA, name: resourceZPAAppConnectors}: newListGetHandler(
			resourceZPAAppConnectors,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaappconnectorcontroller.AppConnector, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpaappconnectorcontroller.AppConnector](ctx, service, zpaAPIV1, "/connector")
			}),
			zpaSDKStringGet(client, zpaappconnectorcontroller.Get),
			jsonSourceRecord[zpaappconnectorcontroller.AppConnector],
		),
		{product: resources.ProductZPA, name: resourceZPAConnectorGrps}: newListGetHandler(
			resourceZPAConnectorGrps,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaappconnectorgroup.AppConnectorGroup, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpaappconnectorgroup.AppConnectorGroup](ctx, service, zpaAPIV1, "/appConnectorGroup")
			}),
			zpaSDKStringGet(client, zpaappconnectorgroup.Get),
			jsonSourceRecord[zpaappconnectorgroup.AppConnectorGroup],
		),
		{product: resources.ProductZPA, name: resourceZPAAppServers}: newListGetHandler(
			resourceZPAAppServers,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaappservercontroller.ApplicationServer, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpaappservercontroller.ApplicationServer](ctx, service, zpaAPIV1, "/server")
			}),
			zpaSDKStringGet(client, zpaappservercontroller.Get),
			jsonSourceRecord[zpaappservercontroller.ApplicationServer],
		),
		{product: resources.ProductZPA, name: resourceZPAMachineGroups}: newListGetHandler(
			resourceZPAMachineGroups,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpamachinegroup.MachineGroup, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpamachinegroup.MachineGroup](ctx, service, zpaAPIV1, "/machineGroup")
			}),
			zpaSDKStringGet(client, zpamachinegroup.Get),
			jsonSourceRecord[zpamachinegroup.MachineGroup],
		),
		{product: resources.ProductZPA, name: resourceZPATrustedNets}: newListGetHandler(
			resourceZPATrustedNets,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpatrustednetwork.TrustedNetwork, *http.Response, error) {
				return getZPAAllPages[zpatrustednetwork.TrustedNetwork](ctx, service, zpaAPIV2, "/network")
			}),
			zpaSDKStringGet(client, zpatrustednetwork.Get),
			jsonSourceRecord[zpatrustednetwork.TrustedNetwork],
		),
		{product: resources.ProductZPA, name: resourceZPAServiceGrps}: newListGetHandler(
			resourceZPAServiceGrps,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaserviceedgegroup.ServiceEdgeGroup, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpaserviceedgegroup.ServiceEdgeGroup](ctx, service, zpaAPIV1, "/serviceEdgeGroup")
			}),
			zpaSDKStringGet(client, zpaserviceedgegroup.Get),
			jsonSourceRecord[zpaserviceedgegroup.ServiceEdgeGroup],
		),
		{product: resources.ProductZPA, name: resourceZPAServiceEdges}: newListGetHandler(
			resourceZPAServiceEdges,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaserviceedgecontroller.ServiceEdgeController, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpaserviceedgecontroller.ServiceEdgeController](ctx, service, zpaAPIV1, "/serviceEdge")
			}),
			zpaSDKStringGet(client, zpaserviceedgecontroller.Get),
			jsonSourceRecord[zpaserviceedgecontroller.ServiceEdgeController],
		),
		{product: resources.ProductZPA, name: resourceZPACloudConnGrps}: newListGetHandler(
			resourceZPACloudConnGrps,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpacloudconnectorgroup.CloudConnectorGroup, *http.Response, error) {
				return getZPAAllPages[zpacloudconnectorgroup.CloudConnectorGroup](ctx, service, zpaAPIV1, "/cloudConnectorGroup")
			}),
			zpaSDKStringGet(client, zpacloudconnectorgroup.Get),
			jsonSourceRecord[zpacloudconnectorgroup.CloudConnectorGroup],
		),
		{product: resources.ProductZPA, name: resourceZPACloudConns}: newListOnlyHandler(
			resourceZPACloudConns,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpacloudconnector.CloudConnector, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpacloudconnector.CloudConnector](ctx, service, zpaAPIV1, "/cloudConnector")
			}),
			jsonSourceRecord[zpacloudconnector.CloudConnector],
		),
		{product: resources.ProductZPA, name: resourceZPAPostureProfs}: newListGetHandler(
			resourceZPAPostureProfs,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpapostureprofile.PostureProfile, *http.Response, error) {
				return getZPAAllPages[zpapostureprofile.PostureProfile](ctx, service, zpaAPIV2, "/posture")
			}),
			zpaSDKStringGet(client, zpapostureprofile.Get),
			jsonSourceRecord[zpapostureprofile.PostureProfile],
		),
		{product: resources.ProductZPA, name: resourceZPACBIZPAProfs}: newListGetHandler(
			resourceZPACBIZPAProfs,
			zpaSDKList(client, zpacbizpaprofile.GetAll),
			// The SDK's Get scans GetAll and reports a miss as a plain error.
			scanGetByStringID(
				zpaSDKList(client, zpacbizpaprofile.GetAll),
				func(profile zpacbizpaprofile.ZPAProfiles) string { return profile.ID },
			),
			jsonSourceRecord[zpacbizpaprofile.ZPAProfiles],
		),
		{product: resources.ProductZPA, name: resourceZPAC2CIPRanges}: newListGetHandler(
			resourceZPAC2CIPRanges,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpac2cipranges.IPRanges, *http.Response, error) {
				items, resp, err := zpac2cipranges.GetAll(ctx, service)
				if err != nil {
					return nil, resp, err
				}
				out := make([]zpac2cipranges.IPRanges, 0, len(items))
				for _, item := range items {
					if item != nil {
						out = append(out, *item)
					}
				}
				return out, resp, nil
			}),
			zpaSDKStringGet(client, zpac2cipranges.Get),
			jsonSourceRecord[zpac2cipranges.IPRanges],
		),
		{product: resources.ProductZPA, name: resourceZPAConfigOvrds}: newListOnlyHandler(
			resourceZPAConfigOvrds,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaconfigoverride.ConfigOverrides, *http.Response, error) {
				return getZPAAllPages[zpaconfigoverride.ConfigOverrides](ctx, service, zpaAPIV1, "/configOverrides")
			}),
			jsonSourceRecord[zpaconfigoverride.ConfigOverrides],
		),
	}
	for name, policyType := range zpaPolicyRuleTypes {
		entries[resourceKey{product: resources.ProductZPA, name: name}] = newListOnlyHandler(
			name,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpapolicyset.PolicyRule, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpapolicyset.PolicyRule](ctx, service, zpaAPIV1, "/policySet/rules/policyType/"+policyType)
			}),
			jsonSourceRecord[zpapolicyset.PolicyRule],
		)
	}
	for k, v := range entries {
		addHandler(m, k, v)
	}
	addZPAIdentityHandlers(m, client)
	addZPAScoutHandlers(m, client)
}

// zpaPolicyRuleTypes maps each ZPA policy-rule resource to the policy type
// it lists. Every type is its own policy set, read through the bounded
// pager at the endpoint the SDK's policysetcontroller.GetAllByType uses; no
// policy-set helper (create, update, reorder, delete) is bound.
var zpaPolicyRuleTypes = map[string]string{ // #nosec G101 -- resource slugs and ZPA policy type names, not credentials
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

// zpaAppSegmentsWhere lists every application segment in the service
// microtenant decoded as T and keeps those matching keep. Browser access,
// inspection and PRA segments are application segments with their own
// populated sub-list; the API has no route that lists only one kind.
func zpaAppSegmentsWhere[T any](keep func(T) bool) func(context.Context, *zsdk.Service) ([]T, *http.Response, error) {
	return func(ctx context.Context, service *zsdk.Service) ([]T, *http.Response, error) {
		items, response, err := getZPAAllPagesForMicrotenant[T](ctx, service, zpaAPIV1, "/application")
		if err != nil {
			return nil, response, err
		}
		filtered := make([]T, 0, len(items))
		for _, item := range items {
			if keep(item) {
				filtered = append(filtered, item)
			}
		}
		return filtered, response, nil
	}
}
