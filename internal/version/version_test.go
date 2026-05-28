package version

import "testing"

func TestCurrentUsesInjectedVersion(t *testing.T) {
	oldVersion := version
	t.Cleanup(func() { version = oldVersion })

	version = "v1.2.3"

	if got := Current().Version; got != "1.2.3" {
		t.Errorf("Current().Version = %q, want 1.2.3", got)
	}
}

func TestCurrentDefaultsToDev(t *testing.T) {
	oldVersion := version
	t.Cleanup(func() { version = oldVersion })

	version = "dev"

	if got := Current().Version; got == "" {
		t.Error("Current().Version is empty, want dev or Go module version")
	}
}
