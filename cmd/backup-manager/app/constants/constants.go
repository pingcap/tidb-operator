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

package constants

import "time"

const (
	// ResyncDuration is the informer's resync duration
	ResyncDuration = 10 * time.Second

	// PollInterval is the interval to check if the tidb cluster is ready
	PollInterval = 5 * time.Second

	// CheckTimeout is the maximum time to wait for the tidb cluster ready
	CheckTimeout = 30 * time.Minute

	// BackupRootPath is the root path to backup data
	BackupRootPath = "/backup"

	// MetaDataFile is the file which store the mydumper's meta info
	MetaDataFile = "metadata"

	// TikvGCLifeTime is the safe gc life time for dump tidb cluster data
	TikvGCLifeTime = "72h"

	// TikvGCVariable is the tikv gc life time variable name
	TikvGCVariable = "tikv_gc_life_time"

	// TidbMetaDB is the database name for store meta info
	TidbMetaDB = "mysql"

	// TidBMetaTable is the table name for store meta info
	TidbMetaTable = "tidb"

	// DefaultArchiveExtention represent the data archive type
	DefaultArchiveExtention = ".tgz"

	// RcloneConfigFile represents the path to the file that contains rclone
	// configs. This path should be the same as defined in docker entrypoint
	// script from backup-manager/entrypoint.sh. /tmp/rclone.conf
	RcloneConfigFile = "/tmp/rclone.conf"

	// RcloneConfigArg represents the config argument to rclone cmd
	RcloneConfigArg = "--config=" + RcloneConfigFile

	// MetaFile is the file name for meta data of backup with BR
	MetaFile = "backupmeta"

	// BR certificate storage path
	BRCertPath = "/var/lib/br-tls"

	// ServiceAccountCAPath is where is CABundle of serviceaccount locates
	ServiceAccountCAPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

	// DefaultTableRegex is the default regular expression for 'db.table' matching
	DefaultTableRegex = "^(?!(mysql|INFORMATION_SCHEMA|PERFORMANCE_SCHEMA|METRICS_SCHEMA|INSPECTION_SCHEMA))"
)
