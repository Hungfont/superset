#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# Helper: Verify cluster health
#
# Chạy TRÊN MASTER
# ============================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "============================================"
echo "  🩺 CLUSTER HEALTH CHECK"
echo "============================================"
echo ""

# 1. Nodes
echo ">>> [1] Nodes:"
NODES=$(kubectl get nodes --no-headers 2>/dev/null | wc -l)
kubectl get nodes -o wide
NOT_READY=$(kubectl get nodes --no-headers | grep -v " Ready" | wc -l)
if [ "$NOT_READY" -eq 0 ]; then
  echo -e " ${GREEN}✅ All $NODES nodes are Ready${NC}"
else
  echo -e " ${RED}❌ $NOT_READY node(s) not Ready${NC}"
fi

echo ""

# 2. Pod health (all namespaces)
echo ">>> [2] Pod health (all ns):"
kubectl get pods -A --no-headers 2>/dev/null | awk '{print $4}' | sort | uniq -c | sort -rn || true
POD_TOTAL=$(kubectl get pods -A --no-headers 2>/dev/null | wc -l)
POD_FAIL=$(kubectl get pods -A --no-headers 2>/dev/null | grep -vE "Running|Completed" | wc -l)
if [ "$POD_FAIL" -eq 0 ]; then
  echo -e " ${GREEN}✅ All $POD_TOTAL pods healthy${NC}"
else
  echo -e " ${YELLOW}⚠️  $POD_FAIL pods không ở trạng thái Running/Completed${NC}"
  kubectl get pods -A | grep -vE "Running|Completed"
fi

echo ""

# 3. Component status
echo ">>> [3] Component status:"
if kubectl get componentstatus 2>/dev/null | head -5; then
  echo ""
fi

# 4. Calico specific
echo ">>> [4] Calico:"
CALICO_PODS=$(kubectl -n kube-system get pods -l k8s-app=calico-node --no-headers 2>/dev/null | wc -l)
if [ "$CALICO_PODS" -gt 0 ]; then
  echo -e " ${GREEN}✅ Calico running on $CALICO_PODS nodes${NC}"
  kubectl -n kube-system get pods -l k8s-app=calico-node -o wide
else
  echo -e " ${RED}❌ Calico chưa được cài${NC}"
fi

echo ""

# 5. CNI config
echo ">>> [5] CNI config:"
ls -la /etc/cni/net.d/ 2>/dev/null || echo " (no CNI config found)"

echo ""

# 6. DNS test
echo ">>> [6] DNS test:"
kubectl run -it --rm dns-test --image=registry.k8s.io/e2e-test-images/jessie-dnsutils:1.0 \
  --restart=Never -- nslookup kubernetes.default.svc.cluster.local 2>/dev/null && \
  echo -e " ${GREEN}✅ DNS resolution OK${NC}" || \
  echo -e " ${YELLOW}⚠️  DNS test skipped (cần network cho image pull)${NC}"

echo ""
echo "============================================"
echo "  ✅ VERIFICATION COMPLETE"
echo "============================================"
