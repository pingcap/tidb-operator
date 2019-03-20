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

package internal

import (
	"database/sql"
	"math/rand"
	"strings"

	"github.com/golang/glog"
)

func padLeft(str, pad string, length int) string {
	if len(str) >= length {
		return str
	}
	padding := strings.Repeat(pad, length)
	str = padding + str
	return str[len(str)-length:]
}

func padRight(str, pad string, length int) string {
	if len(str) >= length {
		return str
	}
	padding := strings.Repeat(pad, length)
	str = str + padding
	return str[:length]
}

func enableTiKVGC(db *sql.DB) {
	sql := "update mysql.tidb set VARIABLE_VALUE = '10m' where VARIABLE_NAME = 'tikv_gc_life_time';"
	_, err := db.Exec(sql)
	if err != nil {
		glog.Warningf("Failed to enable TiKV GC")
	}
}

func disableTiKVGC(db *sql.DB) {
	sql := "update mysql.tidb set VARIABLE_VALUE = '500h' where VARIABLE_NAME = 'tikv_gc_life_time';"
	_, err := db.Exec(sql)
	if err != nil {
		glog.Warningf("Failed to disable TiKV GC")
	}
}

// parallel run functions in parallel and wait until all of them are completed.
// If one of them returns error, the result is that error.
func parallel(funcs ...func() error) error {
	cr := make(chan error, len(funcs))
	for _, foo := range funcs {
		go func(foo func() error) {
			err := foo()
			cr <- err
		}(foo)
	}
	var err error
	for i := 0; i < len(funcs); i++ {
		r := <-cr
		if r != nil {
			err = r
		}
	}
	return err
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randSeq(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

const numberBytes = "0123456789"

func randNum(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = numberBytes[rand.Int63()%int64(len(numberBytes))]
	}
	return b
}

func randMD() (m int, d int) {
	for m == 0 {
		m = rand.Intn(MAXDECIMALM)
	}
	min := m
	if min > MAXDECIMALN {
		min = MAXDECIMALN
	}
	d = rand.Intn(min)
	return
}

func randDecimal(m, d int) string {
	ms := randNum(m - d)
	ds := randNum(d)
	var i int
	for i = range ms {
		if ms[i] != byte('0') {
			break
		}
	}
	ms = ms[i:]
	l := len(ms) + len(ds) + 1
	flag := rand.Intn(2)
	//check for 0.0... avoid -0.0
	zeroFlag := true
	for i := range ms {
		if ms[i] != byte('0') {
			zeroFlag = false
		}
	}
	for i := range ds {
		if ds[i] != byte('0') {
			zeroFlag = false
		}
	}
	if zeroFlag {
		flag = 0
	}
	vs := make([]byte, 0, l+flag)
	if flag == 1 {
		vs = append(vs, '-')
	}
	vs = append(vs, ms...)
	if len(ds) == 0 {
		return string(vs)
	}
	vs = append(vs, '.')
	vs = append(vs, ds...)
	return string(vs)
}

const FieldNameLen = 8

func randFieldName(m map[string]interface{}) string {
	name := randSeq(FieldNameLen)
	_, ok := m[name]
	for ok {
		name = randSeq(FieldNameLen)
		_, ok = m[name]
	}
	return name
}

const EnumValueLen = 5

func randEnumString(m map[string]struct{}) string {
	l := rand.Intn(EnumValueLen) + 1
	name := randSeq(l)
	nameL := strings.ToLower(name)
	_, ok := m[nameL]
	for ok {
		l = rand.Intn(EnumValueLen) + 1
		name = randSeq(l)
		nameL = strings.ToLower(name)
		_, ok = m[nameL]
	}
	m[nameL] = struct{}{}
	return name
}
