package resources

// Reference targets for the ZPA ledger-scout batch (catalog_zpa_scout.go);
// see fieldReferences in references.go.
var zpaScoutFieldReferences = map[string]map[string]referenceTargets{
	"zpa/managed-browser-profiles": {"microtenantId": zpaRef("microtenants")},
	"zpa/iam-idp-mapping":          {"iamIdpIdToIamIdMapping.mappings.idpId": zpaRef("idps")},
}

var zpaScoutUnmappedReferences = map[string]map[string]string{
	"zpa/dr-tool-versions": {"customerId": "the tenant's own customer id"},
	"zpa/managed-browser-profiles": {
		"customerId":           "the tenant's own customer id",
		"chromePostureProfile": "the profile's own Chrome posture settings, not a pointer",
	},
	"zpa/client-settings": {"enrollmentCertId": "an enrollment certificate; certificates are not read"},
	"zpa/iam-idp-mapping": {
		"iamIdpIdToIamIdMapping.orgId":             "the tenant's Zidentity organization id",
		"iamIdpIdToIamIdMapping.mappings.iamIdpId": "a Zidentity IdP id; no Zidentity IdP resource in the catalog",
	},
}
