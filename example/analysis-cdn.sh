#!/bin/bash

export PULUMI_CONFIG_PASSPHRASE=""
URL="localhost:$(cd deploy && pulumi stack output godepgraph-port)"

./bin/godepgraph-cli --url $URL cdn create \
  --name github.com/ctfer-io/chall-manager \
  --version v0.5.1
