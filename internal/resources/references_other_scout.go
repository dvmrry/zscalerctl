package resources

// Reference targets for the ZIA/ZCC/Zidentity ledger-scout batch
// (catalog_other_scout.go); see fieldReferences in references.go.
var otherScoutFieldReferences = map[string]map[string]referenceTargets{
	"zia/registered-devices": {"user": ziaRef("users")},
}

var otherScoutUnmappedReferences = map[string]map[string]string{
	"zia/registered-devices": {"hardwareId": "the device's hardware identifier, not a catalog object"},
	"zia/smpc-instances": {
		"nodes":            "the instance's own nodes, not a pointer",
		"nodes.instanceId": "the parent instance's id",
	},
	"zidentity/api-clients": {
		"clientResources":                "the resource servers the client may call; match to zidentity/resource-servers is not verified",
		"clientResources.selectedScopes": "scopes granted on that resource server, not a catalog object",
	},
}
