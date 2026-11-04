package resources

// ZPA tenant settings and summaries from the SDK ledger scout. Summaries
// are id/name/enabled views; the tenant's own object names render in share
// like other ZPA names, but extranet partner names are other organizations
// and stay standard only. Tenant domains, customer ids and Zidentity org ids
// are standard only.
//
// Booleans read through local pointer-bool mirrors render false as false
// and an absent field as null (the SDK's omitempty dropped a false).
func catalogZPAScout() ResourceCatalog {
	summary := func(name string, nameField FieldSpec) ResourceSpec {
		return ResourceSpec{
			Product:    ProductZPA,
			Name:       name,
			Operations: ListOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				nameField,
				operationalField("enabled", allModes()),
			},
		}
	}
	ownName := tenantConfigField("name", standardShareModes())
	return ResourceCatalog{
		summary("branch-connector-groups", ownName),
		summary("extranet-partners", sensitiveIdentifierField("name")),
		summary("location-summaries", ownName),
		summary("workload-tag-groups", ownName),
		{
			Product:    ProductZPA,
			Name:       "dr-tool-versions",
			Operations: ListOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				operationalField("platform", allModes()),
				operationalField("version", allModes()),
				operationalField("latest", allModes()),
				sensitiveIdentifierField("customerId"),
				operationalField("creationTime", allModes()),
				operationalField("modifiedBy", allModes()),
				operationalField("modifiedTime", allModes()),
			},
		},
		{
			Product:    ProductZPA,
			Name:       "managed-browser-profiles",
			Operations: ListOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				freeTextField("description", "ZPA managed browser profile description"),
				operationalField("browserType", allModes()),
				sensitiveIdentifierField("customerId"),
				operationalField("microtenantId", allModes()),
				tenantConfigField("microtenantName", standardShareModes()),
				{
					Name:           "chromePostureProfile",
					Classification: ClassTenantConfig,
					AllowedModes:   standardShareModes(),
					Fields: []FieldSpec{
						operationalField("id", allModes()),
						operationalField("browserType", allModes()),
						operationalField("crowdStrikeAgent", allModes()),
						operationalField("creationTime", allModes()),
						operationalField("modifiedBy", allModes()),
						operationalField("modifiedTime", allModes()),
					},
				},
				operationalField("creationTime", allModes()),
				operationalField("modifiedBy", allModes()),
				operationalField("modifiedTime", allModes()),
			},
		},
		{
			Product:    ProductZPA,
			Name:       "client-settings",
			Operations: ListOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				tenantConfigField("clientCertificateType", standardShareModes()),
				operationalField("enrollmentCertId", allModes()),
				tenantConfigField("enrollmentCertName", standardShareModes()),
				operationalField("singningCertExpiryInEpochSec", allModes()),
				operationalField("creationTime", allModes()),
				operationalField("modifiedBy", allModes()),
			},
		},
		{
			Product:    ProductZPA,
			Name:       "admin-sso-options",
			Operations: ListOperations(),
			Shape:      ShapeSingleton,
			Fields: []FieldSpec{
				tenantConfigField("ssologinonly", standardShareModes()),
			},
		},
		{
			Product:    ProductZPA,
			Name:       "session-termination-settings",
			Operations: ListOperations(),
			Shape:      ShapeSingleton,
			Fields: []FieldSpec{
				tenantConfigField("sessionTerminationOnReauth", standardShareModes()),
				tenantConfigField("allowDisableSessionTerminationOnReauth", standardShareModes()),
			},
		},
		{
			Product:    ProductZPA,
			Name:       "auth-domains",
			Operations: ListOperations(),
			Shape:      ShapeSingleton,
			Fields: []FieldSpec{
				sensitiveIdentifierField("authDomains"),
			},
		},
		{
			Product:    ProductZPA,
			Name:       "iam-idp-mapping",
			Operations: ListOperations(),
			Shape:      ShapeSingleton,
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				operationalField("sequence", allModes()),
				{
					Name:           "iamIdpIdToIamIdMapping",
					Classification: ClassTenantConfig,
					AllowedModes:   standardShareModes(),
					Fields: []FieldSpec{
						operationalField("deliveryTag", allModes()),
						sensitiveIdentifierField("orgId"),
						{
							Name:           "mappings",
							Classification: ClassTenantConfig,
							AllowedModes:   standardShareModes(),
							Fields: []FieldSpec{
								sensitiveIdentifierField("iamIdpId"),
								operationalField("idpId", allModes()),
								tenantConfigField("idpName", standardShareModes()),
							},
						},
					},
				},
			},
		},
		{
			Product:    ProductZPA,
			Name:       "step-up-auth-levels",
			Operations: ListOperations(),
			Shape:      ShapeSingleton,
			Fields: []FieldSpec{
				tenantConfigField("levels", standardShareModes()),
			},
		},
	}
}
