package monitor

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	corelisterv1 "k8s.io/client-go/listers/core/v1"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
)

// Store is a store that fetches and caches TLS materials, bearer tokens
// and auth credentials from configmaps and secrets.
// Data can be referenced directly from a TiDBMonitor object.
// In practice a new store is created and used by
// each reconciliation loop.
//
// Store doesn't support concurrent access.
type Store struct {
	objStore cache.Store
	cmClient corelisterv1.ConfigMapLister
	sClient  corelisterv1.SecretLister

	TLSAssets map[TLSAssetKey]TLSAsset
}

// NewStore returns an empty assetStore.
func NewStore(cmClient corelisterv1.ConfigMapLister, sClient corelisterv1.SecretLister) *Store {
	return &Store{
		cmClient:  cmClient,
		sClient:   sClient,
		TLSAssets: make(map[TLSAssetKey]TLSAsset),
		objStore:  cache.NewStore(assetKeyFunc),
	}
}

func assetKeyFunc(obj interface{}) (string, error) {
	switch v := obj.(type) {
	case *v1.ConfigMap:
		return fmt.Sprintf("0/%s/%s", v.GetNamespace(), v.GetName()), nil
	case *v1.Secret:
		return fmt.Sprintf("1/%s/%s", v.GetNamespace(), v.GetName()), nil
	}
	return "", errors.Errorf("unsupported type: %T", obj)
}

// AddSafeTLSConfig validates the given SafeTLSConfig and adds it to the store.
func (s *Store) AddSafeTLSConfig(ctx context.Context, ns string, tlsConfig *v1alpha1.SafeTLSConfig) error {
	if tlsConfig == nil {
		return nil
	}

	err := tlsConfig.Validate()
	if err != nil {
		return errors.Wrap(err, "failed to validate TLS configuration")
	}

	return s.addTLSAssets(ctx, ns, *tlsConfig)
}

// addTLSAssets processes the given SafeTLSConfig and adds the referenced CA, certificate and key to the store.
func (s *Store) addTLSAssets(ctx context.Context, ns string, tlsConfig v1alpha1.SafeTLSConfig) error {
	var (
		err  error
		ca   string
		cert string
		key  string
	)

	ca, err = s.GetKey(ctx, ns, tlsConfig.CA)
	if err != nil {
		return errors.Wrap(err, "failed to get CA")
	}

	cert, err = s.GetKey(ctx, ns, tlsConfig.Cert)
	if err != nil {
		return errors.Wrap(err, "failed to get cert")
	}

	if tlsConfig.KeySecret != nil {
		key, err = s.GetSecretKey(ctx, ns, *tlsConfig.KeySecret)
		if err != nil {
			return errors.Wrap(err, "failed to get key")
		}
	}

	if ca != "" {
		block, _ := pem.Decode([]byte(ca))
		if block == nil {
			return errors.New("failed to decode CA certificate")
		}
		_, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			return errors.Wrap(err, "failed to parse CA certificate")
		}
		s.TLSAssets[TLSAssetKeyFromSelector(ns, tlsConfig.CA)] = TLSAsset(ca)
	}

	if cert != "" && key != "" {
		_, err = tls.X509KeyPair([]byte(cert), []byte(key))
		if err != nil {
			return errors.Wrap(err, "failed to load X509 key pair")
		}
		s.TLSAssets[TLSAssetKeyFromSelector(ns, tlsConfig.Cert)] = TLSAsset(cert)
		s.TLSAssets[TLSAssetKeyFromSelector(ns, v1alpha1.SecretOrConfigMap{Secret: tlsConfig.KeySecret})] = TLSAsset(key)
	}

	return nil
}

// GetKey processes the given SecretOrConfigMap selector and returns the referenced data.
func (s *Store) GetKey(ctx context.Context, namespace string, sel v1alpha1.SecretOrConfigMap) (string, error) {
	switch {
	case sel.Secret != nil:
		return s.GetSecretKey(ctx, namespace, *sel.Secret)
	case sel.ConfigMap != nil:
		return s.GetConfigMapKey(ctx, namespace, *sel.ConfigMap)
	default:
		return "", nil
	}
}

// GetConfigMapKey processes the given ConfigMapKeySelector and returns the referenced data.
func (s *Store) GetConfigMapKey(ctx context.Context, namespace string, sel v1.ConfigMapKeySelector) (string, error) {
	obj, exists, err := s.objStore.Get(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sel.Name,
			Namespace: namespace,
		},
	})
	if err != nil {
		return "", errors.Wrapf(err, "unexpected store error when getting configmap %q", sel.Name)
	}

	if !exists {
		cm, err := s.cmClient.ConfigMaps(namespace).Get(sel.Name)
		if err != nil {
			return "", errors.Wrapf(err, "unable to get configmap %q", sel.Name)
		}
		if err = s.objStore.Add(cm); err != nil {
			return "", errors.Wrapf(err, "unexpected store error when adding configmap %q", sel.Name)
		}
		obj = cm
	}

	cm := obj.(*v1.ConfigMap)
	if _, found := cm.Data[sel.Key]; !found {
		return "", errors.Errorf("key %q in configmap %q not found", sel.Key, sel.Name)
	}

	return cm.Data[sel.Key], nil
}

// GetSecretKey processes the given SecretKeySelector and returns the referenced data.
func (s *Store) GetSecretKey(ctx context.Context, namespace string, sel v1.SecretKeySelector) (string, error) {
	obj, exists, err := s.objStore.Get(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sel.Name,
			Namespace: namespace,
		},
	})
	if err != nil {
		return "", errors.Wrapf(err, "unexpected store error when getting secret %q", sel.Name)
	}

	if !exists {
		secret, err := s.sClient.Secrets(namespace).Get(sel.Name)
		if err != nil {
			return "", errors.Wrapf(err, "unable to get secret %q", sel.Name)
		}
		if err = s.objStore.Add(secret); err != nil {
			return "", errors.Wrapf(err, "unexpected store error when adding secret %q", sel.Name)
		}
		obj = secret
	}

	secret := obj.(*v1.Secret)
	if _, found := secret.Data[sel.Key]; !found {
		return "", errors.Errorf("key %q in secret %q not found", sel.Key, sel.Name)
	}

	return string(secret.Data[sel.Key]), nil
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
