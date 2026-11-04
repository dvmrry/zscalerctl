package zscaler

import (
	"reflect"

	"zscalerctl/internal/resources"

	zidcommon "github.com/zscaler/zscaler-sdk-go/v3/zscaler/ziam/services/common"
	zidgroups "github.com/zscaler/zscaler-sdk-go/v3/zscaler/ziam/services/groups"
	zidresourceservers "github.com/zscaler/zscaler-sdk-go/v3/zscaler/ziam/services/resource_servers"
	zidusers "github.com/zscaler/zscaler-sdk-go/v3/zscaler/ziam/services/users"
)

func reviewedSDKShapesZidentity() []sdkShapeReview {
	return []sdkShapeReview{
		{
			name:         "zidgroups.Groups",
			resource:     resources.ProductZidentity,
			resourceName: resourceZidentityGroups,
			typ:          reflect.TypeFor[zidgroups.Groups](),
			catalogFields: []string{
				"id",
				"name",
				"description",
				"source",
				"isDynamicGroup",
				"dynamicGroup",
				"adminEntitlementEnabled",
				"serviceEntitlementEnabled",
				"idp",
			},
		},
		{
			name:         "zidcommon.IDNameDisplayName (groups)",
			resource:     resources.ProductZidentity,
			resourceName: resourceZidentityGroups,
			typ:          reflect.TypeFor[zidcommon.IDNameDisplayName](),
			catalogFields: []string{
				"id",
				"name",
				"displayName",
			},
		},
		{
			name:         "zidusers.Users",
			resource:     resources.ProductZidentity,
			resourceName: resourceZidentityUsers,
			typ:          reflect.TypeFor[zidusers.Users](),
			catalogFields: []string{
				"id",
				"source",
				"loginName",
				"displayName",
				"firstName",
				"lastName",
				"primaryEmail",
				"secondaryEmail",
				"status",
				"department",
				"idp",
				"customAttrsInfo",
				"title",
				"division",
				"costCenter",
				"organization",
				"skipMfaUntil",
				"guest",
				"belongInternalDomain",
				"hostedIdp",
				"entityType",
				"groups",
			},
			ignoredFields: mergeIgnoredFields(
				ignoredBecause("deliberate: personal contact data with no configuration value; never read into output",
					"mobilePhone", "primaryPhone", "addresses"),
				ignoredBecause("deliberate: per-user display preference with no configuration value",
					"language", "timeZone"),
				ignoredBecause("deliberate: org-chart relationship returned as a bare user id; not configuration",
					"manager"),
				ignoredBecause("deliberate: mirrors loginName on every user observed; loginName renders",
					"name", "objectName"),
			),
		},
		{
			name:         "zidusers.UserGroup",
			resource:     resources.ProductZidentity,
			resourceName: resourceZidentityUsers,
			typ:          reflect.TypeFor[zidusers.UserGroup](),
			catalogFields: []string{
				"id",
				"name",
			},
			ignoredFields: ignoredBecause("deliberate: group attributes belong to zidentity/groups; a user's membership renders id and name only",
				"source", "status", "dynamicGroup", "emergencyGroup", "serviceEntitlementDisabled",
				"adminEntitlementDisabled", "readOnly", "entityType", "objectName"),
		},
		{
			name:         "zidcommon.IDNameDisplayName (users)",
			resource:     resources.ProductZidentity,
			resourceName: resourceZidentityUsers,
			typ:          reflect.TypeFor[zidcommon.IDNameDisplayName](),
			catalogFields: []string{
				"id",
				"name",
				"displayName",
			},
		},
		{
			name:         "zidresourceservers.ResourceServers",
			resource:     resources.ProductZidentity,
			resourceName: resourceZidentityResourceServers,
			typ:          reflect.TypeFor[zidresourceservers.ResourceServers](),
			catalogFields: []string{
				"id",
				"name",
				"displayName",
				"description",
				"primaryAud",
				"defaultApi",
				"serviceScopes",
			},
		},
		{
			name:         "zidresourceservers.ServiceScopes",
			resource:     resources.ProductZidentity,
			resourceName: resourceZidentityResourceServers,
			typ:          reflect.TypeFor[zidresourceservers.ServiceScopes](),
			catalogFields: []string{
				"service",
				"scopes",
			},
		},
		{
			name:         "zidresourceservers.Service",
			resource:     resources.ProductZidentity,
			resourceName: resourceZidentityResourceServers,
			typ:          reflect.TypeFor[zidresourceservers.Service](),
			catalogFields: []string{
				"id",
				"name",
				"displayName",
				"externalName",
				"cloudName",
				"orgName",
			},
			ignoredFields: ignoredBecause("deliberate: undocumented flag whose meaning is unknown", "zapsOnboarded"),
		},
		{
			name:         "zidresourceservers.Scopes",
			resource:     resources.ProductZidentity,
			resourceName: resourceZidentityResourceServers,
			typ:          reflect.TypeFor[zidresourceservers.Scopes](),
			catalogFields: []string{
				"id",
				"name",
			},
		},
	}
}
