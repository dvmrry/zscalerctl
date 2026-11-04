package zscaler

import (
	"context"
	"errors"

	zsdk "github.com/zscaler/zscaler-sdk-go/v3/zscaler"
	zccwebprivacy "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zcc/services/web_privacy"
	ziadevices "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/devices"
	ziapartners "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/partner_integrations"
	ziasmpc "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zia/services/smpc_instance"
	zidapiclients "github.com/zscaler/zscaler-sdk-go/v3/zscaler/ziam/services/api_clients"

	"zscalerctl/internal/resources"
)

const zidentityAPIClientsEndpoint = "/ziam/admin/api/v1/api-clients"

// addOtherScoutHandlers binds the ZIA, ZCC and Zidentity reads the SDK ledger
// scout found answering. Records convert through structSourceRecord, which
// keeps an SDK boolean's false (the SDK tags them omitempty). API client
// secrets (api_clients.GetSecrets) are never requested; public keys and
// certificates in the client record never render.
func addOtherScoutHandlers(m map[resourceKey]resourceHandler, client sdkClient) {
	entries := map[resourceKey]resourceHandler{
		{product: resources.ProductZIA, name: resourceZIARegisteredDevices}: newListGetHandler(
			resourceZIARegisteredDevices,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]ziadevices.Devices, error) {
				return getZIAAllPages[ziadevices.Devices](ctx, service, "/zia/api/v1/devices")
			}),
			ziaSDKGet(client, ziadevices.Get),
			structSourceRecord[ziadevices.Devices],
		),
		{product: resources.ProductZIA, name: resourceZIAIntegrationPartners}: newListOnlyHandler(
			resourceZIAIntegrationPartners,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]ziapartners.PartnerIntegrations, error) {
				// The route answers an array; the SDK's GetPartnerIntegrations
				// decodes one object. One unpaged read, as the SDK makes.
				var partners []ziapartners.PartnerIntegrations
				if err := service.Client.Read(ctx, "/zia/api/v1/integrationPartners", &partners); err != nil {
					return nil, err
				}
				if partners == nil {
					return nil, errors.New("read zia integration partners: response is not an array")
				}
				return partners, nil
			}),
			structSourceRecord[ziapartners.PartnerIntegrations],
		),
		{product: resources.ProductZIA, name: resourceZIASMPCInstances}: newListOnlyHandler(
			resourceZIASMPCInstances,
			ziaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]ziasmpc.Data, error) {
				// The SDK's GetAll returns nil for a body without "data"; a
				// missing or null list fails instead of reading as none.
				var instance struct {
					Data *[]ziasmpc.Data `json:"data"`
				}
				if err := service.Client.Read(ctx, "/zia/api/v1/smpcInstance", &instance); err != nil {
					return nil, err
				}
				if instance.Data == nil {
					return nil, errors.New("read zia smpc instances: response has no data list")
				}
				return *instance.Data, nil
			}),
			structSourceRecord[ziasmpc.Data],
		),
		{product: resources.ProductZCC, name: resourceZCCWebPrivacy}: newSingletonHandler(
			resourceZCCWebPrivacy,
			sdkProductShow(resources.ProductZCC, client, zccwebprivacy.GetWebPrivacyInfo),
			structSourceRecord[zccwebprivacy.WebPrivacyInfo],
		),
		{product: resources.ProductZidentity, name: resourceZidentityAPIClients}: newListGetHandler(
			resourceZidentityAPIClients,
			sdkProductList(resources.ProductZidentity, client, func(ctx context.Context, service *zsdk.Service) ([]zidapiclients.APIClients, error) {
				return zidentityListAll[zidapiclients.APIClients](ctx, service, zidentityAPIClientsEndpoint)
			}),
			sdkProductStringGet(resources.ProductZidentity, client, zidapiclients.Get),
			structSourceRecord[zidapiclients.APIClients],
		),
	}
	for k, v := range entries {
		addHandler(m, k, v)
	}
}
