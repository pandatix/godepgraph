#!/bin/bash

# Prepare Pulumi configuration
export PULUMI_CONFIG_PASSPHRASE=""
pulumi login --local

cd ..

# Delete previous iterations
docker stop neo4j && docker rm $_
docker network disconnect kind rdg
docker stop rdg && docker rm $_
(cd deploy && pulumi cancel --stack tmp --yes && pulumi stack rm tmp --force --yes)

# Then load all Docker images we are deploying
export REGISTRY="localhost:5000"
drp() {
  docker pull "$1" && docker tag $_ "${REGISTRY}/$_" && docker push $_
}
drp library/neo4j:5.22.0 &

REGISTRY="${REGISTRY}/" TAG=dev make docker &

wait

# Deploy GoDepGraph as a service, in a hardened infrastructure
(
  cd deploy
  pulumi stack init tmp
  pulumi config set name godepgraph
  pulumi config set registry $REGISTRY
  pulumi config set tag dev
  pulumi config set swagger true # for testing things with the API
  pulumi config set expose-godepgraph true # for testing things with the API
  pulumi config set expose-neo4j true # to use the processed data directly
  pulumi up -y
)

go build -o bin/godepgraph-cli cmd/godepgraph-cli/main.go

echo ""
echo "=== Creating CDN ==="
./analysis-cdn.sh

echo ""
echo "=== Creating RDG ==="
./analysis-rdg.sh

echo ""
echo "=== Creating SIG ==="
./analysis-sig.sh
