package client

import (
	"fmt"

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

	var bestPackage *bookingv1.Package
	var bestScore int64 = -1

	for _, pkg := range ps.packages {
		if pkg.Status != bookingv1.PackageStatus_PACKAGE_STATUS_ACTIVE {
			continue
		}

		if pkg.ResourceConfiguration == nil {
			continue
		}

		pkgRAM, err := resource.ParseQuantity(pkg.ResourceConfiguration.Ram)
		if err != nil {
			continue
		}

		pkgCPU, err := resource.ParseQuantity(pkg.ResourceConfiguration.Cpu)
		if err != nil {
			continue
		}

		pkgDisk, err := resource.ParseQuantity(pkg.ResourceConfiguration.Disk)
		if err != nil {
			continue
		}

		// Check if package meets requirements
		if minRAM != nil && pkgRAM.Cmp(*minRAM) < 0 {
			continue
		}
		if minCPU != nil && pkgCPU.Cmp(*minCPU) < 0 {
			continue
		}
		if minDisk != nil && pkgDisk.Cmp(*minDisk) < 0 {
			continue
		}

		// Calculate a score based on total resources (lower is better for smallest match)
		score := pkgRAM.Value() + pkgDisk.Value() + (pkgCPU.MilliValue() * 1024 * 1024)

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
