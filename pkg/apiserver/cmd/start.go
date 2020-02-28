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

package cmd

import (
	"flag"
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apiserver/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapifilters "k8s.io/apiserver/pkg/endpoints/filters"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/server"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericfilters "k8s.io/apiserver/pkg/server/filters"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/logs"
	"k8s.io/klog"
	openapi "k8s.io/kube-openapi/pkg/common"

	"sigs.k8s.io/apiserver-builder-alpha/pkg/apiserver"
	"sigs.k8s.io/apiserver-builder-alpha/pkg/builders"
)

var GetOpenApiDefinition openapi.GetOpenAPIDefinitions

type ServerOptions struct {
	RecommendedOptions *genericoptions.RecommendedOptions
	APIBuilders        []*builders.APIGroupBuilder

	InsecureServingOptions *genericoptions.DeprecatedInsecureServingOptionsWithLoopback

	PrintBearerToken bool
	RunDelegatedAuth bool
	BearerToken      string
	PostStartHooks   []PostStartHook

	GetOpenApiDefinition openapi.GetOpenAPIDefinitions

	RestConfig        *rest.Config
	StorageNamespace  string
	StorageKubeConfig string
	Codec             runtime.Codec
}

type PostStartHook struct {
	Fn   genericapiserver.PostStartHookFunc
	Name string
}

// StartApiServer starts an apiserver.
func StartApiServer(
	apis []*builders.APIGroupBuilder,
	openapidefs openapi.GetOpenAPIDefinitions,
	title,
	version string) {
	logs.InitLogs()
	defer logs.FlushLogs()

	GetOpenApiDefinition = openapidefs

	signalCh := genericapiserver.SetupSignalHandler()
	// To disable providers, manually specify the list provided by getKnownProviders()
	cmd := NewCommandStartServer(apis, signalCh, title, version)

	cmd.Flags().AddFlagSet(pflag.CommandLine)
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

// NewCommandStartServer create a cobra start server command
func NewCommandStartServer(builders []*builders.APIGroupBuilder, stopCh <-chan struct{}, title, version string) *cobra.Command {
	o := NewServerOptions(builders)

	// Support overrides
	cmd := &cobra.Command{
		Short: "Launch an API server",
		Long:  "Launch an API server",
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return o.RunServer(stopCh, title, version)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&o.PrintBearerToken, "print-bearer-token", false,
		"Print a curl command with the bearer token to test the server")
	flags.BoolVar(&o.RunDelegatedAuth, "delegated-auth", true,
		"Setup delegated auth")
	flags.StringVar(&o.StorageNamespace, "storage-namespace", "default",
		"Kubernetes namespace that data stored to")
	flags.StringVar(&o.StorageKubeConfig, "storage-kubeconfig", "",
		"kubeconfig path used to build the client of kubernetes, use in-cluster config is not specified")
	o.RecommendedOptions.AddFlags(flags)
	o.InsecureServingOptions.AddFlags(flags)

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	flags.AddGoFlagSet(klogFlags)

	// Sync the klog and klog flags.
	klogFlags.VisitAll(func(f *flag.Flag) {
		goFlag := flag.CommandLine.Lookup(f.Name)
		if goFlag != nil {
			goFlag.Value.Set(f.Value.String())
		}
	})

	return cmd
}

// NewServerOptions create a ServerOptions
func NewServerOptions(b []*builders.APIGroupBuilder) *ServerOptions {
	internalVersions := []schema.GroupVersion{}
	for _, b := range b {
		internalVersions = append(internalVersions, b.UnVersioned.GroupVersion)
	}

	codec := builders.Codecs.LegacyCodec(internalVersions...)
	o := &ServerOptions{
		RecommendedOptions: genericoptions.NewRecommendedOptions("", codec, nil),
		APIBuilders:        b,
		RunDelegatedAuth:   true,
		Codec:              codec,
	}

	o.RecommendedOptions.Authorization.RemoteKubeConfigFileOptional = true
	o.RecommendedOptions.Authentication.RemoteKubeConfigFileOptional = true
	o.InsecureServingOptions = func() *genericoptions.DeprecatedInsecureServingOptionsWithLoopback {
		o := genericoptions.DeprecatedInsecureServingOptions{}
		return o.WithLoopback()
	}()

	return o
}

// Config create an apiserver config
func (o ServerOptions) Config() (*apiserver.Config, error) {
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts(
		"localhost", nil, nil); err != nil {

		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewRecommendedConfig(builders.Codecs)
	serverConfig.Config.RESTOptionsGetter = &storage.ApiServerRestOptionsFactory{
		RestConfig:       o.RestConfig,
		StorageNamespace: o.StorageNamespace,
		Codec:            o.Codec,
	}

	err := applyOptions(
		&serverConfig.Config,
		func(cfg *genericapiserver.Config) error {
			return o.RecommendedOptions.SecureServing.ApplyTo(&cfg.SecureServing, &cfg.LoopbackClientConfig)
		},
		func(cfg *genericapiserver.Config) error {
			return o.RecommendedOptions.Audit.ApplyTo(cfg, nil, nil, nil, nil)
		},
		o.RecommendedOptions.Features.ApplyTo,
	)
	if err != nil {
		return nil, err
	}

	loopbackKubeConfig, kubeInformerFactory, err := o.buildLoopback()
	if err != nil {
		klog.Warningf("attempting to instantiate loopback client but failed, %v", err)
	} else {
		serverConfig.LoopbackClientConfig = loopbackKubeConfig
		serverConfig.SharedInformerFactory = kubeInformerFactory
	}

	var insecureServingInfo *genericapiserver.DeprecatedInsecureServingInfo
	if err := o.InsecureServingOptions.ApplyTo(&insecureServingInfo, &serverConfig.LoopbackClientConfig); err != nil {
		return nil, err
	}
	config := &apiserver.Config{
		RecommendedConfig:   serverConfig,
		InsecureServingInfo: insecureServingInfo,
		PostStartHooks:      make(map[string]genericapiserver.PostStartHookFunc),
	}

	if o.RunDelegatedAuth {
		err := applyOptions(
			&serverConfig.Config,
			func(cfg *genericapiserver.Config) error {
				return o.RecommendedOptions.Authentication.ApplyTo(&cfg.Authentication, cfg.SecureServing, cfg.OpenAPIConfig)
			},
			func(cfg *genericapiserver.Config) error {
				return o.RecommendedOptions.Authorization.ApplyTo(&cfg.Authorization)
			},
		)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func (o *ServerOptions) Complete() error {
	// If storageKubeConfig provided, use that in favor of local test
	var err error
	if o.StorageKubeConfig != "" {
		o.RestConfig, err = clientcmd.BuildConfigFromFlags("", o.StorageKubeConfig)
	} else {
		o.RestConfig, err = rest.InClusterConfig()
	}
	return err
}

func (o *ServerOptions) Validate() error {
	if o.StorageNamespace == "" {
		return fmt.Errorf("Storage namespace must not be empty")
	}
	if o.RestConfig == nil {
		return fmt.Errorf("Kubernetes rest client config must not be nil")
	}
	return nil
}

func (o *ServerOptions) RunServer(stopCh <-chan struct{}, title, version string) error {
	aggregatedAPIServerConfig, err := o.Config()
	if err != nil {
		return err
	}
	genericConfig := aggregatedAPIServerConfig.RecommendedConfig.Config

	if o.PrintBearerToken {
		klog.Infof("Serving on loopback...")
		klog.Infof("\n\n********************************\nTo test the server run:\n"+
			"curl -k -H \"Authorization: Bearer %s\" %s\n********************************\n\n",
			genericConfig.LoopbackClientConfig.BearerToken,
			genericConfig.LoopbackClientConfig.Host)
	}
	o.BearerToken = genericConfig.LoopbackClientConfig.BearerToken

	for _, provider := range o.APIBuilders {
		aggregatedAPIServerConfig.AddApi(provider)
	}

	aggregatedAPIServerConfig.Init()

	genericConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(GetOpenApiDefinition, openapinamer.NewDefinitionNamer(builders.Scheme))
	genericConfig.OpenAPIConfig.Info.Title = title
	genericConfig.OpenAPIConfig.Info.Version = version

	genericServer, err := aggregatedAPIServerConfig.Complete().New()
	if err != nil {
		return err
	}

	for _, h := range o.PostStartHooks {
		if err := genericServer.GenericAPIServer.AddPostStartHook(h.Name, h.Fn); err != nil {
			return err
		}
	}

	s := genericServer.GenericAPIServer.PrepareRun()
	if aggregatedAPIServerConfig.InsecureServingInfo != nil {
		handler := s.GenericAPIServer.UnprotectedHandler()
		handler = genericapifilters.WithAudit(handler, genericConfig.AuditBackend, genericConfig.AuditPolicyChecker, genericConfig.LongRunningFunc)
		handler = genericapifilters.WithAuthentication(handler, server.InsecureSuperuser{}, nil, nil)
		handler = genericfilters.WithCORS(handler, genericConfig.CorsAllowedOriginList, nil, nil, nil, "true")
		handler = genericfilters.WithTimeoutForNonLongRunningRequests(handler, genericConfig.LongRunningFunc, genericConfig.RequestTimeout)
		handler = genericfilters.WithMaxInFlightLimit(handler, genericConfig.MaxRequestsInFlight, genericConfig.MaxMutatingRequestsInFlight, genericConfig.LongRunningFunc)
		handler = genericapifilters.WithRequestInfo(handler, server.NewRequestInfoResolver(&genericConfig))
		handler = genericfilters.WithPanicRecovery(handler)
		if err := aggregatedAPIServerConfig.InsecureServingInfo.Serve(handler, genericConfig.RequestTimeout, stopCh); err != nil {
			return err
		}
	}

	return s.Run(stopCh)
}

func (o *ServerOptions) buildLoopback() (*rest.Config, informers.SharedInformerFactory, error) {
	var loopbackConfig *rest.Config
	var err error
	if len(o.RecommendedOptions.CoreAPI.CoreAPIKubeconfigPath) == 0 {
		klog.Infof("loading in-cluster loopback client...")
		loopbackConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, err
		}
	} else {
		klog.Infof("loading out-of-cluster loopback client according to `--kubeconfig` settings...")
		loopbackConfig, err = clientcmd.BuildConfigFromFlags("", o.RecommendedOptions.CoreAPI.CoreAPIKubeconfigPath)
		if err != nil {
			return nil, nil, err
		}
	}
	loopbackClient, err := kubernetes.NewForConfig(loopbackConfig)
	if err != nil {
		return nil, nil, err
	}
	kubeInformerFactory := informers.NewSharedInformerFactory(loopbackClient, 0)
	return loopbackConfig, kubeInformerFactory, nil
}

func applyOptions(config *genericapiserver.Config, applyTo ...func(*genericapiserver.Config) error) error {
	for _, fn := range applyTo {
		if err := fn(config); err != nil {
			return err
		}
	}
	return nil
}
