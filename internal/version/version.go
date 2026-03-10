package version

import (
	"fmt"
	"strings"
)

var Version = "0.0.0"

func CompareVersions(current, latest string) int {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	currentParts := strings.Split(current, "-")
	latestParts := strings.Split(latest, "-")

	currentMain := currentParts[0]
	latestMain := latestParts[0]

	currentMajor, currentMinor, currentPatch, err := ParseVersion(currentMain)
	if err != nil {
		return 0
	}

	latestMajor, latestMinor, latestPatch, err := ParseVersion(latestMain)
	if err != nil {
		return 0
	}

	if latestMajor > currentMajor {
		return 1
	}
	if latestMajor < currentMajor {
		return -1
	}

	if latestMinor > currentMinor {
		return 1
	}
	if latestMinor < currentMinor {
		return -1
	}

	if latestPatch > currentPatch {
		return 1
	}
	if latestPatch < currentPatch {
		return -1
	}

	if len(latestParts) > 1 && len(currentParts) == 1 {
		return 1
	}
	if len(currentParts) > 1 && len(latestParts) == 1 {
		return -1
	}

	return 0
}

func IsUpdateAvailable(current, latest string) bool {
	return CompareVersions(current, latest) > 0
}

func ParseVersion(v string) (major, minor, patch int, err error) {
	v = strings.TrimPrefix(v, "v")

	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid version format: %s", v)
	}

	if _, err := fmt.Sscanf(parts[0], "%d", &major); err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version: %s", parts[0])
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &minor); err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version: %s", parts[1])
	}
	if _, err := fmt.Sscanf(parts[2], "%d", &patch); err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return major, minor, patch, nil
}
