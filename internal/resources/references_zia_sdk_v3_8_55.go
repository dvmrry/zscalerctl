package resources

// Reference targets for the ZIA SDK v3.8.55 batch (catalog_zia_sdk_v3_8_55.go);
// see fieldReferences in references.go.
var ziaSDKv3855FieldReferences = map[string]map[string]referenceTargets{
	"zia/email-dlp-rules": {
		"departments":            ziaRef("departments"),
		"dlpEngines":             ziaRef("dlp-engines"),
		"emailRecipientProfiles": ziaRef("email-profiles"),
		"excludedDepartments":    ziaRef("departments"),
		"excludedGroups":         ziaRef("groups"),
		"excludedUsers":          ziaRef("users"),
		"groups":                 ziaRef("groups"),
		"includedDomainProfiles": ziaRef("domain-profiles"),
		"labels":                 ziaRef("rule-labels"),
		"lastModifiedBy":         ziaAuditRef("admin-users"),
		"notificationTemplate":   ziaRef("dlp-notification-templates"),
		"users":                  ziaRef("users"),
	},
	"zia/endpoint-dlp-rules": {
		"departments":          ziaRef("departments"),
		"deviceGroups":         ziaRef("device-groups"),
		"devices":              ziaRef("devices"),
		"dlpEngines":           ziaRef("dlp-engines"),
		"groups":               ziaRef("groups"),
		"labels":               ziaRef("rule-labels"),
		"lastModifiedBy":       ziaAuditRef("admin-users"),
		"notificationTemplate": ziaRef("dlp-notification-templates"),
		"users":                ziaRef("users"),
	},
}

var ziaSDKv3855UnmappedReferences = map[string]map[string]string{ // #nosec G101 -- API field names and reasons
	"zia/adaptive-access-profiles": {
		"iamAapId": "the profile's own identity-side id, not a pointer",
	},
	"zia/alert-definitions": {
		"entity": "what the entity names depends on the alert scope; not modeled",
	},
	"zia/dlp-global-options": {
		"urlCategories": "integer ids; url-categories is keyed by string category id",
	},
	"zia/email-dlp-rules": {
		"auditor":         "auditor id; whether admin-users lists it is not verified",
		"emailTenants":    "email tenant ids; match to casb-tenants is not verified",
		"receiver":        "incident receiver; which receiver resource lists it is not verified",
		"receiver.tenant": "the receiver's SaaS tenant, not a record id",
		"timeWindows":     "time-windows and time-intervals both list rule time ranges; which one the ids belong to is not verified",
	},
	"zia/endpoint-dlp-rules": mergeStringMaps(
		map[string]string{
			"auditor":                           "auditor id; whether admin-users lists it is not verified",
			"eunTemplateId":                     "no end-user notification template resource in the catalog",
			"ucTemplateId":                      "no user coaching template resource in the catalog",
			"receiver":                          "incident receiver; which receiver resource lists it is not verified",
			"receiver.tenant":                   "the receiver's SaaS tenant, not a record id",
			"resources":                         "no endpoint resource in the catalog",
			"resourceGroups":                    "no endpoint resource group in the catalog",
			"endPointApplicationGroups.groupId": "endpoint application groups are a declared gap (zia/endpoint-application-groups)",
			"endPointApplicationGroups.modUId":  "the modifying admin's uid; whether admin-users lists it is not verified",
		},
		ziaEndpointApplicationUnmapped("endPointApplications."),
		ziaEndpointApplicationUnmapped("endPointApplicationGroups.endPointApplications."),
	),
	"zia/http-header-action-profiles": {
		"slotId":                      "the profile's header slot index, not a record id",
		"httpHeaderActionProfileKeys": "the profile's own header entries, not a pointer",
	},
	"zia/http-header-profiles": {
		"slotId":                    "the profile's header slot index, not a record id",
		"httpHeaderProfileCriteria": "the profile's own criteria entries, not a pointer",
	},
}

// ziaEndpointApplicationUnmapped covers the id fields of an endpoint
// application embedded in a rule or group. Rules embed catalog and custom
// applications alike, and neither list is read (zia/endpoint-custom-apps is a
// declared gap).
func ziaEndpointApplicationUnmapped(prefix string) map[string]string {
	return map[string]string{
		prefix + "resourceId": "catalog or custom endpoint application id; neither list is read",
		prefix + "modUId":     "the modifying admin's uid; whether admin-users lists it is not verified",
		prefix + "zappId":     "Zscaler application catalog id, not a record id",
	}
}

// mergeStringMaps merges reason tables; a path declared twice is a table
// bug, so it panics at init rather than keeping one reason silently.
func mergeStringMaps(maps ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			if _, dup := out[k]; dup {
				panic("mergeStringMaps: duplicate path " + k)
			}
			out[k] = v
		}
	}
	return out
}
