# ServiceAccount Delete Webhook

This project implements a Kubernetes admission webhook to prevent deletion of ServiceAccounts that are still in use by Pods.

## Features
- Validates ServiceAccount deletion requests.
- Denies deletion if any Pod in the namespace is using the ServiceAccount.
- Graceful startup and shutdown.
- Minimal, secure Docker image using distroless.

## Build and Run

### Prerequisites
- Go 1.21+
- Docker

### Build Locally

```
go build -o webhook-server ./main.go
```

### Build Docker Image

```
docker build -t sa-delete-webhook:latest .
```

### Run Locally

```
./webhook-server --cert-file=cert.crt --key-file=key.key
```

### Run with Docker

```
docker run --rm -p 8443:8443 -v $(pwd)/cert.crt:/cert.crt -v $(pwd)/key.key:/key.key sa-delete-webhook:latest --cert-file=/cert.crt --key-file=/key.key
```

## Kubernetes Deployment
- Deploy the webhook server as a Deployment or DaemonSet.
- Expose via a Kubernetes Service.
- Register the webhook using a ValidatingWebhookConfiguration.

## Configuration
- `--cert-file`: Path to TLS certificate file (default: cert.crt)
- `--key-file`: Path to TLS key file (default: key.key)
- `SDW_CERT_FILE` and `SDW_KEY_FILE` environment variables are also supported.

## API
- Endpoint: `/validate`
- Method: POST
- Content-Type: application/json
- Accepts: AdmissionReview (v1)
- Responds: AdmissionReview (v1)

## Example AdmissionReview Request
```
{
  "apiVersion": "admission.k8s.io/v1",
  "kind": "AdmissionReview",
  "request": { ... }
}
```

## License
MIT
