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

package util

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

type glogEncoder struct {
	zapcore.Encoder
	zapcore.EncoderConfig
}

func (ge glogEncoder) Clone() zapcore.Encoder {
	return glogEncoder{Encoder: ge.Encoder.Clone(), EncoderConfig: ge.EncoderConfig}
}

func (ge glogEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf := bufpool.Get()

	switch ent.Level {
	case zapcore.DebugLevel:
		buf.AppendByte('D')
	case zapcore.InfoLevel:
		buf.AppendByte('I')
	case zapcore.WarnLevel:
		buf.AppendByte('W')
	case zapcore.ErrorLevel:
		buf.AppendByte('E')
	case zapcore.DPanicLevel, zap.PanicLevel:
		buf.AppendByte('P')
	case zapcore.FatalLevel:
		buf.AppendByte('F')
	default:
		buf.AppendString(fmt.Sprintf("%d!", ent.Level))
	}

	buf.AppendString(ent.Time.Format("0102 15:04:05.000000"))
	k := 8
	for n := pid; n > 0; k-- {
		n /= 10
	}
	for ; k > 0; k-- {
		buf.AppendByte(' ')
	}
	buf.AppendInt(int64(pid))
	buf.AppendByte(' ')
	buf.AppendString(ent.Caller.TrimmedPath())
	buf.AppendString("] ")

	enc := ge.Encoder.Clone()
	line, _ := enc.EncodeEntry(ent, fields)
	buf.AppendString(line.String())
	line.Free()

	return buf, nil
}

var (
	pid     = os.Getpid()
	bufpool = buffer.NewPool()
)

func NewGLogEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	cfgCopy := cfg
	cfgCopy.TimeKey = ""
	cfgCopy.LevelKey = ""
	cfgCopy.NameKey = ""
	cfgCopy.CallerKey = ""
	return glogEncoder{Encoder: zapcore.NewConsoleEncoder(cfgCopy), EncoderConfig: cfg}
}

func NewGLogDevConfig() zap.Config {
	return zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.DebugLevel),
		Development:      true,
		Encoding:         "glog",
		EncoderConfig:    zap.NewDevelopmentEncoderConfig(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
}

func NewGLogDev(options ...zap.Option) (*zap.Logger, error) {
	return NewGLogDevConfig().Build(options...)
}

func init() {
	zap.RegisterEncoder("glog", func(config zapcore.EncoderConfig) (encoder zapcore.Encoder, e error) {
		return NewGLogEncoder(config), nil
	})
}
