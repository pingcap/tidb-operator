package compact

import (
	stderrs "errors"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/client-go/discovery"
)

type unsupportedShardedJobK8sVersionError struct {
	message string
}

func (e *unsupportedShardedJobK8sVersionError) Error() string {
	return e.message
}

func isUnsupportedShardedJobK8sVersionError(err error) bool {
	var target *unsupportedShardedJobK8sVersionError
	return stderrs.As(err, &target)
}

func requireShardedJobK8sVersion(dc discovery.DiscoveryInterface) error {
	serverVersion, err := dc.ServerVersion()
	if err != nil {
		return fmt.Errorf("get Kubernetes server version: %w", err)
	}

	major, err := strconv.Atoi(strings.TrimSuffix(serverVersion.Major, "+"))
	if err != nil {
		return fmt.Errorf("parse Kubernetes major version %q: %w", serverVersion.Major, err)
	}
	minor, err := strconv.Atoi(strings.TrimSuffix(serverVersion.Minor, "+"))
	if err != nil {
		return fmt.Errorf("parse Kubernetes minor version %q: %w", serverVersion.Minor, err)
	}

	if major > 1 || (major == 1 && minor >= 29) {
		return nil
	}

	return &unsupportedShardedJobK8sVersionError{
		message: fmt.Sprintf("sharded compact backup requires Kubernetes >= 1.29, current version is %s.%s", serverVersion.Major, serverVersion.Minor),
	}
}
