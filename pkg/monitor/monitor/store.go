package monitor

import "k8s.io/client-go/tools/cache"

// Store is a store that fetches and caches TLS materials, bearer tokens
// and auth credentials from configmaps and secrets.
// Data can be referenced directly from a Prometheus object or indirectly (for
// instance via ServiceMonitor). In practice a new store is created and used by
// each reconciliation loop.
//
// Store doesn't support concurrent access.
type Store struct {
	objStore cache.Store

	TLSAssets         map[TLSAssetKey]TLSAsset
	BearerTokenAssets map[string]BearerToken
	BasicAuthAssets   map[string]BasicAuthCredentials
}

// TLSAssetKey is a key for a TLS asset.
type TLSAssetKey struct {
	from string
	ns   string
	name string
	key  string
}

// TLSAsset represents any TLS related opaque string, e.g. CA files, client
// certificates.
type TLSAsset string

// BearerToken represents a bearer token, see
// https://tools.ietf.org/html/rfc6750.
type BearerToken string

// BasicAuthCredentials represents a username password pair to be used with
// basic http authentication, see https://tools.ietf.org/html/rfc7617.
type BasicAuthCredentials struct {
	Username string
	Password string
}
