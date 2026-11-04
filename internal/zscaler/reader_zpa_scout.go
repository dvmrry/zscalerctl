package zscaler

import (
	"context"
	"errors"
	"net/http"

	zsdk "github.com/zscaler/zscaler-sdk-go/v3/zscaler"
	zpaclientsettings "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/client_settings"
	zpacustomer "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/customer_controller"
	zpaoneidentity "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/one_identity_controller"

	"zscalerctl/internal/resources"
)

// zpaStepUpAuthLevels wraps the bare string array the step-up route answers.
type zpaStepUpAuthLevels struct {
	Levels []string `json:"levels"`
}

// Local mirrors of SDK structs whose booleans are tagged omitempty, which
// drops a false: "SSO-only login is off" would render as {}. Pointer bools
// without omitempty render false as false and an absent field as null.
// TestZPAScoutMirrorsMatchSDK keeps their JSON fields equal to the SDK's.
type zpaSummary struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Enabled *bool  `json:"enabled"`
}

type zpaDRToolVersion struct {
	CreationTime string `json:"creationTime,omitempty"`
	CustomerID   string `json:"customerId,omitempty"`
	ID           string `json:"id,omitempty"`
	Latest       *bool  `json:"latest"`
	ModifiedBy   string `json:"modifiedBy,omitempty"`
	ModifiedTime string `json:"modifiedTime,omitempty"`
	Name         string `json:"name,omitempty"`
	Platform     string `json:"platform,omitempty"`
	Version      string `json:"version,omitempty"`
}

type zpaManagedBrowserProfile struct {
	BrowserType          string                   `json:"browserType,omitempty"`
	CreationTime         string                   `json:"creationTime,omitempty"`
	CustomerID           string                   `json:"customerId,omitempty"`
	Description          string                   `json:"description,omitempty"`
	ID                   string                   `json:"id,omitempty"`
	ModifiedBy           string                   `json:"modifiedBy,omitempty"`
	ModifiedTime         string                   `json:"modifiedTime,omitempty"`
	Name                 string                   `json:"name,omitempty"`
	MicrotenantID        string                   `json:"microtenantId,omitempty"`
	MicrotenantName      string                   `json:"microtenantName,omitempty"`
	ChromePostureProfile *zpaChromePostureProfile `json:"chromePostureProfile,omitempty"`
}

type zpaChromePostureProfile struct {
	ID               string `json:"id,omitempty"`
	BrowserType      string `json:"browserType,omitempty"`
	CrowdStrikeAgent *bool  `json:"crowdStrikeAgent"`
	CreationTime     string `json:"creationTime,omitempty"`
	ModifiedBy       string `json:"modifiedBy,omitempty"`
	ModifiedTime     string `json:"modifiedTime,omitempty"`
}

type zpaAdminSSOOptions struct {
	SSOLoginOnly *bool `json:"ssologinonly"`
}

type zpaSessionTermination struct {
	AllowDisableSessionTerminationOnReauth *bool `json:"allowDisableSessionTerminationOnReauth"`
	SessionTerminationOnReauth             *bool `json:"sessionTerminationOnReauth"`
}

// errZPASettingMissing: a settings object without the field it exists to
// report reads as a failure, never as unset.
var errZPASettingMissing = errors.New("read zpa settings: a setting is missing from the response")

// addZPAScoutHandlers binds the ZPA tenant settings and summaries the SDK
// ledger scout found answering. Every read goes through zscalerctl's own
// requests at the SDK's routes: paged lists through the bounded pager,
// singletons and bare arrays through zpaGetJSON (client settings answer an
// array, though the SDK decodes one object). No mutator is bound; the ZIA
// cloud credentials beside session termination are never requested.
func addZPAScoutHandlers(m map[resourceKey]resourceHandler, client sdkClient) {
	// The SDK scopes branch connector and workload tag groups to the
	// service microtenant, and extranet partners and location summaries not.
	pagedSummary := func(name, suffix string, microtenantScoped bool) resourceHandler {
		return newListOnlyHandler(
			name,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaSummary, *http.Response, error) {
				if microtenantScoped {
					return getZPAAllPagesForMicrotenant[zpaSummary](ctx, service, zpaAPIV1, suffix)
				}
				return getZPAAllPages[zpaSummary](ctx, service, zpaAPIV1, suffix)
			}),
			jsonSourceRecord[zpaSummary],
		)
	}
	entries := map[resourceKey]resourceHandler{
		{product: resources.ProductZPA, name: resourceZPABranchConnectorGroups}: pagedSummary(resourceZPABranchConnectorGroups, "/branchConnectorGroup/summary", true),
		{product: resources.ProductZPA, name: resourceZPAExtranetPartners}:      pagedSummary(resourceZPAExtranetPartners, "/extranetResource/partner", false),
		{product: resources.ProductZPA, name: resourceZPALocationSummaries}:     pagedSummary(resourceZPALocationSummaries, "/location/summary", false),
		{product: resources.ProductZPA, name: resourceZPAWorkloadTagGroups}:     pagedSummary(resourceZPAWorkloadTagGroups, "/workloadTagGroup/summary", true),
		{product: resources.ProductZPA, name: resourceZPADRToolVersions}: newListOnlyHandler(
			resourceZPADRToolVersions,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaDRToolVersion, *http.Response, error) {
				return getZPAAllPages[zpaDRToolVersion](ctx, service, zpaAPIV1, "/customerDRToolVersion")
			}),
			jsonSourceRecord[zpaDRToolVersion],
		),
		{product: resources.ProductZPA, name: resourceZPAManagedBrowserProfiles}: newListOnlyHandler(
			resourceZPAManagedBrowserProfiles,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaManagedBrowserProfile, *http.Response, error) {
				return getZPAAllPagesForMicrotenant[zpaManagedBrowserProfile](ctx, service, zpaAPIV1, "/managedBrowserProfile/search")
			}),
			jsonSourceRecord[zpaManagedBrowserProfile],
		),
		{product: resources.ProductZPA, name: resourceZPAClientSettings}: newListOnlyHandler(
			resourceZPAClientSettings,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaclientsettings.ClientSettings, *http.Response, error) {
				return getZPAUnpagedList[zpaclientsettings.ClientSettings](ctx, service, zpaAPIV1, "/clientSetting/all", nil)
			}),
			jsonSourceRecord[zpaclientsettings.ClientSettings],
		),
		{product: resources.ProductZPA, name: resourceZPAAdminSSOOptions}: newSingletonHandler(
			resourceZPAAdminSSOOptions,
			zpaSDKShow(client, func(ctx context.Context, service *zsdk.Service) (*zpaAdminSSOOptions, *http.Response, error) {
				item, response, err := getZPASingleton[zpaAdminSSOOptions](ctx, service, "/v2/ssoLoginOptions")
				if err == nil && item.SSOLoginOnly == nil {
					return nil, response, errZPASettingMissing
				}
				return item, response, err
			}),
			jsonSourceRecord[zpaAdminSSOOptions],
		),
		{product: resources.ProductZPA, name: resourceZPASessionTermination}: newSingletonHandler(
			resourceZPASessionTermination,
			zpaSDKShow(client, func(ctx context.Context, service *zsdk.Service) (*zpaSessionTermination, *http.Response, error) {
				item, response, err := getZPASingleton[zpaSessionTermination](ctx, service, "/config/sessionTerminationOnReauth")
				if err == nil && item.SessionTerminationOnReauth == nil {
					return nil, response, errZPASettingMissing
				}
				return item, response, err
			}),
			jsonSourceRecord[zpaSessionTermination],
		),
		{product: resources.ProductZPA, name: resourceZPAAuthDomains}: newSingletonHandler(
			resourceZPAAuthDomains,
			zpaSDKShow(client, func(ctx context.Context, service *zsdk.Service) (*zpacustomer.AuthDomain, *http.Response, error) {
				return getZPASingleton[zpacustomer.AuthDomain](ctx, service, "/authDomains")
			}),
			jsonSourceRecord[zpacustomer.AuthDomain],
		),
		{product: resources.ProductZPA, name: resourceZPAIAMIdPMapping}: newSingletonHandler(
			resourceZPAIAMIdPMapping,
			zpaSDKShow(client, func(ctx context.Context, service *zsdk.Service) (*zpaoneidentity.OneIdentityController, *http.Response, error) {
				return getZPASingleton[zpaoneidentity.OneIdentityController](ctx, service, "/iamidpmapping")
			}),
			jsonSourceRecord[zpaoneidentity.OneIdentityController],
		),
		{product: resources.ProductZPA, name: resourceZPAStepUpAuthLevels}: newSingletonHandler(
			resourceZPAStepUpAuthLevels,
			zpaSDKShow(client, func(ctx context.Context, service *zsdk.Service) (*zpaStepUpAuthLevels, *http.Response, error) {
				levels, response, err := getZPAUnpagedList[string](ctx, service, zpaAPIV1, "/stepupauthlevel", service.MicroTenantID())
				if err != nil {
					return nil, response, err
				}
				return &zpaStepUpAuthLevels{Levels: levels}, response, nil
			}),
			jsonSourceRecord[zpaStepUpAuthLevels],
		),
	}
	for k, v := range entries {
		addHandler(m, k, v)
	}
}
