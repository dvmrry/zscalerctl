package resources

// Reference targets for the ZPA admin-role and identity batch
// (catalog_zpa_identity.go); see fieldReferences in references.go.
var zpaIdentityFieldReferences = map[string]map[string]referenceTargets{
	"zpa/admins": {
		"roleId":        zpaRef("roles"),
		"role":          zpaRef("roles"),
		"microtenantId": zpaRef("microtenants"),
	},
	"zpa/roles":           {"microtenantId": zpaRef("microtenants")},
	"zpa/saml-attributes": {"idpId": zpaRef("idps")},
	"zpa/scim-groups":     {"idpId": zpaRef("idps")},
	"zpa/scim-attributes": {"idpId": zpaRef("idps")},
}

var zpaIdentityUnmappedReferences = map[string]map[string]string{
	"zpa/admins": {
		"groupIds": "admin groups the API does not document; no catalog resource is known to hold them",
		"tokenId":  "the admin's two-factor token, which never renders",
	},
	"zpa/roles": {
		"classPermissionGroups":                                       "a Zscaler-defined permission group the role grants, not a catalog resource",
		"classPermissionGroups.classPermissions":                      "the group's own permission entries, not a pointer",
		"classPermissionGroups.classPermissions.classType":            "a Zscaler-defined permission class, not a catalog resource",
		"classPermissionGroups.classPermissions.classType.customerId": "the tenant's own customer id",
		"permissions":                      "the role's own permission entries, not a pointer",
		"permissions.customerId":           "the tenant's own customer id",
		"permissions.classType":            "a Zscaler-defined permission class, not a catalog resource",
		"permissions.classType.customerId": "the tenant's own customer id",
	},
	"zpa/idps": {
		"adminSpSigningCertId":     "a service-provider signing certificate; certificates are not read",
		"userSpSigningCertId":      "a service-provider signing certificate; certificates are not read",
		"idpEntityId":              "the IdP's SAML entity identifier, outside the tenant's ZPA objects",
		"adminMetadata.spEntityId": "ZPA's own SAML entity identifier for the IdP",
		"userMetadata.spEntityId":  "ZPA's own SAML entity identifier for the IdP",
	},
	"zpa/scim-groups": {
		"idpGroupId": "the group's id inside the external IdP",
		"internalId": "an internal ZPA id with no read route",
		"iamIdpId":   "a Zidentity IdP id; no Zidentity IdP resource in the catalog",
	},
}
