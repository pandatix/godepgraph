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

URL="localhost:$(cd deploy && pulumi stack output godepgraph-port)"

go build -o bin/godepgraph-cli cmd/godepgraph-cli/main.go

echo ""
echo "=== Creating CDN ==="
./bin/godepgraph-cli --url $URL cdn create \
  --name github.com/ctfer-io/chall-manager \
  --version v0.5.1

echo ""
echo "=== Creating RDG ==="
# Simulate a state management server, close to what we would see in reality
docker run -d --name rdg \
  --network kind \
  -v $(pwd)/example/extract:/data \
  python:3 \
  python3 -m http.server 8000 --directory /data --bind 0.0.0.0
IP=$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' rdg)

sleep 5 # make sure the Python server is up & running

./bin/godepgraph-cli --url $URL rdg create \
  --uri "http://${IP}:8000/state.json"

docker network disconnect kind rdg
docker stop rdg && docker rm $_ # don't need it anymore

echo ""
echo "=== Creating SIG ==="
./bin/godepgraph-cli --url $URL sig create \
  --file example/extract/otel_traces
