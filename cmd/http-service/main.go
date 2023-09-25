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

package main

import (
	"context"
	"errors"
	"flag"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pingcap/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pingcap/tidb-operator/http-service/middlewares"
	"github.com/pingcap/tidb-operator/http-service/pbgen/api"
	"github.com/pingcap/tidb-operator/http-service/server"
	"github.com/pingcap/tidb-operator/http-service/version"
)

const (
	defaultLogMaxDays = 7
	defaultLogMaxSize = 512 // MB
)

func main() {
	// init config
	cfg := NewConfig()
	err := cfg.Parse(os.Args[1:])
	switch err {
	case nil:
	case flag.ErrHelp:
		os.Exit(0)
	default:
		os.Exit(2)
	}

	// init logger
	logger, props, err := log.InitLogger(&log.Config{
		Level: cfg.LogLevel,
		File: log.FileLogConfig{
			Filename: cfg.LogFile,
			MaxSize:  defaultLogMaxSize,
			MaxDays:  defaultLogMaxDays,
		}})
	if err != nil {
		panic(err)
	}
	log.ReplaceGlobals(logger, props)
	log.Info("Starting http-service", zap.String("version", version.GetRawInfo()))
	defer log.Sync()

	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		setupAndRunGRPCServer(ctx, log.L(), cfg.InternalGRPCAddr)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		setupAndRunGRPCGateway(ctx, log.L(), cfg.Addr, cfg.InternalGRPCAddr)
	}()

	wg.Wait()
	stop()

	log.Info("HTTP service exiting")
}

func setupAndRunGRPCServer(ctx context.Context, logger *zap.Logger, addr string) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("Failed to listen for gRPC server", zap.String("addr", addr), zap.Error(err))
	}
	s := grpc.NewServer()
	api.RegisterClusterServer(s, &server.ClusterServer{})
	log.Info("Starting internal gRPC server", zap.String("addr", addr))
	go func() {
		if err2 := s.Serve(lis); err2 != nil {
			log.Info("Internal gRPC server returned with error", zap.String("addr", addr), zap.Error(err2))
		}
	}()

	<-ctx.Done()

	log.Info("Stopping internal gRPC server...", zap.String("addr", addr))
	s.GracefulStop()
	log.Info("Internal gRPC server stopped", zap.String("addr", addr))
}

func setupAndRunGRPCGateway(ctx context.Context, logger *zap.Logger, addr, grpcAddr string) {
	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(CustomMatcher),
	)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := api.RegisterClusterHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		log.Fatal("Failed to register gRPC gateway", zap.Error(err))
	}

	router := gin.New()
	router.Use(middlewares.LoggingMiddleware(), gin.Recovery()) // log with custom format
	router.Group("/v1beta/*{grpc_gateway}").Any("", gin.WrapH(mux))

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	log.Info("Starting gRPC gateway", zap.String("addr", addr))
	go func() {
		if err2 := srv.ListenAndServe(); err2 != nil && !errors.Is(err2, http.ErrServerClosed) {
			log.Info("The gRPC gateway returned with error", zap.String("addr", addr), zap.Error(err2))
		}
	}()

	<-ctx.Done()

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	log.Info("Stopping gRPC gatway...", zap.String("addr", addr))
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("gRPC gateway forced to shutdown", zap.String("addr", addr), zap.Error(err))
	} else {
		log.Info("gRPC gateway stopped", zap.String("addr", addr))
	}
}

func CustomMatcher(key string) (string, bool) {
	switch key {
	case textproto.CanonicalMIMEHeaderKey(server.HeaderKeyKubernetesID):
		return key, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}
