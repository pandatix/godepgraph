#!/bin/bash

export PULUMI_CONFIG_PASSPHRASE=""
URL="localhost:$(cd deploy && pulumi stack output godepgraph-port)"

./bin/godepgraph-cli --url $URL sig create \
  --file example/extract/otel_traces
