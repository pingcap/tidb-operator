// Copyright 2023 PingCAP, Inc.
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

package middlewares

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pingcap/log"
	"go.uber.org/zap"
)

func LoggingMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Starting time
		startTime := time.Now()

		var body []byte
		if log.GetLevel().Enabled(zap.DebugLevel) {
			// copy body for logging
			body, _ = io.ReadAll(ctx.Request.Body)
			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		// Processing request
		ctx.Next()

		// End Time
		endTime := time.Now()

		// execution time
		latencyTime := endTime.Sub(startTime)

		// Request method
		reqMethod := ctx.Request.Method

		// Request route
		reqUri := ctx.Request.RequestURI

		// status code
		statusCode := ctx.Writer.Status()

		// Request IP
		clientIP := ctx.ClientIP()

		logger := log.L().With(
			zap.String("method", reqMethod),
			zap.String("uri", reqUri),
			zap.Int("status", statusCode),
			zap.Duration("latency", latencyTime),
			zap.String("client", clientIP),
			zap.Any("params", ctx.Request.URL.Query()),
		)
		if errs := ctx.Errors.ByType(gin.ErrorTypePrivate); len(errs) > 0 {
			logger = logger.With(zap.String("error", errs.String()))
		}

		if log.GetLevel().Enabled(zap.DebugLevel) {
			logger = logger.With(zap.ByteString("body", body))
			logger.Debug("")
		} else {
			logger.Info("")
		}
	}
}
