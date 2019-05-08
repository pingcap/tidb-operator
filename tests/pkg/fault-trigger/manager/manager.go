// Copyright 2019 PingCAP, Inc.
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

package manager

import (
	"fmt"
	"sync"
)

// Manager to manager fault trigger
type Manager struct {
	sync.RWMutex
	vmCache         map[string]string
	kubeProxyImages string
}

// NewManager returns a manager instance
func NewManager(kubeProxyImage string) *Manager {
	return &Manager{
		kubeProxyImages: kubeProxyImage,
		vmCache:         make(map[string]string),
	}
}

func (m *Manager) setVMCache(key, val string) {
	m.Lock()
	m.vmCache[key] = val
	m.Unlock()
}

func (m *Manager) getVMCache(key string) (string, error) {
	m.RLock()
	defer m.RUnlock()

	val, ok := m.vmCache[key]
	if !ok {
		return "", fmt.Errorf("vm %s not in cache", key)
	}

	return val, nil
}
