package resources

// ZDX report resources: time-windowed telemetry read with the report
// operation only. App, rule, department and location names and aggregate
// counts render in share; cities and geolocation names stay standard only.
func catalogZDX() ResourceCatalog {
	metricSeries := func(name string) FieldSpec {
		return FieldSpec{
			Name:           name,
			Classification: ClassOperational,
			AllowedModes:   allModes(),
			Fields: []FieldSpec{
				operationalField("metric", allModes()),
				operationalField("unit", allModes()),
				{
					Name:           "datapoints",
					Classification: ClassOperational,
					AllowedModes:   allModes(),
					Fields: []FieldSpec{
						operationalField("timestamp", allModes()),
						namedOperationalField("value", allModes(), "a metric data point's numeric value"),
					},
				},
			},
		}
	}
	idName := func(name string, extra ...FieldSpec) FieldSpec {
		return FieldSpec{
			Name:           name,
			Classification: ClassTenantConfig,
			AllowedModes:   standardShareModes(),
			Fields: append([]FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
			}, extra...),
		}
	}
	alert := func(name string) ResourceSpec {
		return ResourceSpec{
			Product:    ProductZDX,
			Name:       name,
			Shape:      ShapeReport,
			Operations: ReportOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("rule_name", standardShareModes()),
				operationalField("severity", allModes()),
				operationalField("alert_type", allModes()),
				operationalField("alert_status", allModes()),
				operationalField("num_geolocations", allModes()),
				operationalField("num_devices", allModes()),
				operationalField("started_on", allModes()),
				operationalField("ended_on", allModes()),
				idName("application"),
				idName("departments", operationalField("num_devices", allModes())),
				idName("locations", operationalField("num_devices", allModes()),
					idName("groups")),
				{
					Name:           "geolocations",
					Classification: ClassTenantConfig,
					AllowedModes:   standardShareModes(),
					Fields: []FieldSpec{
						sensitiveIdentifierField("id"),
						sensitiveIdentifierField("name"),
						operationalField("num_devices", allModes()),
					},
				},
			},
		}
	}
	dimension := func(name string) ResourceSpec {
		return ResourceSpec{
			Product:    ProductZDX,
			Name:       name,
			Shape:      ShapeReport,
			Operations: ReportOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
			},
		}
	}
	return ResourceCatalog{
		{
			Product:    ProductZDX,
			Name:       "apps",
			Shape:      ShapeReport,
			Operations: ReportOperations(),
			Fields: []FieldSpec{
				operationalField("id", allModes()),
				tenantConfigField("name", standardShareModes()),
				operationalField("score", allModes()),
				operationalField("total_users", allModes()),
				{
					Name:           "most_impacted_region",
					Classification: ClassTenantConfig,
					AllowedModes:   standardShareModes(),
					Fields: []FieldSpec{
						sensitiveIdentifierField("id"),
						sensitiveIdentifierField("city"),
						tenantConfigField("region", standardShareModes()),
						tenantConfigField("country", standardShareModes()),
						operationalField("geo_type", allModes()),
					},
				},
				{
					Name:           "stats",
					Classification: ClassOperational,
					AllowedModes:   allModes(),
					Fields: []FieldSpec{
						operationalField("active_users", allModes()),
						operationalField("active_devices", allModes()),
						operationalField("num_poor", allModes()),
						operationalField("num_okay", allModes()),
						operationalField("num_good", allModes()),
					},
				},
				// Present only on a one-app report (apps report <id>).
				metricSeries("score_trend"),
				metricSeries("metrics"),
			},
		},
		alert("alerts"),
		alert("alert-history"),
		dimension("departments"),
		dimension("locations"),
	}
}
