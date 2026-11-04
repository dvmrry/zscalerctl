package resources

// SDKLedgerEntry says what accounts for one zscaler-sdk-go service package
// that may read (it has read functions, or only unknown-verb ones) but that
// zscalerctl does not import. An imported package counts as covered by the
// handlers that import it; that is a per-package convention, and a read
// function such a package exposes but no handler binds is a coverage gap.
// Every other such package must appear here, and scripts/verify-sdk-ledger.sh
// fails when an SDK version adds one the ledger does not name.
//
// Exactly one disposition is set: Resources (catalog resources that read the
// package's records through zscalerctl's own pagers, possibly at a sibling
// route), Gaps (declared coverage gaps of the same product), Command (a
// non-catalog command), Excluded (no tenant read: an SDK helper or program),
// OutOfScope (scouted and deliberately not read; the note says why), or
// Pending (not yet scouted; the note says which track takes it).
type SDKLedgerEntry struct {
	// Package is the path under zscaler/, e.g. "zpa/services/idpcontroller".
	Package    string
	Resources  []ResourceRef
	Gaps       []string
	Command    string
	Excluded   string
	OutOfScope string
	Pending    string
}

const (
	outOfScopeZWA = "ZWA DLP incidents (with evidence and matched content) and customer audit logs; incident telemetry behind a separate ZWA service client, not configuration."
)

func ledgerZPA(name string) ResourceRef { return ResourceRef{Product: ProductZPA, Resource: name} }

func ledgerZIA(name string) ResourceRef { return ResourceRef{Product: ProductZIA, Resource: name} }

func ledgerZTW(name string) ResourceRef { return ResourceRef{Product: ProductZTW, Resource: name} }

// sdkLedger is sorted by Package.
var sdkLedger = []SDKLedgerEntry{
	// ZCC
	{Package: "zcc/services/admin_users", Gaps: []string{"admin-users"}},
	{Package: "zcc/services/download_devices", Gaps: []string{"device-export"}},
	{Package: "zcc/services/entitlements", Gaps: []string{"zdx-group-entitlements", "zpa-group-entitlements"}},
	{Package: "zcc/services/notification_template", Gaps: []string{"notification-templates"}},
	{Package: "zcc/services/secrets/getotp", Gaps: []string{"passwords"}},
	{Package: "zcc/services/secrets/getpasswords", Gaps: []string{"passwords"}},
	{Package: "zcc/services/trusted_network_v2", Gaps: []string{"trusted-networks-v2"}},
	{Package: "zcc/services/web_policy", Gaps: []string{"web-policies"}},
	{Package: "zcc/services/zia_posture", Gaps: []string{"zia-posture-profiles"}},

	// ZDX
	{Package: "zdx/services/inventory", Gaps: []string{"software-inventory"}},
	{Package: "zdx/services/reports/devices", Gaps: []string{"devices"}},
	{Package: "zdx/services/reports/users", Gaps: []string{"users"}},
	{Package: "zdx/services/troubleshooting/analysis", Gaps: []string{"deep-traces"}},
	{Package: "zdx/services/troubleshooting/deeptrace", Gaps: []string{"deep-traces"}},

	// ZIA
	{Package: "zia/services/adminauditlogs", Command: "zia admin-audit-logs status"},
	{Package: "zia/services/apptotal", Gaps: []string{"app-total"}},
	{Package: "zia/services/endpoint_dlp/endpoint_application_groups", Gaps: []string{"endpoint-application-groups"}},
	{Package: "zia/services/endpoint_dlp/endpoint_applications", Gaps: []string{"endpoint-applications"}},
	{Package: "zia/services/endpoint_dlp/endpoint_custom_apps", Gaps: []string{"endpoint-custom-apps"}},
	{Package: "zia/services/endpoint_dlp/endpoint_resource_channel", Gaps: []string{"endpoint-resources"}},
	{Package: "zia/services/endpoint_dlp/endpoint_resource_group", Gaps: []string{"endpoint-resource-groups"}},
	{Package: "zia/services/eventlogentryreport", Gaps: []string{"reports"}},
	{Package: "zia/services/iotreport", Gaps: []string{"reports"}},
	{Package: "zia/services/location/locationlite", Resources: []ResourceRef{ledgerZIA("locations-lite")}},
	{Package: "zia/services/policy_export", Gaps: []string{"policy-export"}},
	{Package: "zia/services/sandbox/sandbox_report", Gaps: []string{"reports"}},
	{Package: "zia/services/scim_api", Gaps: []string{"scim"}},
	{Package: "zia/services/security_ueba_alerts/alert_configurations", Gaps: []string{"alert-configuration-rules"}},
	{Package: "zia/services/security_ueba_alerts/ueba_rules", Gaps: []string{"ueba-rules"}},
	{Package: "zia/services/security_ueba_alerts/webhooks", Gaps: []string{"ueba-webhooks"}},
	{Package: "zia/services/shadowitreport", Gaps: []string{"reports"}},
	{Package: "zia/services/traffic_capture", Gaps: []string{"traffic-capture-rules"}},
	{Package: "zia/services/trafficforwarding/extranet", Gaps: []string{"extranets"}},
	{Package: "zia/services/trafficforwarding/greinternalipranges", Gaps: []string{"gre-internal-ip-ranges"}},
	{Package: "zia/services/trafficforwarding/gretunnelinfo", Gaps: []string{"gre-tunnel-info"}},
	{Package: "zia/services/trafficforwarding/region/datacenter", Gaps: []string{"regions"}},
	{Package: "zia/services/trafficforwarding/region/geo_coordinates", Gaps: []string{"regions"}},
	{Package: "zia/services/trafficforwarding/region/ip_address", Gaps: []string{"regions"}},
	{Package: "zia/services/trafficforwarding/region/search", Gaps: []string{"regions"}},
	{Package: "zia/services/trafficforwarding/virtualipaddress", Gaps: []string{"virtual-ip-addresses"}},
	{Package: "zia/services/trafficforwarding/vpncredentials", Gaps: []string{"vpn-credentials"}},
	{Package: "zia/services/votiro_cdr", Gaps: []string{"votiro-cdr-policies", "votiro-cdr-token"}},

	// Zidentity (the SDK's zscaler/ziam tree)
	{Package: "ziam/services/user_entitlement", Gaps: []string{"entitlements"}},

	// ZPA
	{Package: "zpa/services/admin_sso_controller", Resources: []ResourceRef{ledgerZPA("admin-sso-options")}},
	{Package: "zpa/services/api_keys", Gaps: []string{"api-keys"}},
	{Package: "zpa/services/appconnectorschedule", Gaps: []string{"app-connector-schedule"}},
	// getAppsByType is a lighter view of the records the /application pagers read.
	{Package: "zpa/services/applicationsegmentbytype", Resources: []ResourceRef{ledgerZPA("browser-access"), ledgerZPA("inspection-app-segments"), ledgerZPA("pra-app-segments")}},
	{Package: "zpa/services/b2b_policy_controller", Gaps: []string{"policy-groups"}},
	{Package: "zpa/services/bacertificate", Gaps: []string{"ba-certificates"}},
	{Package: "zpa/services/branch_connector_group", Resources: []ResourceRef{ledgerZPA("branch-connector-groups")}},
	{Package: "zpa/services/browser_access_groups", Gaps: []string{"browser-access-groups"}},
	{Package: "zpa/services/browser_protection", Gaps: []string{"browser-protection-profiles"}},
	{Package: "zpa/services/cloudbrowserisolation/cbibannercontroller", Gaps: []string{"cbi-banners"}},
	{Package: "zpa/services/cloudbrowserisolation/cbicertificatecontroller", Gaps: []string{"cbi-certificates"}},
	{Package: "zpa/services/cloudbrowserisolation/cbiprofilecontroller", Gaps: []string{"cbi-profiles"}},
	{Package: "zpa/services/cloudbrowserisolation/cbiregions", Gaps: []string{"cbi-regions"}},
	{Package: "zpa/services/custom_config_controller", Resources: []ResourceRef{ledgerZPA("session-termination-settings")}}, // ZIA cloud config is the gap zia-cloud-config
	{Package: "zpa/services/customer_domain_controller", Gaps: []string{"customer-domains"}},
	{Package: "zpa/services/customer_dr_tool", Resources: []ResourceRef{ledgerZPA("dr-tool-versions")}},
	{Package: "zpa/services/emergencyaccess", Gaps: []string{"emergency-access-users"}},
	{Package: "zpa/services/enrollmentcert", Gaps: []string{"enrollment-certificates"}},
	{Package: "zpa/services/extranet_resource", Resources: []ResourceRef{ledgerZPA("extranet-partners")}},
	{Package: "zpa/services/federated_application", Gaps: []string{"federated-applications"}},
	{Package: "zpa/services/inspectioncontrol/inspection_custom_controls", Gaps: []string{"inspection-custom-controls"}},
	{Package: "zpa/services/inspectioncontrol/inspection_predefined_controls", Gaps: []string{"inspection-predefined-controls"}},
	{Package: "zpa/services/inspectioncontrol/inspection_profile", Gaps: []string{"inspection-profiles"}},
	{Package: "zpa/services/location_controller", Resources: []ResourceRef{ledgerZPA("location-summaries")}},
	{Package: "zpa/services/lssconfigcontroller", Gaps: []string{"lss-configs"}},
	{Package: "zpa/services/managed_browser", Resources: []ResourceRef{ledgerZPA("managed-browser-profiles")}},
	{Package: "zpa/services/np_client", Gaps: []string{"vpn-connected-users"}},
	{Package: "zpa/services/policy_group", Gaps: []string{"policy-groups"}},
	{Package: "zpa/services/policy_group_rule", Gaps: []string{"policy-groups"}},
	{Package: "zpa/services/policy_group_set", Gaps: []string{"policy-groups"}},
	// The per-type rule lists; GetPolicyCount and GetRiskScoreValues are not read.
	{Package: "zpa/services/policysetcontrollerv2", Resources: []ResourceRef{
		ledgerZPA("access-policy-rules"), ledgerZPA("capabilities-policy-rules"), ledgerZPA("client-forwarding-policy-rules"),
		ledgerZPA("clientless-session-protection-policy-rules"), ledgerZPA("credential-policy-rules"), ledgerZPA("inspection-policy-rules"),
		ledgerZPA("isolation-policy-rules"), ledgerZPA("redirection-policy-rules"), ledgerZPA("siem-policy-rules"), ledgerZPA("timeout-policy-rules"),
	}},
	{Package: "zpa/services/private_cloud", Gaps: []string{"private-clouds"}},
	{Package: "zpa/services/private_cloud_controller", Gaps: []string{"private-cloud-controllers"}},
	{Package: "zpa/services/private_cloud_group", Gaps: []string{"private-cloud-groups"}},
	{Package: "zpa/services/privilegedremoteaccess/praapproval", Gaps: []string{"pra-approvals"}},
	{Package: "zpa/services/privilegedremoteaccess/praconsole", Gaps: []string{"pra-consoles"}},
	{Package: "zpa/services/privilegedremoteaccess/pracredential", Gaps: []string{"pra-credentials"}},
	{Package: "zpa/services/privilegedremoteaccess/pracredentialpool", Gaps: []string{"pra-credential-pools"}},
	{Package: "zpa/services/privilegedremoteaccess/praportal", Gaps: []string{"pra-portals"}},
	{Package: "zpa/services/provisioningkey", Gaps: []string{"provisioning-keys"}},
	{Package: "zpa/services/scim_api", Gaps: []string{"scim-api"}},
	{Package: "zpa/services/serviceedgeschedule", Gaps: []string{"service-edge-schedule"}},
	{Package: "zpa/services/step_up_auth", Resources: []ResourceRef{ledgerZPA("step-up-auth-levels")}},
	{Package: "zpa/services/tag_controller/tag_group", Gaps: []string{"tag-groups"}},
	{Package: "zpa/services/tag_controller/tag_key", Gaps: []string{"tag-keys"}},
	{Package: "zpa/services/tag_controller/tag_namespace", Gaps: []string{"tag-namespaces"}},
	{Package: "zpa/services/workload_tag_group", Resources: []ResourceRef{ledgerZPA("workload-tag-groups")}},

	// ZTW
	{Package: "ztw/services/activation_cli", Excluded: "An activation program (package main); activation mutates."},
	{Package: "ztw/services/forwarding_gateways/dns_forwarding_gateway", Resources: []ResourceRef{ledgerZTW("dns-gateways")}},
	{Package: "ztw/services/locationmanagement/locationlite", Gaps: []string{"locations-lite"}},
	{Package: "ztw/services/partner_integrations", Gaps: []string{"cloud-formation-template", "supported-regions", "workload-discovery-settings"}},
	{Package: "ztw/services/provisioning/api_keys", Gaps: []string{"api-keys"}},
	{Package: "ztw/services/provisioning/provisioning_url", Gaps: []string{"provisioning-urls"}},

	// ZWA
	{Package: "zwa/services/common", OutOfScope: "The generic pagination helpers the ZWA incident and audit reads use."},
	{Package: "zwa/services/customeraudit", OutOfScope: outOfScopeZWA},
	{Package: "zwa/services/dlp_incidents", OutOfScope: outOfScopeZWA},
}

// sdkLedgerProduct maps a ledger package's SDK tree to the product whose gaps
// and resources it may name; ZWA has none.
var sdkLedgerProduct = map[string]Product{
	"zia":  ProductZIA,
	"zpa":  ProductZPA,
	"ztw":  ProductZTW,
	"zcc":  ProductZCC,
	"ziam": ProductZidentity,
	"zdx":  ProductZDX,
}
