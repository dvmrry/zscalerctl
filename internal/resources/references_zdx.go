package resources

// Reference posture for the ZDX report resources (catalog_zdx.go); see
// fieldReferences in references.go. ZDX ids are ZDX's own report
// dimensions; matches to ZIA objects are not verified.
var zdxUnmappedReferences = func() map[string]map[string]string {
	alert := map[string]string{
		"application":      "a ZDX application; zdx/apps reports it",
		"departments":      "a ZDX report department; zdx/departments reports it",
		"locations":        "a ZDX report location; zdx/locations reports it",
		"locations.groups": "a ZDX location group, not a catalog object",
		"geolocations":     "a geographic area, not a catalog object",
	}
	return map[string]map[string]string{
		"zdx/apps":          {"most_impacted_region": "a geographic area, not a catalog object"},
		"zdx/alerts":        alert,
		"zdx/alert-history": alert,
	}
}()
