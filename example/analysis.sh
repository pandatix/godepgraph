#!/bin/bash

# Prepare Pulumi configuration
export PULUMI_CONFIG_PASSPHRASE=""
pulumi login --local

# Delete previous iterations
docker stop neo4j && docker rm $_
(cd ../deploy && pulumi cancel --yes && pulumi stack rm tmp --force --yes)

# Then load all Docker images we are deploying
export REGISTRY="localhost:5000"
drp() {
  docker pull "$1" && docker tag $_ "${REGISTRY}/$_" && docker push $_
}
drp library/neo4j:5.22.0 &

REGISTRY="${REGISTRY}/" TAG=dev make docker &

wait

# Deploy GoDepGraph as a service, in a hardened infrastructure
cd deploy
pulumi stack init tmp
pulumi config set name godepgraph
pulumi config set registry $REGISTRY
pulumi config set tag dev
pulumi config set swagger true # for testing things with the API
pulumi config set expose-godepgraph true # for testing things with the API
pulumi config set expose-neo4j true # to use the processed data directly
pulumi up -y
cd ..

URL="localhost:$(cd deploy && pulumi stack output godepgraph-port)"

go build -o bin/godepgraph-cli cmd/godepgraph-cli/main.go

echo "=== Creating CDN ==="
time ./bin/godepgraph-cli --url $URL cdn create \
  --name github.com/ctfer-io/chall-manager \
  --version v0.5.1

# TODO @lucas export experiment stack info
# echo "=== Creating RDG ==="

echo "=== Creating SIG ==="
time ./bin/godepgraph-cli --url $URL sig create \
  --file extract/otel_traces
