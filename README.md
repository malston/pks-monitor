# PKS API Monitor

Monitoring tool checking if PKS API is up and expose metric to Prometheus.

## Usage

Login in UAA as admin and create a UAA Cli for this service:
```shell script
uaa get-client-credentials-token admin --format=opaque -s <Pks Uaa Management Admin Client secret>

uaa create-client <uaa-cli-id> -s <uaa-cli-secret> --authorized_grant_types client_credentials --authorities pks.clusters.manage
```

### Running locally
 
Export the following environment variables:
```shell script
export PKS_API=https://api.pks.<your-domain>
export UAA_CLI_ID=<uaa-cli-id>
export UAA_CLI_SECRET=<uaa-cli-secret>
```

Run with Go:
```shell script
go rum cmd/main.go
```

Look for `pks_api_up` metric at: localhost:8080/metrics
 
### Running with Docker
Create a `envvars` file with your pks api and uaa config:

```shell script
PKS_API=https://api.pks.<your-domain>
UAA_CLI_ID=<uaa-cli-id>
UAA_CLI_SECRET=<uaa-cli-secret>
```

Build image and run container

```shell script
docker build  -t pupimvictor/pks-monitor .

docker run -it -p 8080:8080 --env-file=envvars  pupimvictor/pks-monitor
```

Look for `pks_api_up` metric at: localhost:8080/metrics

### Running on Kubernetes
//todo