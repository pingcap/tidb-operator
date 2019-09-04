package apiserver

import (
	"context"
	"fmt"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"github.com/pingcap/tidb-operator/api/pkg/list"
	"net/http"
	"net/url"
	"strings"
	"time"

	"k8s.io/klog"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/handlers/negotiation"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	utiltrace "k8s.io/utils/trace"

	"k8s.io/apiserver/pkg/endpoints/handlers"
)

// getterFunc performs a get request with the given context and object name. The request
// may be used to deserialize an options object to pass to the getter.
type getterFunc func(ctx context.Context, name string, req *http.Request, trace *utiltrace.Trace) (runtime.Object, error)

// getResourceHandler is an HTTP handler function for get requests. It delegates to the
// passed-in getterFunc to perform the actual get.
func getResourceHandler(scope *handlers.RequestScope, getter getterFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		trace := utiltrace.New("Get", utiltrace.Field{"url", req.URL.Path})
		defer trace.LogIfLong(500 * time.Millisecond)

		namespace, name, err := scope.Namer.Name(req)
		if err != nil {
			errResponse(scope, err, w, req)
			return
		}
		ctx := req.Context()
		ctx = request.WithNamespace(ctx, namespace)

		outputMediaType, _, err := negotiation.NegotiateOutputMediaType(req, scope.Serializer, scope)
		if err != nil {
			errResponse(scope, err, w, req)
			return
		}

		result, err := getter(ctx, name, req, trace)
		if err != nil {
			errResponse(scope, err, w, req)
			return
		}

		trace.Step("About to write a response")
		transformResponseObject(ctx, scope, trace, req, w, http.StatusOK, outputMediaType, result)
		trace.Step("Transformed response object")
	}
}

// GetResource returns a function that handles retrieving a single resource from a rest.Storage object.
func GetResource(r rest.Getter, e rest.Exporter, scope *handlers.RequestScope) http.HandlerFunc {
	return getResourceHandler(scope,
		func(ctx context.Context, name string, req *http.Request, trace *utiltrace.Trace) (runtime.Object, error) {
			// check for export
			options := metav1.GetOptions{}
			if values := req.URL.Query(); len(values) > 0 {
				exports := metav1.ExportOptions{}
				if err := metainternalversion.ParameterCodec.DecodeParameters(values, scope.MetaGroupVersion, &exports); err != nil {
					err = errors.NewBadRequest(err.Error())
					return nil, err
				}
				if exports.Export {
					if e == nil {
						return nil, errors.NewBadRequest(fmt.Sprintf("export of %q is not supported", scope.Resource.Resource))
					}
					return e.Export(ctx, name, exports)
				}
				if err := metainternalversion.ParameterCodec.DecodeParameters(values, scope.MetaGroupVersion, &options); err != nil {
					err = errors.NewBadRequest(err.Error())
					return nil, err
				}
			}
			if trace != nil {
				trace.Step("About to Get from storage")
			}
			return r.Get(ctx, name, &options)
		})
}

// GetResourceWithOptions returns a function that handles retrieving a single resource from a rest.Storage object.
func GetResourceWithOptions(r rest.GetterWithOptions, scope *handlers.RequestScope, isSubresource bool) http.HandlerFunc {
	return getResourceHandler(scope,
		func(ctx context.Context, name string, req *http.Request, trace *utiltrace.Trace) (runtime.Object, error) {
			opts, subpath, subpathKey := r.NewGetOptions()
			trace.Step("About to process Get options")
			if err := getRequestOptions(req, scope, opts, subpath, subpathKey, isSubresource); err != nil {
				err = errors.NewBadRequest(err.Error())
				return nil, err
			}
			if trace != nil {
				trace.Step("About to Get from storage")
			}
			return r.Get(ctx, name, opts)
		})
}

// getRequestOptions parses out options and can include path information.  The path information shouldn't include the subresource.
func getRequestOptions(req *http.Request, scope *handlers.RequestScope, into runtime.Object, subpath bool, subpathKey string, isSubresource bool) error {
	if into == nil {
		return nil
	}

	query := req.URL.Query()
	if subpath {
		newQuery := make(url.Values)
		for k, v := range query {
			newQuery[k] = v
		}

		ctx := req.Context()
		requestInfo, _ := request.RequestInfoFrom(ctx)
		startingIndex := 2
		if isSubresource {
			startingIndex = 3
		}

		p := strings.Join(requestInfo.Parts[startingIndex:], "/")

		// ensure non-empty subpaths correctly reflect a leading slash
		if len(p) > 0 && !strings.HasPrefix(p, "/") {
			p = "/" + p
		}

		// ensure subpaths correctly reflect the presence of a trailing slash on the original request
		if strings.HasSuffix(requestInfo.Path, "/") && !strings.HasSuffix(p, "/") {
			p += "/"
		}

		newQuery[subpathKey] = []string{p}
		query = newQuery
	}
	return scope.ParameterCodec.DecodeParameters(query, scope.Kind.GroupVersion(), into)
}

func ListResource(r list.Lister, rw rest.Watcher, scope *handlers.RequestScope, forceWatch bool, minRequestTimeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// For performance tracking purposes.
		trace := utiltrace.New("List", utiltrace.Field{"url", req.URL.Path})

		namespace, err := scope.Namer.Namespace(req)
		if err != nil {
			errResponse(scope, err, w, req)
			return
		}

		// Watches for single objects are routed to this function.
		// Treat a name parameter the same as a field selector entry.
		hasName := true
		_, name, err := scope.Namer.Name(req)
		if err != nil {
			hasName = false
		}

		ctx := req.Context()
		ctx = request.WithNamespace(ctx, namespace)

		outputMediaType, _, err := negotiation.NegotiateOutputMediaType(req, scope.Serializer, scope)
		if err != nil {
			errResponse(scope, err, w, req)
			return
		}

		opts := metainternalversion.ListOptions{}
		if err := metainternalversion.ParameterCodec.DecodeParameters(req.URL.Query(), scope.MetaGroupVersion, &opts); err != nil {
			err = errors.NewBadRequest(err.Error())
			errResponse(scope, err, w, req)
			return
		}

		// transform fields
		// TODO: DecodeParametersInto should do this.
		if opts.FieldSelector != nil {
			fn := func(label, value string) (newLabel, newValue string, err error) {
				return scope.Convertor.ConvertFieldLabel(scope.Kind, label, value)
			}
			if opts.FieldSelector, err = opts.FieldSelector.Transform(fn); err != nil {
				// TODO: allow bad request to set field causes based on query parameters
				err = errors.NewBadRequest(err.Error())
				errResponse(scope, err, w, req)
				return
			}
		}

		if hasName {
			// metadata.name is the canonical internal name.
			// SelectionPredicate will notice that this is a request for
			// a single object and optimize the storage query accordingly.
			nameSelector := fields.OneTermEqualSelector("metadata.name", name)

			// Note that fieldSelector setting explicitly the "metadata.name"
			// will result in reaching this branch (as the value of that field
			// is propagated to requestInfo as the name parameter.
			// That said, the allowed field selectors in this branch are:
			// nil, fields.Everything and field selector matching metadata.name
			// for our name.
			if opts.FieldSelector != nil && !opts.FieldSelector.Empty() {
				selectedName, ok := opts.FieldSelector.RequiresExactMatch("metadata.name")
				if !ok || name != selectedName {
					errResponse(scope, errors.NewBadRequest("fieldSelector metadata.name doesn't match requested name"), w, req)
					return
				}
			} else {
				opts.FieldSelector = nameSelector
			}
		}

		//if opts.Watch || forceWatch {
		//	if rw == nil {
		//		errResponse(scope, errors.NewMethodNotSupported(scope.Resource.GroupResource(), "watch"), w, req)
		//		return
		//	}
		//	// TODO: Currently we explicitly ignore ?timeout= and use only ?timeoutSeconds=.
		//	timeout := time.Duration(0)
		//	if opts.TimeoutSeconds != nil {
		//		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
		//	}
		//	if timeout == 0 && minRequestTimeout > 0 {
		//		timeout = time.Duration(float64(minRequestTimeout) * (rand.Float64() + 1.0))
		//	}
		//	klog.V(3).Infof("Starting watch for %s, rv=%s labels=%s fields=%s timeout=%s", req.URL.Path, opts.ResourceVersion, opts.LabelSelector, opts.FieldSelector, timeout)
		//	ctx, cancel := context.WithTimeout(ctx, timeout)
		//	defer cancel()
		//	watcher, err := rw.Watch(ctx, &opts)
		//	if err != nil {
		//		errResponse(scope, err, w, req)
		//		return
		//	}
		//	requestInfo, _ := request.RequestInfoFrom(ctx)
		//	metrics.RecordLongRunning(req, requestInfo, metrics.APIServerComponent, func() {
		//		serveWatch(watcher, scope, outputMediaType, req, w, timeout)
		//	})
		//	return
		//}

		// Log only long List requests (ignore Watch).
		defer trace.LogIfLong(500 * time.Millisecond)
		trace.Step("About to List from storage")
		o := list.ListOptions{Size: 10}
		if err := list.ParameterCodec.DecodeParameters(req.URL.Query(), scope.Kind.GroupVersion(), &o); err != nil {
			err = errors.NewBadRequest(err.Error())
			errResponse(scope, err, w, req)
			return
		}
		result, err := r.List(ctx, &opts, &o)
		if err != nil {
			errResponse(scope, err, w, req)
			return
		}
		trace.Step("Listing from storage done")

		transformResponseObject(ctx, scope, trace, req, w, http.StatusOK, outputMediaType, result)
		trace.Step("Writing http response done", utiltrace.Field{"count", meta.LenList(result)})
	}
}

func errResponse(scope *handlers.RequestScope, err error, w http.ResponseWriter, req *http.Request) {
	responsewriters.ErrorNegotiated(err, scope.Serializer, scope.Kind.GroupVersion(), w, req)
}

// setObjectSelfLink sets the self link of an object as needed.
// TODO: remove the need for the namer LinkSetters by requiring objects implement either Object or List
//   interfaces
func setObjectSelfLink(ctx context.Context, obj runtime.Object, req *http.Request, namer handlers.ScopeNamer) error {
	// We only generate list links on objects that implement ListInterface - historically we duck typed this
	// check via reflection, but as we move away from reflection we require that you not only carry Items but
	// ListMeta into order to be identified as a list.
	if !meta.IsListType(obj) {
		requestInfo, ok := request.RequestInfoFrom(ctx)
		if !ok {
			return fmt.Errorf("missing requestInfo")
		}
		return setSelfLink(obj, requestInfo, namer)
	}

	uri, err := namer.GenerateListLink(req)
	if err != nil {
		return err
	}
	if err := namer.SetSelfLink(obj, uri); err != nil {
		klog.V(4).Infof("Unable to set self link on object: %v", err)
	}
	requestInfo, ok := request.RequestInfoFrom(ctx)
	if !ok {
		return fmt.Errorf("missing requestInfo")
	}

	count := 0
	err = meta.EachListItem(obj, func(obj runtime.Object) error {
		count++
		return setSelfLink(obj, requestInfo, namer)
	})

	if count == 0 {
		if err := meta.SetList(obj, []runtime.Object{}); err != nil {
			return err
		}
	}

	return err
}

// setSelfLink sets the self link of an object (or the child items in a list) to the base URL of the request
// plus the path and query generated by the provided linkFunc
func setSelfLink(obj runtime.Object, requestInfo *request.RequestInfo, namer handlers.ScopeNamer) error {
	// TODO: SelfLink generation should return a full URL?
	uri, err := namer.GenerateLink(requestInfo, obj)
	if err != nil {
		return nil
	}

	return namer.SetSelfLink(obj, uri)
}
