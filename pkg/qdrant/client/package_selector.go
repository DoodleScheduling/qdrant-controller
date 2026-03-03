package client

import (
	"fmt"
	"strconv"
	"strings"

	bookingv1 "github.com/qdrant/qdrant-cloud-public-api/gen/go/qdrant/cloud/booking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// PackageSelector helps select the appropriate package based on resource requirements
type PackageSelector struct {
	packages []*bookingv1.Package
}

// NewPackageSelector creates a new package selector from a list of packages
func NewPackageSelector(packages []*bookingv1.Package) *PackageSelector {
	return &PackageSelector{packages: packages}
}

// SelectPackage finds the smallest package that meets the given requirements
func (ps *PackageSelector) SelectPackage(minRAM, minCPU, minDisk *resource.Quantity) (*bookingv1.Package, error) {
	if len(ps.packages) == 0 {
		return nil, fmt.Errorf("no packages available")
	}

	var minRAMBytes, minCPUMillis, minDiskBytes int64
	if minRAM != nil {
		minRAMBytes = minRAM.Value()
	}
	if minCPU != nil {
		minCPUMillis = minCPU.MilliValue()
	}
	if minDisk != nil {
		minDiskBytes = minDisk.Value()
	}

	var bestPackage *bookingv1.Package
	var bestScore int64 = -1

	for _, pkg := range ps.packages {
		// Skip inactive packages
		if pkg.Status != bookingv1.PackageStatus_PACKAGE_STATUS_ACTIVE {
			continue
		}

		if pkg.ResourceConfiguration == nil {
			continue
		}

		// Parse package resources
		pkgRAM, err := parseResourceString(pkg.ResourceConfiguration.Ram)
		if err != nil {
			continue
		}

		pkgCPU, err := parseCPUString(pkg.ResourceConfiguration.Cpu)
		if err != nil {
			continue
		}

		pkgDisk, err := parseResourceString(pkg.ResourceConfiguration.Disk)
		if err != nil {
			continue
		}

		// Check if package meets requirements
		if minRAMBytes > 0 && pkgRAM < minRAMBytes {
			continue
		}
		if minCPUMillis > 0 && pkgCPU < minCPUMillis {
			continue
		}
		if minDiskBytes > 0 && pkgDisk < minDiskBytes {
			continue
		}

		// Calculate a score based on total resources (lower is better for smallest match)
		score := pkgRAM + pkgDisk + (pkgCPU * 1024 * 1024) // Weight CPU appropriately

		if bestScore == -1 || score < bestScore {
			bestScore = score
			bestPackage = pkg
		}
	}

	if bestPackage == nil {
		return nil, fmt.Errorf("no package found matching requirements (RAM: %v, CPU: %v, Disk: %v)", minRAM, minCPU, minDisk)
	}

	return bestPackage, nil
}

// parseResourceString parses resource strings like "8GiB", "16GB", "2048MiB" into bytes
func parseResourceString(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}

	s = strings.TrimSpace(s)

	// Extract number and unit
	var numStr string
	var unit string

	for i, c := range s {
		if c >= '0' && c <= '9' {
			numStr += string(c)
		} else {
			unit = s[i:]
			break
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("no numeric value found in %s", s)
	}

	value, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, err
	}

	// Convert to bytes based on unit
	unit = strings.ToUpper(strings.TrimSpace(unit))
	switch unit {
	case "B":
		return value, nil
	case "KB":
		return value * 1000, nil
	case "KIB", "K":
		return value * 1024, nil
	case "MB":
		return value * 1000 * 1000, nil
	case "MIB", "M":
		return value * 1024 * 1024, nil
	case "GB":
		return value * 1000 * 1000 * 1000, nil
	case "GIB", "G":
		return value * 1024 * 1024 * 1024, nil
	case "TB":
		return value * 1000 * 1000 * 1000 * 1000, nil
	case "TIB", "T":
		return value * 1024 * 1024 * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}
}

// parseCPUString parses CPU strings like "2000m", "4" into millicores
func parseCPUString(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}

	s = strings.TrimSpace(s)

	// If it ends with 'm', it's already in millicores
	if strings.HasSuffix(s, "m") {
		numStr := strings.TrimSuffix(s, "m")
		return strconv.ParseInt(numStr, 10, 64)
	}

	// Otherwise, it's in cores, convert to millicores
	cores, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}

	return int64(cores * 1000), nil
}

// FormatResourceString formats bytes into a human-readable string
func FormatResourceString(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KiB", "MiB", "GiB", "TiB"}
	return fmt.Sprintf("%.1f%s", float64(bytes)/float64(div), units[exp])
}

// FormatCPUString formats millicores into a human-readable string
func FormatCPUString(millicores int64) string {
	if millicores%1000 == 0 {
		return fmt.Sprintf("%d", millicores/1000)
	}
	return fmt.Sprintf("%dm", millicores)
}
