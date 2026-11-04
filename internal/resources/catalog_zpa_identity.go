package resources

// ZPA administrators, admin roles and identity providers, with the SAML and SCIM attributes
// and SCIM groups policy rules match on. Group, attribute and role names
// render in share, like ZIA and Zidentity groups; anything that carries a
// tenant domain, entity id, URL or external identifier is standard only.
// Role permissions render in share: their groups, classes and masks are
// Zscaler-defined, not tenant values. Administrators are people: their
// usernames, names, emails, phone numbers and time zones are standard only,
// and passwords and the 2FA token id never render. The 2FA flag and type are
// not cataloged: the live route omits them, and a filled-in false would claim
// 2FA is off.
func catalogZPAIdentity() ResourceCatalog {
	return ResourceCatalog{
		{
			Product:    ProductZPA,
			Name:       "admins",
			Operations: ReadOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				sensitiveIdentifierField("username"),
				sensitiveIdentifierField("displayName"),
				sensitiveIdentifierField("email"),
				sensitiveIdentifierField("phoneNumber"),
				freeTextField("comments", "ZPA administrator comments"),
				operationalField("isEnabled", allModes()),
				operationalField("isLocked", allModes()),
				operationalField("localLoginDisabled", allModes()),
				namedOperationalField("forcePwdChange", allModes(), "boolean: whether the admin must change their password, not a password"),
				namedOperationalField("pinSession", allModes(), "boolean: whether the admin session is pinned, not a PIN"),
				operationalField("timezone", standardOnlyMode()),
				operationalField("languageCode", allModes()),
				operationalField("eula", allModes()),
				operationalField("roleId", allModes()),
				{
					Name:           "role",
					Classification: ClassTenantConfig,
					AllowedModes:   allModes(),
					Fields:         []FieldSpec{operationalField("id", allModes())},
				},
				operationalField("groupIds", allModes()),
				operationalField("microtenantId", allModes()),
				tenantConfigField("microtenantName", standardShareModes()),
				operationalField("syncVersion", allModes()),
				operationalField("deliveryTag", allModes()),
				operationalField("operationType", allModes()),
				credentialField("password"),
				credentialField("tmpPassword"),
				tokenField("tokenId"),
			},
		},
		{
			Product:    ProductZPA,
			Name:       "roles",
			Operations: ReadOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				freeTextField("description", "ZPA admin role description"),
				operationalField("creationTime", allModes()),
				operationalField("modifiedBy", allModes()),
				operationalField("modifiedTime", allModes()),
				operationalField("microtenantId", allModes()),
				tenantConfigField("microtenantName", standardShareModes()),
				operationalField("customRole", allModes()),
				operationalField("systemRole", allModes()),
				operationalField("restrictedRole", allModes()),
				operationalField("bypassRemoteAssistanceCheck", allModes()),
				// The API documents neither; they may list admin or API key
				// names, so neither leaves standard mode.
				sensitiveIdentifierField("users"),
				{
					Name:                "apiKeys",
					Classification:      ClassSensitiveIdentifier,
					AllowedModes:        standardOnlyMode(),
					SensitiveNameReason: "the role's API keys as the API summarizes them, never key material",
				},
				{
					Name:           "classPermissionGroups",
					Classification: ClassTenantConfig,
					AllowedModes:   standardShareModes(),
					Fields: []FieldSpec{
						operationalField("id", allModes()),
						tenantConfigField("name", standardShareModes()),
						operationalField("creationTime", allModes()),
						operationalField("modifiedBy", allModes()),
						operationalField("modifiedTime", allModes()),
						operationalField("hidden", allModes()),
						operationalField("internal", allModes()),
						operationalField("localScopePermissionGroup", allModes()),
						{
							Name:           "classPermissions",
							Classification: ClassTenantConfig,
							AllowedModes:   standardShareModes(),
							Fields: []FieldSpec{
								operationalField("id", allModes()),
								operationalField("creationTime", allModes()),
								operationalField("modifiedBy", allModes()),
								operationalField("modifiedTime", allModes()),
								{
									Name:           "permission",
									Classification: ClassTenantConfig,
									AllowedModes:   standardShareModes(),
									Fields: []FieldSpec{
										tenantConfigField("type", standardShareModes()),
										operationalField("mask", allModes()),
										operationalField("maxMask", allModes()),
									},
								},
								zpaRoleClassTypeField(),
							},
						},
					},
				},
				{
					Name:           "permissions",
					Classification: ClassTenantConfig,
					AllowedModes:   standardShareModes(),
					Fields: []FieldSpec{
						operationalField("id", allModes()),
						operationalField("permissionMask", allModes()),
						operationalField("role", allModes()),
						sensitiveIdentifierField("customerId"),
						operationalField("creationTime", allModes()),
						operationalField("modifiedBy", allModes()),
						operationalField("modifiedTime", allModes()),
						zpaRoleClassTypeField(),
					},
				},
			},
		},
		{
			Product:    ProductZPA,
			Name:       "idps",
			Operations: ReadOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				freeTextField("description", "ZPA identity provider description"),
				operationalField("enabled", allModes()),
				tenantConfigField("ssoType", standardShareModes()),
				operationalField("scimEnabled", allModes()),
				operationalField("enableScimBasedPolicy", allModes()),
				operationalField("disableSamlBasedPolicy", allModes()),
				operationalField("forceAuth", allModes()),
				operationalField("reauthOnUserUpdate", allModes()),
				operationalField("redirectBinding", allModes()),
				operationalField("loginHint", allModes()),
				operationalField("useCustomSPMetadata", allModes()),
				operationalField("autoProvision", allModes()),
				operationalField("enableArbitraryAuthDomains", allModes()),
				operationalField("signSamlRequest", allModes()),
				tenantConfigField("loginNameAttribute", standardShareModes()),
				namedOperationalField("scimSharedSecretExists", allModes(), "whether a SCIM shared secret is set, not the secret"),
				operationalField("adminSpSigningCertId", allModes()),
				operationalField("userSpSigningCertId", allModes()),
				sensitiveIdentifierField("domainList"),
				sensitiveIdentifierField("idpEntityId"),
				sensitiveIdentifierField("loginUrl"),
				sensitiveIdentifierField("scimServiceProviderEndpoint"),
				zpaIdPMetadataField("adminMetadata"),
				zpaIdPMetadataField("userMetadata"),
				operationalField("creationTime", allModes()),
				operationalField("modifiedBy", allModes()),
				operationalField("modifiedTime", allModes()),
			},
		},
		{
			Product:    ProductZPA,
			Name:       "saml-attributes",
			Operations: ReadOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				// Custom claim names can embed an IdP org or tenant domain.
				sensitiveIdentifierField("samlName"),
				operationalField("idpId", allModes()),
				tenantConfigField("idpName", standardShareModes()),
				operationalField("userAttribute", allModes()),
				operationalField("delta", allModes()),
				operationalField("creationTime", allModes()),
				operationalField("modifiedBy", allModes()),
				operationalField("modifiedTime", allModes()),
			},
		},
		{
			Product:    ProductZPA,
			Name:       "scim-groups",
			Operations: ListOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				operationalField("idpId", allModes()),
				tenantConfigField("idpName", standardShareModes()),
				sensitiveIdentifierField("idpGroupId"),
				sensitiveIdentifierField("internalId"),
				sensitiveIdentifierField("iamIdpId"),
				tenantConfigField("iamIdpName", standardShareModes()),
				operationalField("creationTime", allModes()),
				operationalField("modifiedTime", allModes()),
			},
		},
		{
			Product:    ProductZPA,
			Name:       "scim-attributes",
			Operations: ListOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				freeTextField("description", "ZPA SCIM attribute description"),
				operationalField("idpId", allModes()),
				tenantConfigField("dataType", standardShareModes()),
				// Allowed values an IdP declares; they can be tenant
				// department or group names.
				sensitiveIdentifierField("canonicalValues"),
				operationalField("caseSensitive", allModes()),
				operationalField("multivalued", allModes()),
				operationalField("required", allModes()),
				operationalField("uniqueness", allModes()),
				operationalField("mutability", allModes()),
				operationalField("returned", allModes()),
				// Custom schema URNs can embed an org name.
				sensitiveIdentifierField("schemaURI"),
				operationalField("creationTime", allModes()),
				operationalField("modifiedBy", allModes()),
				operationalField("modifiedTime", allModes()),
			},
		},
	}
}

// zpaRoleClassTypeField is the permission class a role permission applies
// to; its names are Zscaler-defined.
func zpaRoleClassTypeField() FieldSpec {
	return FieldSpec{
		Name:           "classType",
		Classification: ClassTenantConfig,
		AllowedModes:   standardShareModes(),
		Fields: []FieldSpec{
			operationalField("id", allModes()),
			tenantConfigField("aclClass", standardShareModes()),
			tenantConfigField("friendlyName", standardShareModes()),
			operationalField("localScopeMask", allModes()),
			sensitiveIdentifierField("customerId"),
			operationalField("creationTime", allModes()),
			operationalField("modifiedBy", allModes()),
			operationalField("modifiedTime", allModes()),
		},
	}
}

// zpaIdPMetadataField is the service-provider metadata ZPA publishes for an
// IdP; every URL and the entity id embed the tenant's customer id.
func zpaIdPMetadataField(name string) FieldSpec {
	return FieldSpec{
		Name:           name,
		Classification: ClassSensitiveIdentifier,
		AllowedModes:   standardOnlyMode(),
		Fields: sensitiveIdentifierFields(
			"certificateUrl", "spBaseUrl", "spEntityId", "spMetadataUrl", "spPostUrl",
		),
	}
}
