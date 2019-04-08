package apimachinery

import (
	"crypto/x509"
	"io/ioutil"
	"os"

	"github.com/golang/glog"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
	"k8s.io/kubernetes/test/utils"
)

type certContext struct {
	cert        []byte
	key         []byte
	signingCert []byte
}

// Setup the server cert. For example, user apiservers and admission webhooks
// can use the cert to prove their identify to the kube-apiserver
func setupServerCert(namespaceName, serviceName string) *certContext {
	certDir, err := ioutil.TempDir("", "test-e2e-server-cert")
	if err != nil {
		framework.Failf("Failed to create a temp dir for cert generation %v", err)
	}
	defer os.RemoveAll(certDir)
	signingKey, err := utils.NewPrivateKey()
	if err != nil {
		framework.Failf("Failed to create CA private key %v", err)
	}
	signingCert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: "e2e-server-cert-ca"}, signingKey)
	if err != nil {
		framework.Failf("Failed to create CA cert for apiserver %v", err)
	}
	caCertFile, err := ioutil.TempFile(certDir, "ca.crt")
	if err != nil {
		framework.Failf("Failed to create a temp file for ca cert generation %v", err)
	}
	if err := ioutil.WriteFile(caCertFile.Name(), utils.EncodeCertPEM(signingCert), 0644); err != nil {
		framework.Failf("Failed to write CA cert %v", err)
	}
	key, err := utils.NewPrivateKey()
	if err != nil {
		framework.Failf("Failed to create private key for %v", err)
	}
	signedCert, err := utils.NewSignedCert(
		&cert.Config{
			CommonName: serviceName + "." + namespaceName + ".svc",
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
		key, signingCert, signingKey,
	)
	if err != nil {
		framework.Failf("Failed to create cert%v", err)
	}
	certFile, err := ioutil.TempFile(certDir, "server.crt")
	if err != nil {
		framework.Failf("Failed to create a temp file for cert generation %v", err)
	}
	keyFile, err := ioutil.TempFile(certDir, "server.key")
	if err != nil {
		framework.Failf("Failed to create a temp file for key generation %v", err)
	}
	if err = ioutil.WriteFile(certFile.Name(), utils.EncodeCertPEM(signedCert), 0600); err != nil {
		framework.Failf("Failed to write cert file %v", err)
	}
	privateKeyPEM, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		framework.Failf("Failed to marshal key %v", err)
	}
	if err = ioutil.WriteFile(keyFile.Name(), privateKeyPEM, 0644); err != nil {
		framework.Failf("Failed to write key file %v", err)
	}
	return &certContext{
		cert:        utils.EncodeCertPEM(signedCert),
		key:         privateKeyPEM,
		signingCert: utils.EncodeCertPEM(signingCert),
	}
}
