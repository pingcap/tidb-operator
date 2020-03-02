// Copyright 2020 PingCAP, Inc.
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

package util

import (
	"database/sql"
	"fmt"

	"github.com/pingcap/tidb-operator/cmd/backup-manager/app/constants"
)

// GenericOptions contains the generic input arguments to the backup/restore command
type GenericOptions struct {
	Namespace string
	// ResourceName can be the name of a backup or restore resource
	ResourceName string
	Host         string
	Port         int32
	Password     string
	User         string
}

func (bo *GenericOptions) String() string {
	return fmt.Sprintf("%s/%s", bo.Namespace, bo.ResourceName)
}

func (bo *GenericOptions) GetDSN() string {
	return fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8", bo.User, bo.Password, bo.Host, bo.Port, constants.TidbMetaDB)
}

func (bo *GenericOptions) GetTikvGCLifeTime(db *sql.DB) (string, error) {
	var tikvGCTime string
	sql := fmt.Sprintf("select variable_value from %s where variable_name= ?", constants.TidbMetaTable)
	row := db.QueryRow(sql, constants.TikvGCVariable)
	err := row.Scan(&tikvGCTime)
	if err != nil {
		return tikvGCTime, fmt.Errorf("query cluster %s %s failed, sql: %s, err: %v", bo, constants.TikvGCVariable, sql, err)
	}
	return tikvGCTime, nil
}

func (bo *GenericOptions) SetTikvGCLifeTime(db *sql.DB, gcTime string) error {
	sql := fmt.Sprintf("update %s set variable_value = ? where variable_name = ?", constants.TidbMetaTable)
	_, err := db.Exec(sql, gcTime, constants.TikvGCVariable)
	if err != nil {
		return fmt.Errorf("set cluster %s %s failed, sql: %s, err: %v", bo, constants.TikvGCVariable, sql, err)
	}
	return nil
}
