// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

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
