package diff

import (
	"context"
	"os"

	"zscalerctl/internal/dump"
	"zscalerctl/internal/resources"
)

func readResource(ctx context.Context, root *os.Root, mr dump.ManifestResource, spec resources.ResourceSpec) ([]map[string]any, error) {
	records, _, err := readResourceWithLimit(ctx, root, mr, spec, maxResourceBytes, ErrInvalidDump, false)
	return records, err
}
