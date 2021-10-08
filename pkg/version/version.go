package version

import (
	"fmt"
	"strings"
)

// Build information obtained with the help of -ldflags
var (
	GitVersion  string
	GitCommit   string
	PackageName = "aws-cloud-map-mcs-controller-for-k8s"
)

// GetVersion figures out the version information
// based on variables set by -ldflags.
func GetVersion() string {
	// only set the appVersion if -ldflags was used
	if GitCommit != "" {
		return fmt.Sprintf("%s (%s)", strings.TrimPrefix(GitVersion, "v"), GitCommit)
	}

	return ""
}
