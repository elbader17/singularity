package version

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		latest   string
		expected int
	}{
		{"latest is newer", "1.0.0", "1.1.0", 1},
		{"latest is older", "1.1.0", "1.0.0", -1},
		{"same version", "1.0.0", "1.0.0", 0},
		{"patch update available", "1.0.0", "1.0.1", 1},
		{"minor update available", "1.0.0", "1.1.0", 1},
		{"major update available", "1.0.0", "2.0.0", 1},
		{"prerelease - older", "1.0.0-beta", "1.0.0", -1},
		{"prerelease - newer", "1.0.0", "1.0.0-beta", 1},
		{"with v prefix", "v1.0.0", "v1.1.0", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareVersions(tt.current, tt.latest)
			if result != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.current, tt.latest, result, tt.expected)
			}
		})
	}
}

func TestIsUpdateAvailable(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		latest   string
		expected bool
	}{
		{"update available", "1.0.0", "1.1.0", true},
		{"no update available", "1.1.0", "1.0.0", false},
		{"same version", "1.0.0", "1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUpdateAvailable(tt.current, tt.latest)
			if result != tt.expected {
				t.Errorf("IsUpdateAvailable(%q, %q) = %v, want %v", tt.current, tt.latest, result, tt.expected)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		wantMajor int
		wantMinor int
		wantPatch int
		wantErr   bool
	}{
		{"valid version", "1.2.3", 1, 2, 3, false},
		{"with v prefix", "v1.2.3", 1, 2, 3, false},
		{"invalid version", "1.2", 0, 0, 0, true},
		{"invalid characters", "abc", 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, patch, err := ParseVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion(%q) error = %v, wantErr %v", tt.version, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if major != tt.wantMajor || minor != tt.wantMinor || patch != tt.wantPatch {
					t.Errorf("ParseVersion(%q) = (%d, %d, %d), want (%d, %d, %d)",
						tt.version, major, minor, patch, tt.wantMajor, tt.wantMinor, tt.wantPatch)
				}
			}
		})
	}
}
