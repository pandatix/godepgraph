#!/bin/bash

# Prepare Pulumi configuration
export PULUMI_CONFIG_PASSPHRASE=""
pulumi login --local

# Delete previous iterations
(cd simulation && pulumi cancel --yes && pulumi stack rm tmp --force --yes)

# Then load all Docker images we are deploying
export REGISTRY="localhost:5000"
drp() {
  docker pull "$1" && docker tag $_ "${REGISTRY}/$_" && docker push $_
}
drp prom/prometheus:v3.4.2 &
drp otel/opentelemetry-collector-contrib:0.129.1 &
drp jaegertracing/jaeger:2.8.0 &
drp bitnami/etcd:3.5.16-debian-12-r0 &
drp ctferio/ctfd:3.7.7-0.5.0 &
drp bitnami/redis:7.4.3-debian-12-r0 &
drp bitnami/mariadb:11.4.5-debian-12-r12 &
drp library/busybox:1.28 &
drp library/busybox:1.37.0 &
drp library/registry:3 &

# TODO use CM v0.5.1 ASAP
cd ~/Documents/ctfer.io/chall-manager
REGISTRY="${REGISTRY}/" TAG=v0.5.1 make docker &

wait

# Then deploy the simulated infrastructure
(
  cd simulation
  pulumi stack init tmp
  pulumi config set registry localhost:5000
  pulumi up -y
)

export OTEL_EXPORTER_OTLP_ENDPOINT="dns://localhost:$(pulumi stack output monitoring.nodeport)"
export OTEL_EXPORTER_OTLP_INSECURE="true"

# Configure the challenges
(
  ./bin/ctfops challenges \
  --verbose \
  --one-shot \
  --tracing \
  --service-name tmp-ctfops \
  --dir nbc24/ \
  --url "$(cd simulation && pulumi stack output url)" \
  --oci.insecure \
  --oci.address "localhost:$(cd simulation && pulumi stack output registry.nodeport)" \
  --oci.distant "$(kubectl get svc registry -n fullchain -o jsonpath='{.spec.clusterIP}'):5000"
)

mkdir extract

# Extract the simulation infrastructure Pulumi state (for the RDG)
(
  cd simulation
  pulumi stack export --file extract/state.json
)

# Extract OpenTelemetry Collector signals (traces, metrics, logs ; for the SIG)
go install github.com/ctfer-io/monitoring/cmd/extractor@v0.1.0
extractor \
  --namespace "$(cd simulation && pulumi stack output monitoring.namespace)" \
  --pvc-name "$(cd simulation && pulumi stack output monitoring.otel-cold-extract-pvc-name)" \
  --directory extract/
