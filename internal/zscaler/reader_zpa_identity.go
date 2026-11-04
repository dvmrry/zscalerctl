package zscaler

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	zsdk "github.com/zscaler/zscaler-sdk-go/v3/zscaler"
	zpaadmin "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/administrator_controller"
	zpaidp "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/idpcontroller"
	zparole "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/role_controller"
	zpasamlattribute "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/samlattribute"
	zpascimattribute "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/scimattributeheader"
	zpascimgroup "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/scimgroup"

	"zscalerctl/internal/resources"
)

// addZPAIdentityHandlers binds the ZPA administrator, admin-role and
// identity-provider reads. Lists go through the bounded pagers at the routes the SDK's GetAll
// helpers use (administrators and roles answer one bare array, read
// unpaged); gets call the
// SDK's Get. SCIM groups and SCIM attributes are per-IdP routes, so their
// lists fan out over every IdP. No create, update or delete helper is bound.
func addZPAIdentityHandlers(m map[resourceKey]resourceHandler, client sdkClient) {
	entries := map[resourceKey]resourceHandler{
		{product: resources.ProductZPA, name: resourceZPAAdmins}: newListGetHandler(
			resourceZPAAdmins,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaadmin.AdministratorController, *http.Response, error) {
				return getZPAUnpagedList[zpaadmin.AdministratorController](ctx, service, zpaAPIV1, "/administrators", service.MicroTenantID())
			}),
			zpaSDKStringGet(client, zpaadmin.Get),
			jsonSourceRecord[zpaadmin.AdministratorController],
		),
		{product: resources.ProductZPA, name: resourceZPARoles}: newListGetHandler(
			resourceZPARoles,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zparole.RoleController, *http.Response, error) {
				return getZPAUnpagedList[zparole.RoleController](ctx, service, zpaAPIV1, "/roles", service.MicroTenantID())
			}),
			zpaSDKStringGet(client, zparole.Get),
			jsonSourceRecord[zparole.RoleController],
		),
		{product: resources.ProductZPA, name: resourceZPAIdPs}: newListGetHandler(
			resourceZPAIdPs,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpaidp.IdpController, *http.Response, error) {
				return getZPAAllPages[zpaidp.IdpController](ctx, service, zpaAPIV2, "/idp")
			}),
			zpaSDKStringGet(client, zpaidp.Get),
			jsonSourceRecord[zpaidp.IdpController],
		),
		{product: resources.ProductZPA, name: resourceZPASAMLAttributes}: newListGetHandler(
			resourceZPASAMLAttributes,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpasamlattribute.SamlAttribute, *http.Response, error) {
				return getZPAAllPages[zpasamlattribute.SamlAttribute](ctx, service, zpaAPIV2, "/samlAttribute")
			}),
			zpaSDKStringGet(client, zpasamlattribute.Get),
			jsonSourceRecord[zpasamlattribute.SamlAttribute],
		),
		{product: resources.ProductZPA, name: resourceZPASCIMGroups}: newListOnlyHandler(
			resourceZPASCIMGroups,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpascimgroup.ScimGroup, *http.Response, error) {
				return getZPAPerIdPAllPages(ctx, service, func(ctx context.Context, idpID string) ([]zpascimgroup.ScimGroup, *http.Response, error) {
					return getZPAUserConfigAllPages[zpascimgroup.ScimGroup](ctx, service, "/scimgroup/idpId/"+url.PathEscape(idpID))
				})
			}),
			jsonSourceRecord[zpascimgroup.ScimGroup],
		),
		{product: resources.ProductZPA, name: resourceZPASCIMAttributes}: newListOnlyHandler(
			resourceZPASCIMAttributes,
			zpaSDKList(client, func(ctx context.Context, service *zsdk.Service) ([]zpascimattribute.ScimAttributeHeader, *http.Response, error) {
				return getZPAPerIdPAllPages(ctx, service, func(ctx context.Context, idpID string) ([]zpascimattribute.ScimAttributeHeader, *http.Response, error) {
					return getZPAAllPages[zpascimattribute.ScimAttributeHeader](ctx, service, zpaAPIV1, "/idp/"+url.PathEscape(idpID)+"/scimattribute")
				})
			}),
			jsonSourceRecord[zpascimattribute.ScimAttributeHeader],
		),
	}
	for k, v := range entries {
		addHandler(m, k, v)
	}
}

// getZPAPerIdPAllPages lists every IdP, then reads one per-IdP list for each
// and concatenates them in IdP order. Records carry their own idpId. Any
// IdP's failure fails the whole list: a partial list is never returned.
func getZPAPerIdPAllPages[T any](
	ctx context.Context,
	service *zsdk.Service,
	perIdP func(ctx context.Context, idpID string) ([]T, *http.Response, error),
) ([]T, *http.Response, error) {
	idps, response, err := getZPAAllPages[zpaidp.IdpController](ctx, service, zpaAPIV2, "/idp")
	if err != nil {
		return nil, response, err
	}
	var all []T
	for _, idp := range idps {
		if idp.ID == "" {
			// The route would become the bare collection, not this IdP's list.
			return nil, response, errors.New("read zpa per-idp list: an idp has no id")
		}
		items, itemResponse, err := perIdP(ctx, idp.ID)
		response = itemResponse
		if err != nil {
			return nil, response, err
		}
		all = append(all, items...)
	}
	return all, response, nil
}
