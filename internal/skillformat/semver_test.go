package skillformat

import (
	"strings"
	"testing"
)

// TestCompareSemVer_Equal tests equal versions
func TestCompareSemVer_Equal(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
	}{
		{"Same version 1.0.0", "1.0.0", "1.0.0"},
		{"Same version 0.0.0", "0.0.0", "0.0.0"},
		{"Same version 10.20.30", "10.20.30", "10.20.30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareSemVer(tt.a, tt.b)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if result != 0 {
				t.Errorf("Expected 0 (equal), got %d", result)
			}
		})
	}
}

// TestCompareSemVer_Greater tests a > b cases
func TestCompareSemVer_Greater(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
	}{
		{"Major version greater", "2.0.0", "1.0.0"},
		{"Minor version greater", "1.2.0", "1.1.0"},
		{"Patch version greater", "1.0.2", "1.0.1"},
		{"Major dominates", "2.0.0", "1.9.9"},
		{"Minor dominates", "1.2.0", "1.1.9"},
		{"Large numbers", "100.200.300", "100.200.299"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareSemVer(tt.a, tt.b)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if result != 1 {
				t.Errorf("Expected 1 (greater), got %d", result)
			}
		})
	}
}

// TestCompareSemVer_Less tests a < b cases
func TestCompareSemVer_Less(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
	}{
		{"Major version less", "1.0.0", "2.0.0"},
		{"Minor version less", "1.1.0", "1.2.0"},
		{"Patch version less", "1.0.1", "1.0.2"},
		{"Major dominates", "1.9.9", "2.0.0"},
		{"Minor dominates", "1.1.9", "1.2.0"},
		{"Zero vs one", "0.0.0", "0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareSemVer(tt.a, tt.b)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if result != -1 {
				t.Errorf("Expected -1 (less), got %d", result)
			}
		})
	}
}

// TestCompareSemVer_InvalidFirst tests invalid first version
func TestCompareSemVer_InvalidFirst(t *testing.T) {
	tests := []struct {
		name    string
		a       string
		b       string
		errText string
	}{
		{"Invalid format v1.0.0", "v1.0.0", "1.0.0", "invalid version"},
		{"Too few parts", "1.0", "1.0.0", "expected 3 parts"},
		{"Too many parts", "1.0.0.0", "1.0.0", "expected 3 parts"},
		{"Non-numeric major", "x.0.0", "1.0.0", "invalid number"},
		{"Non-numeric minor", "1.x.0", "1.0.0", "invalid number"},
		{"Non-numeric patch", "1.0.x", "1.0.0", "invalid number"},
		{"Empty string", "", "1.0.0", "expected 3 parts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CompareSemVer(tt.a, tt.b)
			if err == nil {
				t.Fatal("Expected error for invalid first version")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("Expected error containing '%s', got: %v", tt.errText, err)
			}
		})
	}
}

// TestCompareSemVer_InvalidSecond tests invalid second version
func TestCompareSemVer_InvalidSecond(t *testing.T) {
	tests := []struct {
		name    string
		a       string
		b       string
		errText string
	}{
		{"Invalid format v1.0.0", "1.0.0", "v1.0.0", "invalid version"},
		{"Too few parts", "1.0.0", "1.0", "expected 3 parts"},
		{"Too many parts", "1.0.0", "1.0.0.0", "expected 3 parts"},
		{"Non-numeric major", "1.0.0", "x.0.0", "invalid number"},
		{"Non-numeric minor", "1.0.0", "1.x.0", "invalid number"},
		{"Non-numeric patch", "1.0.0", "1.0.x", "invalid number"},
		{"Empty string", "1.0.0", "", "expected 3 parts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CompareSemVer(tt.a, tt.b)
			if err == nil {
				t.Fatal("Expected error for invalid second version")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("Expected error containing '%s', got: %v", tt.errText, err)
			}
		})
	}
}

// TestCompareSemVer_EdgeCases tests edge cases
func TestCompareSemVer_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"Zero versions", "0.0.0", "0.0.0", 0},
		{"Zero vs non-zero major", "0.0.0", "1.0.0", -1},
		{"Zero vs non-zero minor", "1.0.0", "1.1.0", -1},
		{"Zero vs non-zero patch", "1.1.0", "1.1.1", -1},
		{"Large version numbers", "999.999.999", "999.999.999", 0},
		{"Large vs small", "1000.0.0", "999.999.999", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareSemVer(tt.a, tt.b)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestParseSemVer_Valid tests valid semver parsing via CompareSemVer
func TestParseSemVer_Valid(t *testing.T) {
	tests := []string{
		"0.0.0",
		"1.0.0",
		"1.2.3",
		"10.20.30",
		"999.999.999",
	}

	for _, version := range tests {
		t.Run(version, func(t *testing.T) {
			// parseSemVer is private, test via CompareSemVer
			_, err := CompareSemVer(version, version)
			if err != nil {
				t.Errorf("Expected valid version '%s', got error: %v", version, err)
			}
		})
	}
}

// TestParseSemVer_Invalid tests invalid semver parsing via CompareSemVer
func TestParseSemVer_Invalid(t *testing.T) {
	tests := []struct {
		version string
		errText string
	}{
		{"v1.0.0", "invalid number"},
		{"1.0", "expected 3 parts"},
		{"1.0.0.0", "expected 3 parts"},
		{"1.x.0", "invalid number"},
		{"1.0.x", "invalid number"},
		{"x.0.0", "invalid number"},
		{"", "expected 3 parts"},
		{"1", "expected 3 parts"},
		{"1.2.3.4.5", "expected 3 parts"},
		{"1.2.3-beta", "invalid number"},
		{"1.2.3+build", "invalid number"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			_, err := CompareSemVer(tt.version, "1.0.0")
			if err == nil {
				t.Errorf("Expected error for invalid version '%s'", tt.version)
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("Expected error containing '%s', got: %v", tt.errText, err)
			}
		})
	}
}
