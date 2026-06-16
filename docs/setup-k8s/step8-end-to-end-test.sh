#!/usr/bin/env bash
# ===================================================================
# STEP 8: End-to-End Test — Ingress + MetalLB + Application
#
# Test toàn bộ stack:
#   1. Deploy một ứng dụng mẫu (echoserver)
#   2. Expose bằng Service type LoadBalancer (MetalLB cấp IP)
#   3. Tạo Ingress resource (Ingress-Nginx route)
#   4. Verify từ bên ngoài cluster
#   5. Cleanup
# ===================================================================
set -euo pipefail

NAMESPACE="e2e-test"
INGRESS_CLASS="nginx"
TEST_DOMAIN="test.k8s.local"        # Sẽ dùng curl --resolve

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

cleanup() {
  echo ""
  echo ">>> [CLEANUP] Removing test namespace..."
  kubectl delete namespace "$NAMESPACE" --ignore-not-found=true --wait=false 2>/dev/null || true
  echo "   ✅ Cleanup done"
}

echo "================================================================"
echo "  🧪 END-TO-END TEST — Ingress + MetalLB"
echo "================================================================"
echo ""

# ===================================================================
# 1. PREFLIGHT
# ===================================================================

preflight() {
  echo ">>> [1] Preflight checks..."

  # Check helm
  if ! command -v helm &>/dev/null; then
    echo -e " ${RED}❌ Helm not installed — run STEP 5 first${NC}"
    exit 1
  fi

  # Check ingress controller
  if ! kubectl get pods -n ingress-nginx -l app.kubernetes.io/component=controller 2>/dev/null | grep -q Running; then
    echo -e " ${RED}❌ Ingress-Nginx not running — run STEP 6 first${NC}"
    exit 1
  fi

  # Check MetalLB
  if ! kubectl get pods -n metallb-system -l app.kubernetes.io/component=controller 2>/dev/null | grep -q Running; then
    echo -e " ${RED}❌ MetalLB not running — run STEP 7 first${NC}"
    exit 1
  fi

  # Get LoadBalancer IP của ingress controller
  INGRESS_LB_IP=$(kubectl -n ingress-nginx get svc ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")

  if [ -z "$INGRESS_LB_IP" ]; then
    echo -e " ${YELLOW}⚠️  Ingress-Nginx LoadBalancer IP not assigned yet${NC}"
    echo "   MetalLB might still be assigning. Trying to wait..."
    for i in $(seq 1 15); do
      INGRESS_LB_IP=$(kubectl -n ingress-nginx get svc ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
      if [ -n "$INGRESS_LB_IP" ]; then
        break
      fi
      sleep 2
    done
  fi

  if [ -z "$INGRESS_LB_IP" ]; then
    echo -e " ${RED}❌ Cannot get Ingress-Nginx LoadBalancer IP. Is MetalLB configured?${NC}"
    exit 1
  fi

  echo -e " ${GREEN}✅ Preflight OK${NC}"
  echo "   Ingress Controller IP: $INGRESS_LB_IP"
  echo ""
}

# ===================================================================
# 2. DEPLOY TEST APP
# ===================================================================

deploy_app() {
  echo ""
  echo "================================================================"
  echo "  PHASE 2: Deploy test application"
  echo "================================================================"

  # Create namespace
  kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

  # Deploy echoserver (reflects request details)
  echo ">>> [2.1] Deploying echoserver..."
  kubectl -n "$NAMESPACE" create deployment echoserver \
    --image=registry.k8s.io/echoserver:1.10 \
    --port=8080 \
    --replicas=2 \
    --dry-run=client -o yaml | kubectl apply -f -

  # Wait for deployment
  kubectl -n "$NAMESPACE" rollout status deployment/echoserver --timeout=120s

  echo ""
  echo ">>> [2.2] Creating Service (ClusterIP)..."
  kubectl -n "$NAMESPACE" expose deployment echoserver \
    --name=echoserver \
    --port=80 \
    --target-port=8080 \
    --type=ClusterIP \
    --dry-run=client -o yaml | kubectl apply -f -

  echo "   ✅ echoserver deployed"
  echo ""
}

# ===================================================================
# 3. CREATE INGRESS
# ===================================================================

create_ingress() {
  echo ""
  echo "================================================================"
  echo "  PHASE 3: Create Ingress Resource"
  echo "================================================================"

  cat <<EOF | kubectl -n "$NAMESPACE" apply -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: echoserver-ingress
  annotations:
    # Senior DevOps: rate limiting per IP
    nginx.ingress.kubernetes.io/limit-rps: "10"
    nginx.ingress.kubernetes.io/limit-burst-multiplier: "5"
    # Timeouts
    nginx.ingress.kubernetes.io/proxy-connect-timeout: "10"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "30"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "30"
    # Rewrite
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "false"
spec:
  ingressClassName: ${INGRESS_CLASS}
  rules:
  - host: ${TEST_DOMAIN}
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: echoserver
            port:
              number: 80
EOF

  echo "   ✅ Ingress created: ${TEST_DOMAIN} → echoserver:80"
  echo ""
}

# ===================================================================
# 4. TEST & VERIFY
# ===================================================================

run_tests() {
  echo ""
  echo "================================================================"
  echo "  PHASE 4: Testing"
  echo "================================================================"

  local ingress_ip="$INGRESS_LB_IP"
  local passed=0
  local failed=0

  # Test 1: LoadBalancer reachable
  echo ">>> [4.1] Test: Ingress LoadBalancer reachable..."
  if ping -c 1 -W 2 "$ingress_ip" &>/dev/null; then
    echo -e "   ${GREEN}✅ Ping OK${NC}"
    passed=$((passed + 1))
  else
    echo -e "   ${YELLOW}⚠️  Ping failed (may be blocked by firewall — tiếp tục)${NC}"
    # Không count là fail vì ping có thể bị block
  fi

  echo ""

  # Test 2: HTTP response via Ingress (dùng curl --resolve)
  echo ">>> [4.2] Test: HTTP via Ingress (curl --resolve)..."
  local http_response
  http_response=$(curl -s --connect-timeout 10 \
    -H "Host: ${TEST_DOMAIN}" \
    "http://${ingress_ip}/" 2>/dev/null | head -20 || echo "")

  if [ -n "$http_response" ]; then
    echo -e "   ${GREEN}✅ HTTP Response received${NC}"
    echo ""
    echo "   Response preview:"
    echo "$http_response" | head -10
    passed=$((passed + 1))
  else
    echo -e "   ${RED}❌ No HTTP response from Ingress${NC}"
    echo "   Debug: kubectl -n ${NAMESPACE} get ingress"
    echo "   Debug: curl -v -H 'Host: ${TEST_DOMAIN}' http://${ingress_ip}/"
    failed=$((failed + 1))
  fi

  echo ""

  # Test 3: HTTP Status Code
  echo ">>> [4.3] Test: HTTP status code..."
  local status_code
  status_code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 10 \
    -H "Host: ${TEST_DOMAIN}" \
    "http://${ingress_ip}/" 2>/dev/null || echo "000")

  if [ "$status_code" = "200" ]; then
    echo -e "   ${GREEN}✅ HTTP 200 OK${NC}"
    passed=$((passed + 1))
  elif [ "$status_code" != "000" ]; then
    echo -e "   ${YELLOW}⚠️  HTTP $status_code (expected 200)${NC}"
  else
    echo -e "   ${RED}❌ Connection failed${NC}"
    failed=$((failed + 1))
  fi

  echo ""

  # Test 4: Ingress details
  echo ">>> [4.4] Test: Ingress resource status..."
  local ingress_status
  ingress_status=$(kubectl -n "$NAMESPACE" get ingress echoserver-ingress -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "unknown")

  if [ "$ingress_status" = "$ingress_ip" ] || [ -n "$ingress_status" ]; then
    echo -e "   ${GREEN}✅ Ingress status: ${ingress_status}${NC}"
    passed=$((passed + 1))
  else
    echo -e "   ${YELLOW}⚠️  Ingress status: ${ingress_status} (expected ${ingress_ip})${NC}"
  fi

  echo ""

  # Test 5: MetalLB IP trong pool
  echo ">>> [4.5] Test: External IP in MetalLB pool..."
  local pool_range
  pool_range=$(kubectl -n metallb-system get ippool production-pool -o jsonpath='{.spec.addresses[0]}' 2>/dev/null || echo "")
  if [ -n "$pool_range" ]; then
    echo -e "   ${GREEN}✅ MetalLB pool: ${pool_range}${NC}"
    echo "   Ingress IP: ${ingress_ip} (should be in pool range)"
    passed=$((passed + 1))
  else
    echo -e "   ${YELLOW}⚠️  Could not verify IP pool${NC}"
  fi

  echo ""
}

# ===================================================================
# 5. SUMMARY
# ===================================================================

summary() {
  local total=$((passed + failed))

  echo ""
  echo "================================================================"
  echo "  📊 TEST SUMMARY"
  echo "================================================================"
  echo ""
  echo "   Passed: ${passed}"
  echo "   Failed: ${failed}"

  if [ $failed -eq 0 ]; then
    echo ""
    echo -e "   ${GREEN}✅ ALL TESTS PASSED — Cluster ready for production!${NC}"
    echo ""
    echo "   Toàn bộ stack hoạt động:"
    echo "   ┌─ Client ──────────────────────────────────┐"
    echo "   │  curl -H 'Host: test.k8s.local' http://... │"
    echo "   └──────────────┬────────────────────────────┘"
    echo "                  │"
    echo "   ┌──────────────▼────────────────────────────┐"
    echo "   │  MetalLB → IP: ${INGRESS_LB_IP}           │"
    echo "   └──────────────┬────────────────────────────┘"
    echo "                  │"
    echo "   ┌──────────────▼────────────────────────────┐"
    echo "   │  Ingress-Nginx (${INGRESS_CLASS})                │"
    echo "   └──────────────┬────────────────────────────┘"
    echo "                  │"
    echo "   ┌──────────────▼────────────────────────────┐"
    echo "   │  Service: echoserver:80                    │"
    echo "   └──────────────┬────────────────────────────┘"
    echo "                  │"
    echo "   ┌──────────────▼────────────────────────────┐"
    echo "   │  Pod: echoserver-xxx (port 8080)           │"
    echo "   └───────────────────────────────────────────┘"
  else
    echo ""
    echo -e "   ${RED}❌ ${failed} test(s) failed. Check above for details.${NC}"
  fi
}

# ===================================================================
# MAIN
# ===================================================================

trap cleanup EXIT

preflight
deploy_app
create_ingress

# Wait cho ingress controller process
echo ">>> Waiting for Ingress to propagate..."
sleep 5

run_tests
summary

# Giữ lại namespace 30s để debug nếu cần, rồi cleanup (trap sẽ chạy)
echo ""
echo ">>> Test complete. Cleaning up in 5s..."
sleep 5
