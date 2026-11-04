package zscaler

import (
	"reflect"

	zpaadmin "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/administrator_controller"
	zpaidp "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/idpcontroller"
	zparole "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/role_controller"
	zpasamlattribute "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/samlattribute"
	zpascimattribute "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/scimattributeheader"
	zpascimgroup "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zpa/services/scimgroup"

	"zscalerctl/internal/resources"
)

// reviewedSDKShapesZPAIdentity reviews the ZPA administrator, admin-role and
// identity resources (catalog_zpa_identity.go).
func reviewedSDKShapesZPAIdentity() []sdkShapeReview {
	return []sdkShapeReview{
		{
			name:         "zpaadmin.AdministratorController",
			resource:     resources.ProductZPA,
			resourceName: resourceZPAAdmins,
			typ:          reflect.TypeFor[zpaadmin.AdministratorController](),
			catalogFields: []string{
				"id", "username", "displayName", "email", "phoneNumber", "comments", "isEnabled", "isLocked",
				"localLoginDisabled", "forcePwdChange", "pinSession",
				"timezone", "languageCode", "eula", "roleId", "role", "groupIds", "microtenantId", "microtenantName",
				"syncVersion", "deliveryTag", "operationType", "password", "tmpPassword", "tokenId",
			},
			ignoredFields: map[string]string{
				"twoFactorAuthEnabled": "deliberate: the live list omits it, so the false restoreOmittedFalse fills in would claim 2FA is off",
				"twoFactorAuthType":    "deliberate: the live list omits it, like twoFactorAuthEnabled",
			},
		},
		{
			name:         "zparole.RoleController",
			resource:     resources.ProductZPA,
			resourceName: resourceZPARoles,
			typ:          reflect.TypeFor[zparole.RoleController](),
			catalogFields: []string{
				"id", "name", "description", "creationTime", "modifiedBy", "modifiedTime",
				"microtenantId", "microtenantName", "customRole", "systemRole", "restrictedRole",
				"bypassRemoteAssistanceCheck", "users", "apiKeys", "classPermissionGroups", "permissions",
			},
			ignoredFields: map[string]string{
				"newAuditMessage": "deliberate: audit-diff text for writes; the read route does not return it",
				"oldAuditMessage": "deliberate: audit-diff text for writes; the read route does not return it",
			},
		},
		{
			name:         "zpaidp.IdpController",
			resource:     resources.ProductZPA,
			resourceName: resourceZPAIdPs,
			typ:          reflect.TypeFor[zpaidp.IdpController](),
			catalogFields: []string{
				"id", "name", "description", "enabled", "ssoType", "scimEnabled", "enableScimBasedPolicy",
				"disableSamlBasedPolicy", "forceAuth", "reauthOnUserUpdate", "redirectBinding", "loginHint",
				"useCustomSPMetadata", "autoProvision", "enableArbitraryAuthDomains", "signSamlRequest",
				"loginNameAttribute", "scimSharedSecretExists", "adminSpSigningCertId", "userSpSigningCertId",
				"domainList", "idpEntityId", "loginUrl", "scimServiceProviderEndpoint", "adminMetadata",
				"userMetadata", "creationTime", "modifiedBy", "modifiedTime",
			},
		},
		{
			name:         "zpasamlattribute.SamlAttribute",
			resource:     resources.ProductZPA,
			resourceName: resourceZPASAMLAttributes,
			typ:          reflect.TypeFor[zpasamlattribute.SamlAttribute](),
			catalogFields: []string{
				"id", "name", "samlName", "idpId", "idpName", "userAttribute", "delta",
				"creationTime", "modifiedBy", "modifiedTime",
			},
		},
		{
			name:         "zpascimgroup.ScimGroup",
			resource:     resources.ProductZPA,
			resourceName: resourceZPASCIMGroups,
			typ:          reflect.TypeFor[zpascimgroup.ScimGroup](),
			catalogFields: []string{
				"id", "name", "idpId", "idpName", "idpGroupId", "internalId", "iamIdpId", "iamIdpName",
				"creationTime", "modifiedTime",
			},
		},
		{
			name:         "zpascimattribute.ScimAttributeHeader",
			resource:     resources.ProductZPA,
			resourceName: resourceZPASCIMAttributes,
			typ:          reflect.TypeFor[zpascimattribute.ScimAttributeHeader](),
			catalogFields: []string{
				"id", "name", "description", "idpId", "dataType", "canonicalValues", "caseSensitive",
				"multivalued", "required", "uniqueness", "mutability", "returned", "schemaURI",
				"creationTime", "modifiedBy", "modifiedTime",
			},
		},
	}
}
