package version

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func New(major, minor, patch int) *Version {
	return &Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Version) Equal(other Version) bool {
	return v.Major == other.Major && v.Minor == other.Minor && v.Patch == other.Patch
}

func (v Version) GreaterThan(other Version) bool {
	if v.Major > other.Major {
		return true
	} else if v.Major < other.Major {
		return false
	}

	if v.Minor > other.Minor {
		return true
	} else if v.Minor < other.Minor {
		return false
	}

	return v.Patch > other.Patch
}

func (v Version) LessThan(other Version) bool {
	return !v.Equal(other) && !v.GreaterThan(other)
}

func ParseVersion(version string) (*Version, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid version %s", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version %s: %w", parts[0], err)
	}
	if major < 0 {
		return nil, fmt.Errorf("invalid major version %s: cannot be negative", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version %s: %w", parts[1], err)
	}
	if minor < 0 {
		return nil, fmt.Errorf("invalid minor version %s: cannot be negative", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version %s: %w", parts[2], err)
	}
	if patch < 0 {
		return nil, fmt.Errorf("invalid patch version %s: cannot be negative", parts[2])
	}

	return New(major, minor, patch), nil
}

func IsVersionGreaterOrEqual(a, b string) (bool, error) {
	versionA, err := ParseVersion(a)
	if err != nil {
		return false, fmt.Errorf("invalid version %s: %w", a, err)
	}

	versionB, err := ParseVersion(b)
	if err != nil {
		return false, fmt.Errorf("invalid version %s: %w", b, err)
	}
	return versionA.Equal(*versionB) || versionA.GreaterThan(*versionB), nil
}

func IsVersionLessThan(a, b string) (bool, error) {
	greaterOrEqual, err := IsVersionGreaterOrEqual(a, b)
	if err != nil {
		return false, err
	}
	return !greaterOrEqual, nil
}
