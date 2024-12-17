// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tlsutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestLoadTlsConfigFromSecret(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			corev1.ServiceAccountRootCAKey: []byte(
				`-----BEGIN CERTIFICATE-----
MIIC6jCCAdKgAwIBAgIQWCZFlAx6Qqg5VypjCRRDEjANBgkqhkiG9w0BAQsFADAP
MQ0wCwYDVQQDEwRUaURCMB4XDTI0MDIyOTA4NDk0MloXDTM0MDIyNjA4NDk0Mlow
DzENMAsGA1UEAxMEVGlEQjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
ALuJ5+ef6hcALcVGzSUjlKX5CixXbr5To8zImZivjr6IzQVEMeyOLt0xb3SuXL49
nnz/N9s3ET06Sc1Zf3QhyJqxy9wT7vVq352tmZtZwFreR+UfHirh1xoTIJJ7e6ai
SYH2UHi5bUFTamFOqo5/QsueNcK4lQO8WxMgyPBmaUfPfn5b0uuD2gdNlA6yEcCt
7Cr84xeHnpmoevtf7Obk3fv+S5nGahl0AvV0ilXZI0BN6u15fZz8c49JA9rrD6x0
XjdndZPIT3Brz5sRZoZlmdAj9DBXqo1CJ7Z4uezWmaF3+Su/TT/FuoZwnX2GUIxc
VqkYh1QiWSpYI66RQGle1IcCAwEAAaNCMEAwDgYDVR0PAQH/BAQDAgKkMA8GA1Ud
EwEB/wQFMAMBAf8wHQYDVR0OBBYEFNiNgN9eiqNZgxWiUKWuW/yTN+XDMA0GCSqG
SIb3DQEBCwUAA4IBAQB0t7MXGI1bwFBNYb1VbpdsI1KdVUT2S3oKKoeq2wMfwIJd
DYeiU4banRjSSgrbsSd38WiY9G249yLViTWC+Q6p4X8jaVZO86b/rA7fx0wD4NRG
K6JusFhZPKI7rHYDQulpbvIASZ1epJmTAiNz99z3hL3DF9EnDipO9TTEqYsempJu
cSWegdrRAWFpU821DF9DWpnN/4QSWAHsdFnpt+F8h1YK2A0tnLyb5zKYMnYLErrX
FkfMxRTUHY1Rz0fHvb4rSjMObpOf7rEjYyHL1KJIPPFXmy5bDBx8ReHxCscMPSQW
zINRz3PfryIPIyjvVEJ6vvA+Eb4hke2YCZmhrWMk
-----END CERTIFICATE-----`),
			corev1.TLSCertKey: []byte(
				`-----BEGIN CERTIFICATE-----
MIIDATCCAemgAwIBAgIRAKXw/bNMzl1mQR1aqeHogOcwDQYJKoZIhvcNAQELBQAw
DzENMAsGA1UEAxMEVGlEQjAeFw0yNDAzMDEwNjExNTlaFw0yNTAzMDEwNjExNTla
MCExEDAOBgNVBAoTB1BpbmdDQVAxDTALBgNVBAMTBFRpREIwggEiMA0GCSqGSIb3
DQEBAQUAA4IBDwAwggEKAoIBAQCkhF/P9qaCdIsF82CLR6ij5UPjjJC8b/n1RdYD
DLsyByKjEe+T1gTfgPWaKU5vykfSyqxj1sHDBwXUnBbHGQDv4blId0HOaxPXll8w
H7F67rR506R1RsYzPzNjcV8bfRlEDkYn/dCtNek5UHXVWwdhwL6+fieC8VUQmovl
jq309imQsHZ3kYcnqRoihPsSpIZdc55C6R7b16XBQ19wNvAa8dPx2aB1855xREZT
xZrn2O+d9bEt6GKc5J883YIzi8VlZobS1BzKvYVS893BpNGAK5A8tQq2rXjOOEAb
/s/lEIs7U3sQbnCjVcf6ZYKjTvoySlvikpVs34IzruAWqfgBAgMBAAGjRjBEMBMG
A1UdJQQMMAoGCCsGAQUFBwMCMAwGA1UdEwEB/wQCMAAwHwYDVR0jBBgwFoAU2I2A
316Ko1mDFaJQpa5b/JM35cMwDQYJKoZIhvcNAQELBQADggEBAIjRT4b39XV9mczX
TuTBaXBSfNO5drAYDcjAC6U3/r1OSSmpVGKf/ZSG/MWXkgEtWQHzq0A6ggel6HYX
r2I4I/YXC9H6yXbPph9tC7mSPvMF0/4zMjapPSwZzp83TMdNNaVqXrhaEC7s8LgA
ebUVFsrlgCHAAyBQ+AHIf9AmxWEtl7LDdwuvXh96e5Uv/ZagxkOO58nOuoFH6lFy
64ijiZGM9pIL3Bc3v5M5VhPRzXeoeRQI600KeXko6yIlj3i1Jr6DvnNgNsbqVVu2
Oc+TcFgc1vWQo4HfEFOeGHgPKiOuwX+F8nEtog4b4Fo9G5LSIq9mKruaDXPVxeh6
mrVBKbU=
-----END CERTIFICATE-----`),
			corev1.TLSPrivateKeyKey: []byte(
				`-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEApIRfz/amgnSLBfNgi0eoo+VD44yQvG/59UXWAwy7MgcioxHv
k9YE34D1milOb8pH0sqsY9bBwwcF1JwWxxkA7+G5SHdBzmsT15ZfMB+xeu60edOk
dUbGMz8zY3FfG30ZRA5GJ/3QrTXpOVB11VsHYcC+vn4ngvFVEJqL5Y6t9PYpkLB2
d5GHJ6kaIoT7EqSGXXOeQuke29elwUNfcDbwGvHT8dmgdfOecURGU8Wa59jvnfWx
LehinOSfPN2CM4vFZWaG0tQcyr2FUvPdwaTRgCuQPLUKtq14zjhAG/7P5RCLO1N7
EG5wo1XH+mWCo076Mkpb4pKVbN+CM67gFqn4AQIDAQABAoIBAGOMVg2YyhiWPKlV
I04kBj9mMzY1kD714uIvZ9hgk8Up3COgbr+d+UTk27h01il+1QcP7FBdWtGQJk8I
RCAlWRPOGjdnMkKdOFxzeRW9l78zQbGWByWPtc68p3O83jfb8rXjjUAVrXeh74Xm
0eZQNp9H6iOKYo4xSa/KVGyLcWePs/NxATpwbL2MDWdr2aJjiZwma70gvHKZz3WQ
LWpW9vcw+cl6+D+p0lOZzJFL9s/iamJyH1TjAmyIhN+S7szlC8T2srmjFb+Y38I6
0FBld1farVwDp+par4pGtRq+KslUc3yN6XNma+I25B8C2l4+DksnjCeqrm0bm7EY
JKtyVaECgYEAxNzk0xBjrKNW6/RXMDql+ukFcMISljD1IJFktdfVia5deFlzV3eQ
Nq+IrG7zuNIiSmegjJqEunoGLkT9NwO0y48f1QDYj2FBXlXOWEiLG1pitL8X53ns
MiXulfneemiE6RNg8MvM4Q6Mu6Lsbijgjqsj/saIWXdwCszjHiytK/cCgYEA1fAI
PQWqECL3eWdKCO5tFsrvIkYJwTJBYJNJfrbftpZ2mr2AQD+0bA9Ty5tuGyBykFEO
Hua6UfHxYrbBDzqpAiOFFS5wlPyGVJISWFC6oubRPeGXHx5fFpimoMa/J9YZ7v78
DjXcmEDYgCQybAILzVWIpZ97WOF81ksJy+jozccCgYBiVXR3eVhQg8aHViW3EZSX
II53JHnkS9Al1HpZ2tXvUAmgdA4JQs/mgQfkGgfj6hL214x6rzRdcVZlBlD1igRl
KbjczO9fr1TXqkTIFHRn1V44qrtmBKDW69uhTo6y1kKNqgBiR2qvgHULxPYUkJaa
rSHtwX2aMu7kdjN8fxSBQQKBgQClQlsK0FJTTr9+P4SYK52HKtHY1uN4ItsPwBbY
1GkxwT7zP4lPmCZGBv0C3hkKyWDWDFbtFew9mriNOYEew4CEj22hNBNxczRNJd0X
7ZyOc+CUfavgNPTdHqQws/Y7zo6P6NZKH988mXLkYZG1j0sQnY8F6ZE90kk9vA9g
PZWARwKBgCMmUWaz2KdZS8z3DzxY0E0rP+WIiVXPDJr8h/i2wh9XrQhbvY7OAYXU
RwYYRMKBT/YKLf2phDeHgd8D67p9EQqCD6mMl+TG5BWxiA4CjrHWdhNY7XvjybO8
nyCmZvl1qIEV8/2BE4zUWZqizbhrattyt10YvnJCBt6YuFB6Jcqg
-----END RSA PRIVATE KEY-----`),
		},
	}

	tlsConfig, err := LoadTLSConfigFromSecret(secret)
	require.NoError(t, err)
	assert.NotNil(t, tlsConfig)
	assert.NotNil(t, tlsConfig.RootCAs)
	assert.NotNil(t, tlsConfig.ClientCAs)
	assert.Len(t, tlsConfig.Certificates, 1)
}
