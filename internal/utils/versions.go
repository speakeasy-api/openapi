package utils

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseVersion(version string) (int, int, int, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid version %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version %s: %w", parts[0], err)
	}
	if major < 0 {
		return 0, 0, 0, fmt.Errorf("invalid major version %s: cannot be negative", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version %s: %w", parts[1], err)
	}
	if minor < 0 {
		return 0, 0, 0, fmt.Errorf("invalid minor version %s: cannot be negative", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch version %s: %w", parts[2], err)
	}
	if patch < 0 {
		return 0, 0, 0, fmt.Errorf("invalid patch version %s: cannot be negative", parts[2])
	}

	return major, minor, patch, nil
}
