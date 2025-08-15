#!/bin/bash

# Prepare Pulumi configuration
export PULUMI_CONFIG_PASSPHRASE=""
pulumi login --local

# Delete previous iterations
(cd simulation && pulumi cancel --stack tmp --yes && pulumi stack rm tmp --force --yes)
rm -rf extract && mkdir extract

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
drp ctferio/chall-manager:v0.5.1 &
drp ctferio/chall-manager-janitor:v0.5.1 &
drp bitnami/redis:7.4.3-debian-12-r0 &
drp bitnami/mariadb:11.4.5-debian-12-r12 &
drp library/busybox:1.28 &
drp library/busybox:1.37.0 &
drp library/registry:3 &

wait

# Then deploy the simulated infrastructure
(
  cd simulation
  pulumi stack init tmp
  pulumi config set registry localhost:5000
  pulumi up -y

  # ... and extract the state (for the RDG)
  pulumi stack export --file ../extract/state.json
)

export OTEL_EXPORTER_OTLP_ENDPOINT="dns://localhost:$(cd simulation && pulumi stack output monitoring.nodeport)"
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

# Extract OpenTelemetry Collector signals (traces, metrics, logs ; for the SIG)
go install github.com/ctfer-io/monitoring/cmd/extractor@v0.1.0
extractor \
  --namespace "$(cd simulation && pulumi stack output monitoring.namespace)" \
  --pvc-name "$(cd simulation && pulumi stack output monitoring.otel-cold-extract-pvc-name)" \
  --directory extract/
