package apiserver

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/klog"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"

	openapiproto "k8s.io/kube-openapi/pkg/util/proto"
)

type APIGroupInstaller struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

// Exposes given api groups in the API.
func (s *APIGroupInstaller) InstallAPIGroups(apiGroupInfos ...*genericapiserver.APIGroupInfo) error {
	for _, apiGroupInfo := range apiGroupInfos {
		// Do not register empty group or empty version.  Doing so claims /apis/ for the wrong entity to be returned.
		// Catching these here places the error  much closer to its origin
		if len(apiGroupInfo.PrioritizedVersions[0].Group) == 0 {
			return fmt.Errorf("cannot register handler with an empty group for %#v", *apiGroupInfo)
		}
		if len(apiGroupInfo.PrioritizedVersions[0].Version) == 0 {
			return fmt.Errorf("cannot register handler with an empty version for %#v", *apiGroupInfo)
		}
	}

	openAPIModels, err := s.getOpenAPIModels(genericapiserver.APIGroupPrefix, apiGroupInfos...)
	if err != nil {
		return fmt.Errorf("unable to get openapi models: %v", err)
	}

	for _, apiGroupInfo := range apiGroupInfos {
		if err := s.installAPIResources(genericapiserver.APIGroupPrefix, apiGroupInfo, openAPIModels); err != nil {
			return fmt.Errorf("unable to install api resources: %v", err)
		}

		//// setup discovery
		//// Install the version handler.
		//// Add a handler at /apis/<groupName> to enumerate all versions supported by this group.
		//apiVersionsForDiscovery := []metav1.GroupVersionForDiscovery{}
		//for _, groupVersion := range apiGroupInfo.PrioritizedVersions {
		//	// Check the config to make sure that we elide versions that don't have any resources
		//	if len(apiGroupInfo.VersionedResourcesStorageMap[groupVersion.Version]) == 0 {
		//		continue
		//	}
		//	apiVersionsForDiscovery = append(apiVersionsForDiscovery, metav1.GroupVersionForDiscovery{
		//		GroupVersion: groupVersion.String(),
		//		Version:      groupVersion.Version,
		//	})
		//}
		//preferredVersionForDiscovery := metav1.GroupVersionForDiscovery{
		//	GroupVersion: apiGroupInfo.PrioritizedVersions[0].String(),
		//	Version:      apiGroupInfo.PrioritizedVersions[0].Version,
		//}
		//apiGroup := metav1.APIGroup{
		//	Name:             apiGroupInfo.PrioritizedVersions[0].Group,
		//	Versions:         apiVersionsForDiscovery,
		//	PreferredVersion: preferredVersionForDiscovery,
		//}
		//
		//s.GenericAPIServer.DiscoveryGroupManager.AddGroup(apiGroup)
		//s.GenericAPIServer.Handler.GoRestfulContainer.Add(discovery.NewAPIGroupHandler(s.GenericAPIServer.Serializer, apiGroup).WebService())
	}
	return nil
}

// getOpenAPIModels is a private method for getting the OpenAPI models
func (s *APIGroupInstaller) getOpenAPIModels(apiPrefix string, apiGroupInfos ...*genericapiserver.APIGroupInfo) (openapiproto.Models, error) {
	//if s.GenericAPIServer.openAPIConfig == nil {
	//	return nil, nil
	//}
	//pathsToIgnore := openapiutil.NewTrie(s.openAPIConfig.IgnorePrefixes)
	//resourceNames := make([]string, 0)
	//for _, apiGroupInfo := range apiGroupInfos {
	//	groupResources, err := getResourceNamesForGroup(apiPrefix, apiGroupInfo, pathsToIgnore)
	//	if err != nil {
	//		return nil, err
	//	}
	//	resourceNames = append(resourceNames, groupResources...)
	//}
	//
	//Build the openapi definitions for those resources and convert it to proto models
	//openAPISpec, err := openapibuilder.BuildOpenAPIDefinitionsForResources(s.openAPIConfig, resourceNames...)
	//if err != nil {
	//	return nil, err
	//}
	//return utilopenapi.ToProtoModels(openAPISpec)
	return nil, nil
}

// installAPIResources is a private method for installing the REST storage backing each api groupversionresource
func (s *APIGroupInstaller) installAPIResources(apiPrefix string, apiGroupInfo *genericapiserver.APIGroupInfo, openAPIModels openapiproto.Models) error {
	for _, groupVersion := range apiGroupInfo.PrioritizedVersions {
		if len(apiGroupInfo.VersionedResourcesStorageMap[groupVersion.Version]) == 0 {
			klog.Warningf("Skipping API %v because it has no resources.", groupVersion)
			continue
		}

		apiGroupVersion := s.getAPIGroupVersion(apiGroupInfo, groupVersion, apiPrefix)
		if apiGroupInfo.OptionsExternalVersion != nil {
			apiGroupVersion.OptionsExternalVersion = apiGroupInfo.OptionsExternalVersion
		}
		apiGroupVersion.OpenAPIModels = openAPIModels
		//apiGroupVersion.MaxRequestBodyBytes = s.maxRequestBodyBytes

		if err := apiGroupVersion.InstallREST(s.GenericAPIServer.Handler.GoRestfulContainer); err != nil {
			return fmt.Errorf("unable to setup API %v: %v", apiGroupInfo, err)
		}
	}

	return nil
}

func (s *APIGroupInstaller) getAPIGroupVersion(apiGroupInfo *genericapiserver.APIGroupInfo, groupVersion schema.GroupVersion, apiPrefix string) *APIGroupVersion {
	storage := make(map[string]rest.Storage)
	for k, v := range apiGroupInfo.VersionedResourcesStorageMap[groupVersion.Version] {
		storage[strings.ToLower(k)] = v
	}
	version := s.newAPIGroupVersion(apiGroupInfo, groupVersion)
	version.Root = apiPrefix
	version.Storage = storage
	return version
}

func (s *APIGroupInstaller) newAPIGroupVersion(apiGroupInfo *genericapiserver.APIGroupInfo, groupVersion schema.GroupVersion) *APIGroupVersion {
	return &APIGroupVersion{
		GroupVersion:     groupVersion,
		MetaGroupVersion: apiGroupInfo.MetaGroupVersion,

		ParameterCodec:  apiGroupInfo.ParameterCodec,
		Serializer:      apiGroupInfo.NegotiatedSerializer,
		Creater:         apiGroupInfo.Scheme,
		Convertor:       apiGroupInfo.Scheme,
		UnsafeConvertor: runtime.UnsafeObjectConvertor(apiGroupInfo.Scheme),
		Defaulter:       apiGroupInfo.Scheme,
		Typer:           apiGroupInfo.Scheme,
		Linker:          runtime.SelfLinker(meta.NewAccessor()),

		EquivalentResourceRegistry: s.GenericAPIServer.EquivalentResourceRegistry,

		//Admit:             s.admissionControl,
		//MinRequestTimeout: s.minRequestTimeout,
		Authorizer: s.GenericAPIServer.Authorizer,
	}
}
