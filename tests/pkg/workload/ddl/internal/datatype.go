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
	"math/rand"
	"time"
)

// Kind constants.
const (
	KindTINYINT   int = iota
	KindSMALLINT      //int16
	KindMEDIUMINT     //int24
	KindInt32         //int32
	KindBigInt        //int64
	KindBit

	KindFloat
	KindDouble

	KindDECIMAL

	KindChar
	KindVarChar

	KindBLOB
	KindTINYBLOB
	KindMEDIUMBLOB
	KindLONGBLOB

	KindTEXT
	KindTINYTEXT
	KindMEDIUMTEXT
	KindLONGTEXT

	KindBool

	KindDATE
	KindTIME
	KindDATETIME
	KindTIMESTAMP
	KindYEAR

	KindJSON

	KindEnum
	KindSet
)

var ALLFieldType = map[int]string{
	KindTINYINT:   "TINYINT",
	KindSMALLINT:  "SMALLINT",
	KindMEDIUMINT: "MEDIUMINT",
	KindInt32:     "INT",
	KindBigInt:    "BIGINT",
	KindBit:       "BIT",

	KindFloat:  "FLOAT",
	KindDouble: "DOUBLE",

	KindDECIMAL: "DECIMAL",

	KindChar:    "CHAR",
	KindVarChar: "VARCHAR",

	KindBLOB:       "BLOB",
	KindTINYBLOB:   "TINYBLOB",
	KindMEDIUMBLOB: "MEDIUMBLOB",
	KindLONGBLOB:   "LONGBLOB",

	KindTEXT:       "TEXT",
	KindTINYTEXT:   "TINYTEXT",
	KindMEDIUMTEXT: "MEDIUMTEXT",
	KindLONGTEXT:   "LONGTEXT",

	KindBool: "BOOL",

	KindDATE:      "DATE",
	KindTIME:      "TIME",
	KindDATETIME:  "DATETIME",
	KindTIMESTAMP: "TIMESTAMP",
	KindYEAR:      "YEAR",

	KindJSON: "JSON",
	KindEnum: "ENUM",
	KindSet:  "SET",
}

// testFieldType is use to control what kind of data types to test,
// if we just want to test timestamp, we can annotation the another data types.
var testFieldType = []int{
	KindTINYINT,
	KindSMALLINT,
	KindMEDIUMINT,
	KindInt32,
	KindBigInt,

	//KindBit,		// have default value bug unfix.

	KindFloat,
	KindDouble,

	KindDECIMAL,

	KindChar,
	KindVarChar,

	KindBLOB,
	KindTEXT,

	KindBool,

	KindDATE,
	KindTIME,
	KindDATETIME,
	KindTIMESTAMP,
	KindYEAR,

	//KindJSON,			// have `admin check table when index is virtual generated column` bug unfix
	KindEnum,
	KindSet,
}

func randDataType() int {
	i := rand.Intn(len(testFieldType))
	return testFieldType[i]
}

const (
	BitMaxLen        = 64
	CharMaxLen       = 256
	VarCharMaxLen    = 256 // varchar max len , actual range is [0,65536)
	BLOBMaxLen       = 256 // BLOB max len , actual range is [0,65536)
	TINYBLOBMaxLen   = 256 // TINYBLOB max len , actual range is [0,256)
	MEDIUMBLOBMaxLen = 256 // MEDIUMBLOB max len , actual range is [0,16777216)
	LONGBLOBMaxLen   = 256 // LONGBLOB max len , actual range is [0,4294967296)

	TEXTMaxLen       = 256 // TEXT max len , actual range is [0,65536)
	TINYTEXTMaxLen   = 256 // TINYTEXT max len , actual range is [0,256)
	MEDIUMTEXTMaxLen = 256 // MEDIUMTEXT max len , actual range is [0,16777216)
	LONGTEXTMaxLen   = 256 // LONGTEXT max len , actual range is [0,4294967296)

	MAXDECIMALM = 65 // 1~65
	MAXDECIMALN = 30 // 0~30

	EnumMaxLen = 10
	SetMaxLen  = 10

	TimeFormat        = "2006-01-02 15:04:05"
	TimeFormatForDATE = "2006-01-02"
	TimeFormatForTIME = "15:04:05"

	MINDATETIME = "1000-01-01 00:00:00"
	MAXDATETIME = "9999-12-31 23:59:59"

	MINTIMESTAMP = "1970-01-01 00:00:01"
	MAXTIMESTAMP = "2038-01-19 03:14:07"
)

var MinDATETIME time.Time
var MaxDATETIME time.Time
var GapDATETIMEUnix int64

var MinTIMESTAMP time.Time
var MaxTIMESTAMP time.Time
var GapTIMESTAMPUnix int64

func getMaxLenByKind(kind int) int {
	switch kind {
	case KindChar:
		return CharMaxLen
	case KindVarChar:
		return VarCharMaxLen
	case KindBLOB:
		return BLOBMaxLen
	case KindTINYBLOB:
		return TINYBLOBMaxLen
	case KindMEDIUMBLOB:
		return MEDIUMBLOBMaxLen
	case KindLONGBLOB:
		return LONGBLOBMaxLen
	case KindTEXT:
		return TEXTMaxLen
	case KindTINYTEXT:
		return TINYTEXTMaxLen
	case KindMEDIUMTEXT:
		return MEDIUMTEXTMaxLen
	case KindLONGTEXT:
		return LONGTEXTMaxLen
	case KindBit:
		return BitMaxLen
	case KindEnum:
		return EnumMaxLen
	case KindSet:
		return SetMaxLen
	}
	return 0
}

type ambiguousTimeStr struct {
	start string
	end   string
}

type ambiguousTime struct {
	start int64
	end   int64
}

var ambiguousTimeStrSlice = []ambiguousTimeStr{
	// backward
	{
		start: "1900-12-31 23:54:17",
		end:   "1901-01-01 00:00:00",
	},
	// moved forward
	{
		start: "1940-06-02 23:59:59",
		end:   "1940-06-03 01:00:00",
	},
	// move backward
	{
		start: "1940-09-30 23:00:00",
		end:   "1940-10-01 00:00:00",
	},
	// moved forward
	{
		start: "1941-03-15 23:59:59",
		end:   "1941-03-16 01:00:00",
	},
	// move backward
	{
		start: "1941-09-30 23:00:00",
		end:   "1941-10-01 00:00:00",
	},
	// moved forward
	{
		start: "1986-05-03 23:59:59",
		end:   "1986-05-04 01:00:00",
	},
	// move backward
	{
		start: "1986-09-13 23:00:00",
		end:   "1986-09-14 00:00:00",
	},
	// moved forward
	{
		start: "1987-04-11 23:59:59",
		end:   "1987-04-12 01:00:00",
	},
	// move backward
	{
		start: "1987-09-12 23:00:00",
		end:   "1987-09-13 00:00:00",
	},
	// moved forward
	{
		start: "1988-04-09 23:59:59",
		end:   "1988-04-10 01:00:00",
	},
	// move backward
	{
		start: "1988-09-10 23:00:00",
		end:   "1988-09-11 00:00:00",
	},

	// moved forward
	{
		start: "1989-04-15 23:59:59",
		end:   "1989-04-16 01:00:00",
	},
	// move backward
	{
		start: "1989-09-16 23:00:00",
		end:   "1989-09-17 00:00:00",
	},
	// moved forward
	{
		start: "1990-04-14 23:59:59",
		end:   "1990-04-15 01:00:00",
	},
	// move backward
	{
		start: "1990-09-15 23:00:00",
		end:   "1990-09-16 00:00:00",
	},
	// moved forward
	{
		start: "1991-04-13 23:59:59",
		end:   "1991-04-14 01:00:00",
	},
	// move backward
	{
		start: "1991-09-14 23:00:00",
		end:   "1991-09-15 00:00:00",
	},
}

var ambiguousTimeSlice []ambiguousTime

var local = time.Local

func init() {
	var err error
	local, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		local = time.Local
	}
	for _, v := range ambiguousTimeStrSlice {
		start, _ := time.ParseInLocation(TimeFormat, v.start, local)
		end, _ := time.ParseInLocation(TimeFormat, v.end, local)
		amt := ambiguousTime{
			start: start.Unix(),
			end:   end.Unix(),
		}
		ambiguousTimeSlice = append(ambiguousTimeSlice, amt)
	}

	MinDATETIME, _ = time.ParseInLocation(TimeFormat, MINDATETIME, local)
	MaxDATETIME, _ = time.ParseInLocation(TimeFormat, MAXDATETIME, local)
	GapDATETIMEUnix = MaxDATETIME.Unix() - MinDATETIME.Unix()

	MinTIMESTAMP, _ = time.ParseInLocation(TimeFormat, MINTIMESTAMP, local)
	MaxTIMESTAMP, _ = time.ParseInLocation(TimeFormat, MAXTIMESTAMP, local)
	GapTIMESTAMPUnix = MaxTIMESTAMP.Unix() - MinTIMESTAMP.Unix()
}
