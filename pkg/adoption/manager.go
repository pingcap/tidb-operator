// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package adoption

import (
	"sync"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tidb-operator/api/v2/core/v1alpha1"
)

type Manager interface {
	// Register records all standby tidbs and will delete records of normal and deleting tidbs.
	Register(dbs ...*v1alpha1.TiDB)
	// Unregister delete a tidb record directly.
	Unregister(ns, name string)
	// Adopt returns an adoptable tidb for a tidbgroup.
	// Index means the adopt times in one reconciliation.
	// Returned adoptable instance will be locked until UnlockFunc is called
	// or locked db is deleted or activated
	Adopt(dbg *v1alpha1.TiDBGroup, index int) (*v1alpha1.TiDB, UnlockFunc)
}

type UnlockFunc func()

func DoNothing() {}

type manager struct {
	// keyToInstance records all standby instances
	// key is the object key(types.NamespacedName)
	keyToInstance map[string]*instance

	// all standby instances grouped by hash
	standby *groups

	// All locked instances grouped by owner key.
	// Adopting instances are instances which are locked and not observed.
	adopting *groups

	l sync.Mutex

	logger logr.Logger
}

func New(logger logr.Logger) Manager {
	return &manager{
		keyToInstance: make(map[string]*instance),
		standby:       newGroups(),
		adopting:      newGroups(),
		logger:        logger,
	}
}

func (m *manager) deleteKey(key string) {
	item, ok := m.keyToInstance[key]
	if !ok {
		return
	}
	m.logger.Info("unregister instance", "instance", key)
	m.standby.del(item.hash, key)
	m.adopting.del(item.ownerKey(), key)
	delete(m.keyToInstance, key)
}

func (m *manager) lock(owner *v1alpha1.TiDBGroup, in *instance) {
	in.lock(owner)
	m.standby.del(in.hash, in.key())
	m.adopting.add(in.ownerKey(), in.key())
}

func (m *manager) unlockFunc(in *instance) UnlockFunc {
	return func() {
		m.l.Lock()
		defer m.l.Unlock()

		// has been unlocked
		if !in.isLocked() {
			return
		}

		cur, ok := m.keyToInstance[in.key()]
		if !ok || cur != in {
			// previous instance has been deleted
			return
		}

		m.logger.Info("unlock adopting standby instance", "group", in.ownerKey(), "instance", in.key(), "hash", in.hash)

		m.adopting.del(in.ownerKey(), in.key())
		m.standby.add(in.hash, in.key())
		in.unlock()
	}
}

// Register records all standby tidbs and unlock tidbs which have been adopted or are deleted
func (m *manager) Register(dbs ...*v1alpha1.TiDB) {
	m.l.Lock()
	defer m.l.Unlock()

	for _, db := range dbs {
		m.register(db)
	}
}

func (m *manager) Unregister(ns, name string) {
	m.l.Lock()
	defer m.l.Unlock()

	m.deleteKey(client.ObjectKey{
		Namespace: ns,
		Name:      name,
	}.String())
}

func (m *manager) register(db *v1alpha1.TiDB) {
	key := client.ObjectKeyFromObject(db).String()
	// has been adopted or is deleted
	if db.Spec.Mode == v1alpha1.TiDBModeNormal || !db.DeletionTimestamp.IsZero() {
		m.deleteKey(key)
		return
	}

	item, ok := m.keyToInstance[key]
	if !ok {
		hash := hashTiDB(db)
		m.keyToInstance[key] = &instance{
			hash:     hash,
			instance: db.DeepCopy(),
		}
		m.standby.add(hash, key)

		m.logger.Info("register standby instance", "instance", key, "hash", hash, "rv", db.GetResourceVersion())

		return
	}

	rvChanged := item.instance.ResourceVersion != db.ResourceVersion
	uidChanged := item.instance.UID != db.UID

	// instance is not changed
	if !rvChanged && !uidChanged {
		return
	}

	nHash := hashTiDB(db)
	oHash := item.hash

	item.hash = nHash
	item.instance = db.DeepCopy()

	m.logger.Info("update standby instance", "instance", key, "hash", nHash, "rv", db.GetResourceVersion())
	// - applied and uid is not changed
	//   - instance has been adopted and mode is changed to normal.
	//   - just wait until the instance is synced.
	// - lock but not applied yet
	//   - apply will fail because rv or uid is changed and then the instance will be unlocked.
	// - applied but uid is changed
	//   - adopted instance has been deleted and recreated.
	//   - we'll register when the instance is deleting so that no need to do anything
	if item.isLocked() {
		m.logger.Info("standby instance is still locked", "instance", key)

		return
	}

	if oHash != nHash {
		m.logger.Info("standby instance hash changed", "instance", key, "old hash", oHash, "new hash", nHash)

		m.standby.del(oHash, key)
		m.standby.add(nHash, key)
	}
}

// Adopt will returns an adopting instance for updater to apply.
// 1. StandBy -> Adopting -> Applied -> Observed as Normal
// 2. StandBy -> Adopting -> Apply Failed -> Unlock to StandBy
// 3. standBy -> Adopting -> Applied -> Adopt again
func (m *manager) Adopt(dbg *v1alpha1.TiDBGroup, index int) (*v1alpha1.TiDB, UnlockFunc) {
	if dbg.Spec.Template.Spec.Mode == v1alpha1.TiDBModeStandBy {
		return nil, DoNothing
	}

	m.l.Lock()
	defer m.l.Unlock()

	key := client.ObjectKeyFromObject(dbg).String()
	adopting := m.adopting.list(key)
	if index < len(adopting) {
		inKey := adopting[index]
		// should always exist
		item := m.keyToInstance[inKey]

		m.logger.Info("return adopting standby instance",
			"group", key,
			"instance", inKey,
			"index", index,
			"rv", item.instance.GetResourceVersion())

		return item.instance.DeepCopy(), DoNothing
	}

	hash := hashTiDBGroup(dbg)
	standby := m.standby.list(hash)

	m.logger.Info("try to adopt standby instance", "group", key, "hash", hash, "index", index, "matched", len(standby))

	for _, k := range standby {
		item, ok := m.keyToInstance[k]
		if !ok {
			continue
		}
		m.lock(dbg, item)

		m.logger.Info("adopting standby instance",
			"group", key,
			"instance", k,
			"hash", item.hash,
			"index", index,
			"rv", item.instance.GetResourceVersion(),
		)

		return item.instance.DeepCopy(), m.unlockFunc(item)
	}

	// no suitable standby tidb can be adopted
	return nil, DoNothing
}
