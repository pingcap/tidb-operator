# Testing Workload Tool

A tool for testing TiDB cluster connections and performance with TLS support.

## Features

- **ping**: Test database connection
- **workload**: Execute database load testing
- **import**: Batch import test data

## TLS Support

The tool now supports multiple ways to configure TLS connections:

### 1. Configuration via File Paths

```bash
./testing-workload -action=ping -host=localhost -enable-tls \
    -tls-ca=/path/to/ca.crt \
    -tls-cert=/path/to/client.crt \
    -tls-key=/path/to/client.key
```

### 2. Configuration via Mount Path (Recommended for Kubernetes)

```bash
./testing-workload -action=ping -host=localhost -enable-tls \
    -tls-mount-path=/var/lib/tls-certs
```

The mount path should contain the following files:
- `ca.crt`: CA certificate (optional)
- `tls.crt`: Client certificate (optional)
- `tls.key`: Client private key (optional)

### 3. Configuration via Environment Variables

```bash
export TLS_CA_CERT="-----BEGIN CERTIFICATE-----..."
export TLS_CLIENT_CERT="-----BEGIN CERTIFICATE-----..."
export TLS_CLIENT_KEY="-----BEGIN PRIVATE KEY-----..."

./testing-workload -action=ping -host=localhost -enable-tls -tls-from-env
```

## Parameters

### Basic Parameters
- `-action`: Action to execute (ping, workload, import)
- `-host`: TiDB server address
- `-user`: Database username (default: root)
- `-password`: Database password

### TLS Parameters
- `-enable-tls`: Enable TLS connection
- `-tls-cert`: TLS client certificate file path
- `-tls-key`: TLS client private key file path
- `-tls-ca`: TLS CA certificate file path
- `-tls-mount-path`: TLS certificate mount directory path (Kubernetes environment)
- `-tls-from-env`: Load TLS certificates from environment variables
- `-tls-insecure-skip-verify`: Skip TLS certificate verification (for testing only)

### Workload Parameters
- `-duration`: Load test duration in minutes (default: 10)
- `-max-connections`: Maximum number of connections (default: 30)
- `-sleep-interval`: Operation interval in seconds (default: 1)
- `-long-txn-sleep`: Long transaction sleep time in seconds (default: 10)
- `-max-lifetime`: Maximum connection lifetime in seconds (default: 60)

### Import Parameters
- `-batch-size`: Batch import size (default: 1000)
- `-total-rows`: Total rows to import (default: 500000)
- `-import-table`: Import table name (default: t1)
- `-split-region-count`: Number of regions (default: 0)

## Usage Examples

### Basic Connection Test
```bash
./testing-workload -action=ping -host=tidb-server
```

### Connection Test with TLS
```bash
./testing-workload -action=ping -host=tidb-server -enable-tls \
    -tls-mount-path=/var/lib/tidb-tls
```

### Load Testing
```bash
./testing-workload -action=workload -host=tidb-server \
    -duration=5 -max-connections=10 -enable-tls \
    -tls-mount-path=/var/lib/tidb-tls
```

### Data Import
```bash
./testing-workload -action=import -host=tidb-server \
    -total-rows=100000 -batch-size=500 \
    -enable-tls -tls-mount-path=/var/lib/tidb-tls
```

## Usage in Kubernetes Environment

In Kubernetes environments, TLS certificates are typically mounted via Secrets:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: testing-workload
spec:
  containers:
  - name: testing-workload
    image: testing-workload:latest
    command:
    - ./testing-workload
    - -action=ping
    - -host=tidb-service
    - -enable-tls
    - -tls-mount-path=/var/lib/tidb-tls
    volumeMounts:
    - name: tls-certs
      mountPath: /var/lib/tidb-tls
      readOnly: true
  volumes:
  - name: tls-certs
    secret:
      secretName: tidb-client-tls
```
