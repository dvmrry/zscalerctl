package resources

// ZPA policy-rule reference targets, the same for every policy type; see
// fieldReferences in references.go. The group fields are typed as the
// server, connector and service-edge group SDK structs.
var zpaPolicyRuleFieldReferences = perZPAPolicyRuleResource(map[string]referenceTargets{
	"appConnectorGroups":                         zpaRef("app-connector-groups"),
	"appServerGroups":                            zpaRef("server-groups"),
	"serviceEdgeGroups":                          zpaRef("service-edge-groups"),
	"microtenantId":                              zpaRef("microtenants"),
	"conditions.operands.idpId":                  zpaRef("idps"),
	"conditions.microtenantId":                   zpaRef("microtenants"),
	"conditions.operands.microtenantId":          zpaRef("microtenants"),
	"privilegedCapabilities.microtenantId":       zpaRef("microtenants"),
	"privilegedPortalCapabilities.microtenantId": zpaRef("microtenants"),
})

var zpaPolicyRuleUnmappedReferences = perZPAPolicyRuleResource(map[string]string{ // #nosec G101 -- API field names and reasons; gosec flags the whole block
	"actionId":                                  "the action's id; what it points at is not verified",
	"policySetId":                               "no policy-set resource in the catalog",
	"groupId":                                   "no policy-group resource in the catalog",
	"zpnCbiProfileId":                           "match to zpa/cbi-zpa-profiles is not verified",
	"zpnIsolationProfileId":                     "match to zpa/isolation-profiles is not verified",
	"zpnInspectionProfileId":                    "no inspection profile resource in the catalog",
	"browserPostureProfileId":                   "no browser posture resource in the catalog",
	"credential":                                "no PRA credential resource in the catalog",
	"conditions":                                "the rule's own condition entries, not a pointer",
	"conditions.operands":                       "what an operand names depends on its objectType (application, segment group, SAML or SCIM attribute, posture, ...); not modeled",
	"privilegedCapabilities":                    "the rule's own capability set, not a pointer",
	"extranetDTO.locationDTO":                   "a ZIA location used by the extranet; match to zia/locations is not verified",
	"extranetDTO.locationGroupDTO":              "a ZIA location group used by the extranet; match to a ZIA resource is not verified",
	"extranetDTO.locationGroupDTO.ziaLocations": "a ZIA location used by the extranet; match to zia/locations is not verified",
	"extranetDTO.zpnErId":                       "no extranet resource in the catalog",
})

func perZPAPolicyRuleResource[V any](fields map[string]V) map[string]map[string]V {
	out := make(map[string]map[string]V, len(zpaPolicyRuleResources))
	for _, name := range zpaPolicyRuleResources {
		out[string(ProductZPA)+"/"+name] = fields
	}
	return out
}
