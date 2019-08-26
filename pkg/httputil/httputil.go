package httputil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
)

const (
	k8sCAFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

// DeferClose captures and prints the error from closing (if an error occurs).
// This is designed to be used in a defer statement.
func DeferClose(c io.Closer) {
	if err := c.Close(); err != nil {
		glog.Error(err)
	}
}

// ReadErrorBody in the error case ready the body message.
// But return it as an error (or return an error from reading the body).
func ReadErrorBody(body io.Reader) (err error) {
	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	return fmt.Errorf(string(bodyBytes))
}

// GetBodyOK returns the body or an error if the response is not okay
func GetBodyOK(httpClient *http.Client, apiURL string) ([]byte, error) {
	res, err := httpClient.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer DeferClose(res.Body)
	if res.StatusCode >= 400 {
		errMsg := fmt.Errorf("Error response %v URL %s", res.StatusCode, apiURL)
		return nil, errMsg
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, err
}

func ReadCACerts() (*x509.CertPool, error) {
	// try to load system CA certs
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	// load k8s CA cert
	caCert, err := ioutil.ReadFile(k8sCAFile)
	if err != nil {
		return nil, fmt.Errorf("fail to read CA file %s, error: %v", k8sCAFile, err)
	}
	if ok := rootCAs.AppendCertsFromPEM(caCert); !ok {
		glog.Warningf("fail to append CA file to pool, using system CAs only")
	}
	return rootCAs, nil
}

func LoadCerts(cert []byte, key []byte) (*x509.CertPool, tls.Certificate, error) {
	if cert == nil || key == nil {
		return nil, tls.Certificate{}, fmt.Errorf("fail to load certs, cert and key can not be empty")
	}

	rootCAs, err := ReadCACerts()
	if err != nil {
		return rootCAs, tls.Certificate{}, err
	}

	// load client cert
	tlsCert, err := tls.X509KeyPair(cert, key)
	return rootCAs, tlsCert, err
}
