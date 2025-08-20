#!/bin/bash

ds() {
    docker stop "$1" && docker rm $_
}

# Delete previous iterations
kind delete cluster
ds collector
ds jaeger
ds registry

# Create a basic OTEL setup and a local registry (avoid rate limiting due to intensive testing)
docker network create kind

set -e

docker run -d --restart=always -p 16686:16686 --name jaeger --network kind jaegertracing/jaeger:2.8.0

docker run -d --restart=always -p 4317:4317 -v ./otel-collector.yaml:/otel-local-config.yaml --name collector --network kind \
    otel/opentelemetry-collector:0.54.0 --config=/otel-local-config.yaml

docker run -d --restart=always -p 5000:5000 --name registry --network kind registry:3

# Run Kind
# Configuration for local registry from https://github.com/ctfer-io/chall-manager/blob/main/.github/workflows/e2e.yaml
# Configuration for tracing from https://github.com/kyverno/kyverno/blob/main/scripts/config/kind/tracing.yaml
cat <<EOF > kind-config.yaml
apiVersion: kind.x-k8s.io/v1alpha4
kind: Cluster
containerdConfigPatches:
# Local host registry
# TODO migrate to this https://kind.sigs.k8s.io/docs/user/local-registry/
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:5000"]
    endpoint = ["http://registry:5000"]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."10.96.219.72:5000"]
    endpoint = ["http://10.96.219.72:5000"]

kubeadmConfigPatches:
- |-
  kind: ClusterConfiguration
  apiServer:
    extraVolumes:
      - name: tracing-configuration
        hostPath: /etc/kube-tracing/apiserver-tracing.yaml
        mountPath: /etc/kube-tracing/apiserver-tracing.yaml
        readOnly: true
        pathType: File
    extraArgs:
      service-node-port-range: "30000-30010"
      tracing-config-file: /etc/kube-tracing/apiserver-tracing.yaml

- |-
  kind: KubeletConfiguration
  featureGates:
    KubeletTracing: true
  tracing:
    endpoint: collector:4317
    samplingRatePerMillion: 1000000

nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30000
    hostPort: 30000
  - containerPort: 30001
    hostPort: 30001
  - containerPort: 30002
    hostPort: 30002
  - containerPort: 30003
    hostPort: 30003
  - containerPort: 30004
    hostPort: 30004
  - containerPort: 30005
    hostPort: 30005
  - containerPort: 30006
    hostPort: 30006
  - containerPort: 30007
    hostPort: 30007
  - containerPort: 30008
    hostPort: 30008
  - containerPort: 30009
    hostPort: 30009
  - containerPort: 30010
    hostPort: 30010
  extraMounts:
  - hostPath: ./apiserver-tracing.yaml
    containerPath: /etc/kube-tracing/apiserver-tracing.yaml
    readOnly: true
networking:
  disableDefaultCNI: true
EOF
kind create cluster --config=kind-config.yaml
rm kind-config.yaml

# Enable storageclass "standard" RWO/RWX
# From https://github.com/kubernetes-sigs/kind/issues/1487#issuecomment-2211072952
kubectl -n local-path-storage patch configmap local-path-config -p '{"data": {"config.json": "{\n\"sharedFileSystemPath\": \"/var/local-path-provisioner\"\n}"}}'

# Install Cilium as CNI
helm repo add cilium https://helm.cilium.io/
docker pull quay.io/cilium/cilium:v1.17.4
kind load docker-image quay.io/cilium/cilium:v1.17.4

helm install cilium cilium/cilium --version 1.17.4 \
    --namespace kube-system \
    --set image.pullPolicy=IfNotPresent \
    --set ipam.mode=kubernetes
