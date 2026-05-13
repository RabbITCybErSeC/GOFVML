// Package gofvml provides public library APIs for GOFVML.
//
// These APIs are currently internal-domain packages that will be promoted
// to pkg/gofvml after API review.
package gofvml

import "github.com/RabbITCybErSeC/gofvml/internal/version"

// Version returns the current GOFVML version.
func Version() string {
	return version.Version
}
