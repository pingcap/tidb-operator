package portforward

import (
	"context"
	"fmt"
	"strconv"
)

// ForwardOnePort provide a helper to forward only one port
// and return local endpoint
func ForwardOnePort(ctx context.Context, fw PortForwarder, ns, resourceName string, port int) (string, error) {
	ports, err := fw.Forward(ctx, ns, resourceName, []int{port})
	if err != nil {
		return "", err
	}
	if len(ports) != 1 {
		return "", fmt.Errorf("portforward expect only one port, but now %v", len(ports))
	}
	localPort := int(ports[0].Local)
	return "localhost:" + strconv.Itoa(localPort), nil
}
