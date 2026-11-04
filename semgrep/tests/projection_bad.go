//go:build semgrepfixtures

package tests

import "zscalerctl/internal/resources"

func projectUnsafely(spec resources.ResourceSpec, record resources.SourceRecord) {
	_, _, _ = resources.ProjectRecord(spec, "", record)
}
