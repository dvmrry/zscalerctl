package zscaler

import (
	"reflect"

	zdxadmin "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zdx/services/administration"
	zdxalerts "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zdx/services/alerts"
	zdxapps "github.com/zscaler/zscaler-sdk-go/v3/zscaler/zdx/services/reports/applications"

	"zscalerctl/internal/resources"
)

// reviewedSDKShapesZDX reviews the ZDX report resources (catalog_zdx.go).
// zdx/apps adds score_trend and metrics on a one-app report, read from
// GetAppScores and GetAppMetrics beside the Apps record.
func reviewedSDKShapesZDX() []sdkShapeReview {
	alertFields := []string{"id", "rule_name", "severity", "alert_type", "alert_status", "num_geolocations",
		"num_devices", "started_on", "ended_on", "application", "departments", "locations", "geolocations"}
	return []sdkShapeReview{
		{
			name: "zdxapps.Apps", resource: resources.ProductZDX, resourceName: "apps",
			typ:           reflect.TypeFor[zdxapps.Apps](),
			catalogFields: []string{"id", "name", "score", "total_users", "most_impacted_region", "stats"},
		},
		{
			name: "zdxalerts.Alert/alerts", resource: resources.ProductZDX, resourceName: "alerts",
			typ: reflect.TypeFor[zdxalerts.Alert](), catalogFields: alertFields,
		},
		{
			name: "zdxalerts.Alert/alert-history", resource: resources.ProductZDX, resourceName: "alert-history",
			typ: reflect.TypeFor[zdxalerts.Alert](), catalogFields: alertFields,
		},
		{
			name: "zdxadmin.Department", resource: resources.ProductZDX, resourceName: "departments",
			typ: reflect.TypeFor[zdxadmin.Department](), catalogFields: []string{"id", "name"},
		},
		{
			name: "zdxadmin.Location", resource: resources.ProductZDX, resourceName: "locations",
			typ: reflect.TypeFor[zdxadmin.Location](), catalogFields: []string{"id", "name"},
		},
	}
}
