package fileperm

import "testing"

func TestIsUNCPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path string
		want bool
	}{
		// Local paths — must NOT be flagged as UNC.
		{`C:\Users\me\config.yaml`, false},
		{`\\?\C:\Users\me\config.yaml`, false},
		{`\\.\C:\Users\me\config.yaml`, false},
		{`c:\users\me\.config\zscalerctl\config.yaml`, false},
		{``, false},
		{`relative\path`, false},

		// Volume-GUID form is a LOCAL volume, not UNC.
		{`\\?\Volume{11111111-1111-1111-1111-111111111111}\Users\me\file`, false},

		// UNC / network paths — must be flagged.
		{`\\server\share\config.yaml`, true},
		{`\\?\UNC\server\share\config.yaml`, true},
		{`\\.\UNC\server\share\config.yaml`, true},
		{`\\?\unc\server\share\config.yaml`, true}, // case-insensitive marker
		{`//server/share/config.yaml`, true},       // tolerated forward slashes
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			if got := isUNCPath(tc.path); got != tc.want {
				t.Fatalf("isUNCPath(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestVolumeRootFromFinalPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path string
		want string
	}{
		{`\\?\C:\Users\me\config.yaml`, `C:\`},
		{`\\.\C:\Users\me\config.yaml`, `C:\`},
		{`C:\Users\me\config.yaml`, `C:\`},
		{`c:\users\me\config.yaml`, `c:\`},
		{`Z:`, `Z:\`},

		// Volume-GUID form: letterless / folder-mounted local fixed volume.
		{`\\?\Volume{11111111-1111-1111-1111-111111111111}\Users\me\file`, `\\?\Volume{11111111-1111-1111-1111-111111111111}\`},
		{`\\?\Volume{11111111-1111-1111-1111-111111111111}\`, `\\?\Volume{11111111-1111-1111-1111-111111111111}\`},  // root, no trailing component
		{`\\?\volume{11111111-1111-1111-1111-111111111111}\x`, `\\?\volume{11111111-1111-1111-1111-111111111111}\`}, // case-insensitive marker

		// UNC and non-drive inputs: no local drive root.
		{`\\server\share\config.yaml`, ``},
		{`\\?\UNC\server\share\config.yaml`, ``},
		{`\\.\unc\server\share\file`, ``},
		{``, ``},
		{`relative\path`, ``},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			if got := volumeRootFromFinalPath(tc.path); got != tc.want {
				t.Fatalf("volumeRootFromFinalPath(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}
