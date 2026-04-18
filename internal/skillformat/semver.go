package skillformat

import (
	"fmt"
	"strconv"
	"strings"
)

// CompareSemVer compares two semver strings.
// Returns 1 if a > b, -1 if a < b, 0 if equal.
// Returns error if either string is not valid semver.
func CompareSemVer(a, b string) (int, error) {
	ap, err := parseSemVer(a)
	if err != nil {
		return 0, fmt.Errorf("invalid version %q: %w", a, err)
	}
	bp, err := parseSemVer(b)
	if err != nil {
		return 0, fmt.Errorf("invalid version %q: %w", b, err)
	}

	if ap[0] != bp[0] {
		if ap[0] > bp[0] {
			return 1, nil
		}
		return -1, nil
	}
	if ap[1] != bp[1] {
		if ap[1] > bp[1] {
			return 1, nil
		}
		return -1, nil
	}
	if ap[2] != bp[2] {
		if ap[2] > bp[2] {
			return 1, nil
		}
		return -1, nil
	}
	return 0, nil
}

func parseSemVer(v string) ([3]int, error) {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return [3]int{}, fmt.Errorf("expected 3 parts, got %d", len(parts))
	}
	var result [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, fmt.Errorf("invalid number %q", p)
		}
		result[i] = n
	}
	return result, nil
}
