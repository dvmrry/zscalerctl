package resources

// ZIA config surfaces first seen in the SDK v3.8.55 scout: DNS application
// groups, endpoint and email DLP rules, alert definitions, HTTP header
// profiles, adaptive access profiles and the web DLP global options. Rule shapes follow
// dlp-web-rules: scope references render their ids in share mode, user and
// auditor identities stay standard-only.
func catalogZIASDKv3855() ResourceCatalog {
	return ResourceCatalog{
		{
			Product:    ProductZIA,
			Name:       "dns-application-groups",
			Operations: ReadOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				tenantConfigField("dnsApplications", standardShareModes()),
				freeTextField("description", "ZIA DNS application group description"),
			},
		},
		{
			Product:    ProductZIA,
			Name:       "endpoint-dlp-rules",
			Operations: ReadOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				freeTextField("description", "ZIA endpoint DLP rule description"),
				operationalField("state", allModes()),
				operationalField("order", allModes()),
				operationalField("rank", allModes()),
				operationalField("parentRule", allModes()),
				operationalField("lastModifiedTime", allModes()),
				ziaAdminRefField("lastModifiedBy"),
				tenantConfigField("action", standardShareModes()),
				tenantConfigField("severity", standardShareModes()),
				tenantConfigField("fileTypes", standardShareModes()),
				tenantConfigField("dataTransferMethod", standardShareModes()),
				tenantConfigField("deviceTrustLevels", standardShareModes()),
				tenantConfigField("networkType", standardShareModes()),
				operationalField("minSize", allModes()),
				operationalField("withoutContentInspection", allModes()),
				operationalField("eunEnabled", allModes()),
				sensitiveIdentifierField("eunTemplateId"),
				sensitiveIdentifierField("ucTemplateId"),
				sensitiveIdentifierField("externalAuditorEmail"),
				tenantConfigField("userRiskScoreLevels", standardOnlyMode()),
				idNameField("auditor", standardOnlyMode()),
				idNameField("notificationTemplate", standardOnlyMode()),
				ziaDLPReceiverField(),
				idNameExtensionsField("resources", standardOnlyMode()),
				idNameExtensionsField("resourceGroups", standardOnlyMode()),
				idNameExtensionsField("labels", standardOnlyMode()),
				idNameExtensionsField("dlpEngines", standardOnlyMode()),
				idNameExtensionsField("users", standardOnlyMode()),
				idNameExtensionsField("groups", standardOnlyMode()),
				idNameExtensionsField("departments", standardOnlyMode()),
				idNameExtensionsField("devices", standardOnlyMode()),
				idNameExtensionsField("deviceGroups", standardOnlyMode()),
				{
					Name:           "endPointApplications",
					Classification: ClassTenantConfig,
					AllowedModes:   standardShareModes(),
					Fields:         ziaEndpointApplicationFields(),
				},
				{
					Name:           "endPointApplicationGroups",
					Classification: ClassTenantConfig,
					AllowedModes:   standardShareModes(),
					Fields:         ziaEndpointApplicationGroupFields(),
				},
				// Sub-rules are whole endpoint DLP rules (a recursive SDK type).
				unstructuredField("subRules"),
			},
		},
		{
			Product:    ProductZIA,
			Name:       "email-dlp-rules",
			Operations: ReadOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				freeTextField("description", "ZIA outbound email DLP rule description"),
				operationalField("state", allModes()),
				operationalField("order", allModes()),
				operationalField("parentRule", allModes()),
				operationalField("lastModifiedTime", allModes()),
				ziaAdminRefField("lastModifiedBy"),
				tenantConfigField("action", standardShareModes()),
				tenantConfigField("severity", standardShareModes()),
				tenantConfigField("fileTypes", standardShareModes()),
				tenantConfigField("contentLocations", standardShareModes()),
				operationalField("minSize", allModes()),
				operationalField("withoutContentInspection", allModes()),
				sensitiveIdentifierField("externalAuditorEmail"),
				freeTextField("customHeader", "ZIA outbound email DLP custom header"),
				tenantConfigField("userRiskScoreLevels", standardOnlyMode()),
				idNameField("auditor", standardOnlyMode()),
				idNameField("notificationTemplate", standardOnlyMode()),
				ziaDLPReceiverField(),
				idNameExtensionsField("labels", standardOnlyMode()),
				idNameExtensionsField("dlpEngines", standardOnlyMode()),
				idNameExtensionsField("timeWindows", standardOnlyMode()),
				idNameExtensionsField("users", standardOnlyMode()),
				idNameExtensionsField("groups", standardOnlyMode()),
				idNameExtensionsField("departments", standardOnlyMode()),
				idNameExtensionsField("excludedUsers", standardOnlyMode()),
				idNameExtensionsField("excludedGroups", standardOnlyMode()),
				idNameExtensionsField("excludedDepartments", standardOnlyMode()),
				idNameExtensionsField("includedDomainProfiles", standardOnlyMode()),
				idNameExtensionsField("emailRecipientProfiles", standardOnlyMode()),
				idNameExtensionsField("emailTenants", standardOnlyMode()),
				// Sub-rules are whole email DLP rules (a recursive SDK type).
				unstructuredField("subRules"),
			},
		},
		{
			Product:    ProductZIA,
			Name:       "alert-definitions",
			Operations: ReadOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("alertName", standardShareModes()),
				operationalField("status", allModes()),
				tenantConfigField("occurrence", standardShareModes()),
				tenantConfigField("trafficChangePercent", standardShareModes()),
				tenantConfigField("interval", standardShareModes()),
				tenantConfigField("scope", standardShareModes()),
				tenantConfigField("severity", standardShareModes()),
				// The entity the alert watches can be a user, so its name is
				// standard-only like rule user lists.
				idNameExtensionsField("entity", standardOnlyMode()),
				freeTextField("comments", "ZIA alert definition comments"),
			},
		},
		{
			Product:    ProductZIA,
			Name:       "http-header-profiles",
			Operations: ListOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				freeTextField("description", "ZIA HTTP header profile description"),
				operationalField("slotId", allModes()),
				operationalField("deleted", allModes()),
				operationalField("profileReadyForUse", allModes()),
				{
					Name:           "httpHeaderProfileCriteria",
					Classification: ClassTenantConfig,
					AllowedModes:   standardShareModes(),
					Fields: []FieldSpec{
						operationalField("id", allModes()),
						tenantConfigField("header", standardShareModes()),
						operationalField("operator", allModes()),
						tenantConfigField("categoryBitmap", standardShareModes()),
						tenantConfigField("cloudAppBitmap", standardShareModes()),
						tenantConfigField("userAgent", standardShareModes()),
						tenantConfigField("userAgentBitmap", standardShareModes()),
						tenantConfigField("userAgentVersion", standardShareModes()),
					},
				},
			},
		},
		{
			Product:    ProductZIA,
			Name:       "http-header-action-profiles",
			Operations: ListOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				freeTextField("description", "ZIA HTTP header action profile description"),
				operationalField("slotId", allModes()),
				operationalField("deleted", allModes()),
				operationalField("profileReadyForUse", allModes()),
				{
					Name:           "httpHeaderActionProfileKeys",
					Classification: ClassTenantConfig,
					AllowedModes:   standardShareModes(),
					Fields: []FieldSpec{
						operationalField("id", allModes()),
						tenantConfigField("key", standardShareModes()),
						// Inserted header values can carry tenant tokens or
						// access keys, so they never render.
						credentialField("value"),
					},
				},
			},
		},
		{
			Product:    ProductZIA,
			Name:       "adaptive-access-profiles",
			Operations: ListOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				operationalField("type", allModes()),
				operationalField("aapIndex", allModes()),
				operationalField("iamAapId", allModes()),
				operationalField("deleted", allModes()),
			},
		},
		{
			Product:    ProductZIA,
			Name:       "dlp-global-options",
			Operations: ShowOperation(),
			Fields: []FieldSpec{
				tenantConfigField("applications", standardShareModes()),
				sensitiveIdentifierField("urls"),
				idNameExtensionsField("urlCategories", standardOnlyMode()),
				tenantConfigField("httpGetCustomUrlCategories", standardShareModes()),
				tenantConfigField("exemptUrlEncodedData", standardShareModes()),
				tenantConfigField("enableNpkEdmTemplates", standardShareModes()),
				tenantConfigField("enableNpkEdmTemplatesForOrg", standardShareModes()),
				tenantConfigField("enableInlineDlpOcr", standardShareModes()),
				tenantConfigField("enableCasbOcr", standardShareModes()),
				tenantConfigField("enableEmailDlpOcr", standardShareModes()),
				tenantConfigField("enableEvaluateAllDlpRules", standardShareModes()),
				tenantConfigField("enableEdmPopularFormat", standardShareModes()),
			},
		},
	}
}

// ziaDLPReceiverField is the incident receiver on endpoint and email DLP rules
// (the dlp_web_rules.Receiver shape).
func ziaDLPReceiverField() FieldSpec {
	return FieldSpec{
		Name:           "receiver",
		Classification: ClassTenantConfig,
		AllowedModes:   standardOnlyMode(),
		Fields: []FieldSpec{
			operationalField("id", allModes()),
			tenantConfigField("name", standardShareModes()),
			operationalField("type", allModes()),
			idNameExtensionsField("tenant", standardOnlyMode()),
		},
	}
}

// ziaEndpointApplicationFields is an endpoint DLP application: an admin
// defined or catalog application, named by its binary.
func ziaEndpointApplicationFields() []FieldSpec {
	return []FieldSpec{
		operationalField("resourceId", allModes()),
		tenantConfigField("applicationName", standardShareModes()),
		freeTextField("description", "ZIA endpoint application description"),
		operationalField("osType", allModes()),
		operationalField("applicationType", allModes()),
		// Bundle ids and file names of in-house applications carry the
		// company's domain and internal tool names, so they stay local like
		// the ZCC process-based-apps file names.
		sensitiveIdentifierField("bundleID"),
		sensitiveIdentifierField("filename"),
		sensitiveIdentifierField("originalFileName"),
		operationalField("digitallySigned", allModes()),
		operationalField("modUId", allModes()),
		operationalField("lastModifiedTime", allModes()),
		operationalField("zappId", allModes()),
		operationalField("deleted", allModes()),
		ziaEndpointApplicationVersionField("version"),
		ziaEndpointApplicationVersionField("versions"),
	}
}

func ziaEndpointApplicationVersionField(name string) FieldSpec {
	return FieldSpec{
		Name:           name,
		Classification: ClassOperational,
		AllowedModes:   allModes(),
		Fields: fieldList(
			operationalFields(
				"version",
				"z_ver_id_md32",
				"threat_type",
				"threat_level",
				"code_signing_certificate_status",
				"threatLevelUpdated",
			),
			sensitiveIdentifierFields("bundleID"),
		),
	}
}

func ziaEndpointApplicationGroupFields() []FieldSpec {
	return []FieldSpec{
		operationalField("groupId", allModes()),
		tenantConfigField("name", standardShareModes()),
		freeTextField("description", "ZIA endpoint application group description"),
		operationalField("modUId", allModes()),
		operationalField("lastModifiedTime", allModes()),
		{
			Name:           "endPointApplications",
			Classification: ClassTenantConfig,
			AllowedModes:   standardShareModes(),
			Fields:         ziaEndpointApplicationFields(),
		},
	}
}
