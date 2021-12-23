# Admission-webhook Refactor

## Summary

This document presents a design to refactor admission webhook server to meet the subsequent issues of admission webhook.

## Motivation

Admission webhook server uses a third-party library named `github.com/openshift/generic-admission-server` , which is obsolete and complex.  

Admission webhook server is just a simple HTTP server, refer to: <https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#write-an-admission-webhook-server>. As feature requests related to admission webhook increase, we need to refactor admission webhook server to a simple HTTP server that is easy to understand and extend.

### Goals

To make admission webhook server implement by a simple HTTP server.

### Non-Goals

## Proposal

- Remove `github.com/openshift/generic-admission-server` in admission webhook.
- Introduce HTTP server in admission webhook.

```Go
// create webhook controller
statefulset := statefulset.NewAdmissionController()
strategy := strategy.NewAdmissionController()

// create http server
serverMux := http.NewServeMux()
serverMux.HandleFunc("/pingcapstatefulsetvalidations", statefulset.ServeValidate)
serverMux.HandleFunc("/pingcapstatefulsetmutations", statefulset.ServeMutate)
serverMux.HandleFunc("/pingcapstrategyvalidations", strategy.ServeValidate)
serverMux.HandleFunc("/pingcapstrategymutations", strategy.ServeMutate)
```

- Example of `statefulset.go`:

```Go

func NewAdmissionController() *StatefulSetAdmissionControl {
  return &StatefulSetAdmissionControl{}
}

// validate HandleFunc
func (sc *StatefulSetAdmissionControl) ServeValidate(w http.ResponseWriter, r *http.Request) {
  util.ServeAdmission(w, r, sc.validate)
}

// mutate HandleFunc
func (sc *StatefulSetAdmissionControl) ServeMutate(w http.ResponseWriter, r *http.Request) {
  util.ServeAdmission(w, r, sc.mutate)
}

// real validate logic
func (sc *StatefulSetAdmissionControl) validate(ar v1.AdmissionReview) *v1.AdmissionResponse{
    reviewResponse := &v1.AdmissionResponse{Allowed: true}
    // logic here
    return reviewResponse
}

// real validate logic
func (sc *StatefulSetAdmissionControl) mutate(ar v1.AdmissionReview) *v1.AdmissionResponse{
    reviewResponse := &v1.AdmissionResponse{Allowed: true}
    // logic here
    return reviewResponse
}

func (a *StatefulSetAdmissionControl) Initialize(cfg *rest.Config, stopCh <-chan struct{}) error{
    // Initialize
    return nil
}
```

- function `ServeAdmission(w http.ResponseWriter, r *http.Request, admit AdmitFunc)` parse `http.Request` and response result of calling AdmitFunc.

### Test Plan

- Refactor unit tests.
- Refactor e2e cases.
- Manual tests.
