#!/bin/bash

# Delete previous iterations
kind delete cluster | true
docker stop registry && docker rm $_

# Create local registry
docker run -d --restart=always -p 5000:5000 --name registry registry:3

# Run Kind
cat <<EOF > kind-config.yaml
apiVersion: kind.x-k8s.io/v1alpha4
kind: Cluster
containerdConfigPatches:
# Local host registry
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:5000"]
    endpoint = ["http://registry:5000"]

kubeadmConfigPatches:
- |
  kind: ClusterConfiguration
  apiServer:
    extraArgs:
      "service-node-port-range": "30000-30010"

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
networking:
  disableDefaultCNI: true
EOF
kind create cluster --config=kind-config.yaml
rm kind-config.yaml
docker network connect kind registry

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
