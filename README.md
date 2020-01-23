# PKS API Monitor

Monitoring tool to check if PKS API is up and expose metric to Prometheus.

## Usage

Login in UAA as admin and create a UAA Cli for this service:
```shell script
uaa get-client-credentials-token admin --format=opaque -s <Pks Uaa Management Admin Client secret>

uaa create-client <uaa-cli-id> -s <uaa-cli-secret> --authorized_grant_types client_credentials --authorities pks.clusters.manage
```

### API TLS:
Fetch PKS TLS pem certificate from Credhub or Ops Manager.

### Running locally
 
Export the following environment variables:
```shell script
export PKS_API=https://api.pks.<your-domain>
export UAA_CLI_ID=<uaa-cli-id>
export UAA_CLI_SECRET=<uaa-cli-secret>
```

Place PKS TLS cert on `/etc/pks-monitor/certs/cert.pem`

Run with Go:
```shell script
go rum cmd/main.go
```

Look for `pks_api_up` metric at: localhost:8080/metrics
 
### Running with Docker
Create the following environment variables your pks api and uaa config:

```shell script
PKS_API=https://api.pks.<your-domain>
UAA_CLI_ID=<uaa-cli-id>
UAA_CLI_SECRET=<uaa-cli-secret>
```

Create a volume for TLS certificate mounting on `/etc/pks-monitor/certs/`

Build image and run container with the environment variables and a volume with the TLS cert.

```shell script
docker build  -t pks-monitor .

# example using envvars file with the environment variables
docker run -it -p 8080:8080 --env-file=envvars --mount source=myvol,target=/etc/pks-monitor/certs/ pks-monitor
```

Look for `pks_api_up` metric at: localhost:8080/metrics

### Running on Kubernetes

Build image with `./build-image.sh`

Create a `Secret` named `pks-api-monitor` with the api and uaa configs:
```shell script
kubectl create secret generic pks-api-monitor \
--from-literal=pks-api=$PKS_API \
--from-literal=uaa-cli-id=$UAA_CLI_ID \
--from-literal=uaa-cli-secret=$UAA_CLI_SECRET
```

Create a `Secret` named `pks-api-cert` with PKS TLS cert:
```shell script
kubectl create secret generic pks-api-cert --from-file=cert.pem
```

Modify `deployment.yaml` changing the container image to match the image built and pushed with `./build-image.sh`

Apply the deployment: `kubectl apply -f deployment.yaml`