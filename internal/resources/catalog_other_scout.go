package resources

// ZIA, ZCC and Zidentity resources from the SDK ledger scout. Device
// hardware identifiers and owners are standard only, as on zia/devices; API
// client key material and certificates never render.
func catalogOtherScout() ResourceCatalog {
	return ResourceCatalog{
		{
			Product:    ProductZIA,
			Name:       "registered-devices",
			Operations: ReadOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				sensitiveIdentifierField("name"),
				operationalField("active", allModes()),
				operationalField("deleted", allModes()),
				operationalField("rooted", allModes()),
				tenantConfigField("os", standardShareModes()),
				tenantConfigField("version", standardShareModes()),
				tenantConfigField("vendor", standardShareModes()),
				sensitiveIdentifierField("model"),
				tenantConfigField("locale", standardShareModes()),
				sensitiveIdentifierField("hostname"),
				sensitiveIdentifierField("udid"),
				sensitiveIdentifierField("hardwareId"),
				sensitiveIdentifierField("macAddress"),
				// The owner stays standard only, id included, like
				// zia/devices ownerUserId: share output never ties a device
				// to a user.
				{
					Name:           "user",
					Classification: ClassSensitiveIdentifier,
					AllowedModes:   standardOnlyMode(),
					Fields: []FieldSpec{
						operationalField("id", standardOnlyMode()),
						sensitiveIdentifierField("name"),
						unstructuredField("extensions"),
					},
				},
				operationalField("firstRegistrationTimestamp", allModes()),
				operationalField("lastRegistrationTimestamp", allModes()),
				operationalField("unRegistrationTimestamp", allModes()),
			},
		},
		{
			Product:    ProductZIA,
			Name:       "integration-partners",
			Operations: ListOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				tenantConfigField("type", standardShareModes()),
				operationalField("disabled", allModes()),
			},
		},
		{
			Product:    ProductZIA,
			Name:       "smpc-instances",
			Operations: ListOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				operationalField("datacenterName", allModes()),
				operationalField("city", allModes()),
				operationalField("countryCode", allModes()),
				operationalField("zscmIsolationEnabled", allModes()),
				operationalField("zscmIsolationEnabledApiFlag", allModes()),
				operationalField("connectionStatus", allModes()),
				operationalField("dcIsolationStatus", allModes()),
				{
					Name:           "nodes",
					Classification: ClassOperational,
					AllowedModes:   allModes(),
					Fields: []FieldSpec{
						operationalField("id", allModes()),
						operationalField("nodeName", allModes()),
						operationalField("instanceId", allModes()),
						operationalField("adminIsolationEnabled", allModes()),
						operationalField("connectionState", allModes()),
						operationalField("lastTimestamp", allModes()),
					},
				},
			},
		},
		{
			Product:    ProductZCC,
			Name:       "web-privacy",
			Operations: ListOperations(),
			Shape:      ShapeSingleton,
			Fields: append([]FieldSpec{operationalField("id", allModes())}, tenantConfigFields(
				"active", "collectUserInfo", "collectMachineHostname", "collectZdxLocation", "enablePacketCapture",
				"disableCrashlytics", "overrideT2ProtocolSetting", "restrictRemotePacketCapture",
				"grantAccessToZscalerLogFolder", "exportLogsForNonAdmin", "enableAutoLogSnippet",
				"enforceSecurePacUrls", "enableFQDNMatchForVpnBypasses",
			)...),
		},
		{
			Product:    ProductZidentity,
			Name:       "api-clients",
			Operations: ReadOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				freeTextField("description", "Zidentity API client description"),
				operationalField("status", allModes()),
				namedOperationalField("accessTokenLifeTime", allModes(), "the lifetime of issued access tokens in seconds, not a token"),
				{
					Name:           "clientAuthentication",
					Classification: ClassTenantConfig,
					AllowedModes:   standardShareModes(),
					Fields: []FieldSpec{
						tenantConfigField("authType", standardShareModes()),
						sensitiveIdentifierField("clientJWKsUrl"),
						{
							Name:           "publicKeys",
							Classification: ClassTenantConfig,
							AllowedModes:   standardOnlyMode(),
							Fields: []FieldSpec{
								sensitiveIdentifierField("keyName"),
								opaqueBlobField("keyValue"),
							},
						},
						// clientCertificates (certificate content only) is not
						// cataloged, so it never renders.
					},
				},
				{
					Name:           "clientResources",
					Classification: ClassTenantConfig,
					AllowedModes:   standardShareModes(),
					Fields: []FieldSpec{
						operationalField("id", allModes()),
						tenantConfigField("name", standardShareModes()),
						operationalField("defaultApi", allModes()),
						{
							Name:           "selectedScopes",
							Classification: ClassTenantConfig,
							AllowedModes:   standardShareModes(),
							Fields: []FieldSpec{
								operationalField("id", allModes()),
								tenantConfigField("name", standardShareModes()),
							},
						},
					},
				},
			},
		},
	}
}
