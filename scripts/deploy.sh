#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="$PWD/.tools/bin:$PATH"
export KUBECONFIG="$PWD/.tools/kubeconfig"
image_ref="${1:?Pass the image reference}"
cluster_name="${CLUSTER_NAME:-assignment-${GITHUB_RUN_ID:-local}-${GITHUB_RUN_ATTEMPT:-1}}"
local_port="${LOCAL_PORT:-18080}"
forward_pid=""
mkdir -p reports
cleanup() {
  result=$?
  set +e
  kubectl -n assignment get pods,deployments,services,endpointslices -o wide > reports/kubernetes.txt 2>&1
  kubectl -n assignment describe pods > reports/pod-details.txt 2>&1
  kubectl -n assignment logs -l app=assignment --all-containers=true --tail=100 > reports/application.log 2>&1
  kubectl -n assignment get events --sort-by=.metadata.creationTimestamp > reports/events.txt 2>&1
  if [[ -n "$forward_pid" ]]; then kill "$forward_pid" 2>/dev/null; wait "$forward_pid" 2>/dev/null; fi
  kind delete cluster --name "$cluster_name" 2>&1 | tee reports/cleanup.txt
  if kind get clusters 2>/dev/null | grep -Fxq "$cluster_name"; then
    echo 'Cluster cleanup failed' | tee -a reports/cleanup.txt
    result=1
  else
    echo 'Verified: assignment cluster is absent' | tee -a reports/cleanup.txt
  fi
  exit "$result"
}
trap cleanup EXIT
kind create cluster --name "$cluster_name" --wait 180s       --image kindest/node:v1.37.0@sha256:a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5       2>&1 | tee reports/cluster-create.txt
kind load docker-image "$image_ref" --name "$cluster_name"
python3 - "$image_ref" <<'PY'
from pathlib import Path
import sys
Path('.tools/deployment.yaml').write_text(Path('k8s/deployment.yaml').read_text().replace('ASSIGNMENT_IMAGE',sys.argv[1]))
PY
kubectl apply -f k8s/namespace.yaml
kubectl apply -f .tools/deployment.yaml -f k8s/service.yaml
kubectl -n assignment rollout status deployment/assignment --timeout=180s
kubectl -n assignment get pods -o wide
kubectl -n assignment port-forward service/assignment "$local_port:80" --address=127.0.0.1 > reports/port-forward.txt 2>&1 &
forward_pid=$!
python3 scripts/smoke.py "http://127.0.0.1:$local_port" | tee reports/smoke.txt
kubectl -n assignment get deployment/assignment -o json > reports/deployment.json
kubectl -n assignment get pods -o json > reports/pods.json
echo 'PASS: Kubernetes rollout and application smoke checks completed'
