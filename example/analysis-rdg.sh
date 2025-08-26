#!/bin/bash

export PULUMI_CONFIG_PASSPHRASE=""
URL="localhost:$(cd deploy && pulumi stack output godepgraph-port)"

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
