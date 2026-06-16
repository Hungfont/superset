#!/usr/bin/env bash
# ===================================================================
# STEP 7: MetalLB — Production Load Balancer
#
# Cung cấp IP cho Service type LoadBalancer trong bare-metal / on-prem.
#
# Senior DevOps:
#  - L2 mode: đơn giản nhất, dùng ARP announcement
#  - BGP mode: cần router hỗ trợ BGP, phức tạp hơn nhưng failover nhanh hơn
#  - Strict ARP mode (bắt buộc) — cấu hình kube-proxy
#  - IP pool: khai báo range IP cụ thể, tránh conflict với DHCP
#  - Memberlist secret: encryption cho giao tiếp giữa các MetalLB speakers
# ===================================================================
set -euo pipefail

# ===================================================================
# 0. CẤU HÌNH
# ===================================================================

NAMESPACE="metallb-system"

# IP Pool — Senior: PHẢI trùng với dải mạng LAN của bạn
# Các IP này phải:
#   1. Không bị DHCP cấp phát
#   2. Không trùng với IP tĩnh của node
#   3. Cùng subnet với node network (192.168.1.0/24)
#   4. Có thể ping được từ client

# === SỬA DẢI IP NÀY CHO ĐÚNG VỚI MẠNG CỦA BẠN ===
LB_IP_RANGE="192.168.1.200-192.168.1.210"
# === THƯỜNG DÙNG: 192.168.1.200-192.168.1.210 (10 IPs cho services) ===

# Mode: "l2" (L2/ARP) hoặc "bgp"
METALLB_MODE="l2"

# BGP config (chỉ khi METALLB_MODE=bgp)
BGP_ROUTER_ID="10.0.0.1"            # Router ID (thường là IP của router)
BGP_PEER_ADDR="192.168.1.1"         # IP của BGP router
BGP_ASN_LOCAL="64512"               # Local ASN
BGP_ASN_PEER="64512"                # Peer ASN (same if iBGP)
BGP_ANNOUNCE_PREFIXES="${LB_IP_RANGE%%-*}"

# ===================================================================
# 1. KUBE-PROXY STRICT ARP
# ===================================================================
#
# Senior DevOps: BẮT BUỘC — nếu không, MetalLB không hoạt động.

configure_strict_arp() {
  echo ""
  echo "================================================================"
  echo "  PHASE 1: Configure kube-proxy strict ARP"
  echo "================================================================"

  echo ">>> [1.1] Enable strict ARP mode in kube-proxy..."

  # Sửa config map kube-proxy
  # Current value should be false -> change to true
  local current_arp
  current_arp=$(kubectl get configmap kube-proxy -n kube-system -o jsonpath='{.data.config\.yaml}' 2>/dev/null | grep -oP 'strictARP: \K\w+' || echo "false")

  if [ "$current_arp" = "false" ]; then
    kubectl get configmap kube-proxy -n kube-system -o yaml | \
      sed 's/strictARP: false/strictARP: true/' | \
      kubectl apply -f - 2>/dev/null || {
        # Fallback: patch trực tiếp
        kubectl patch configmap kube-proxy -n kube-system --type='json' \
          -p='[{"op": "replace", "path": "/data/config.yaml", "value": "'"$(kubectl get configmap kube-proxy -n kube-system -o jsonpath='{.data.config\.yaml}' | sed 's/strictARP: false/strictARP: true/')"'"}]'
      }
    echo "   ✅ kube-proxy strictARP: true"
  else
    echo "   ✅ kube-proxy strictARP already true"
  fi

  # Restart kube-proxy để load config mới
  echo ">>> [1.2] Restart kube-proxy pods..."
  kubectl -n kube-system delete pods -l k8s-app=kube-proxy --force --grace-period=0 2>/dev/null || true
  sleep 3
  kubectl -n kube-system wait --for=condition=Ready pods -l k8s-app=kube-proxy --timeout=60s 2>/dev/null || true

  echo "   ✅ kube-proxy restarted with strict ARP"
  echo ""
}

# ===================================================================
# 2. DEPLOY METALLB
# ===================================================================

deploy_metallb() {
  echo ""
  echo "================================================================"
  echo "  PHASE 2: Deploy MetalLB via Helm"
  echo "================================================================"

  # Add repo & update
  helm repo add metallb https://metallb.github.io/metallb 2>/dev/null || true
  helm repo update metallb 2>/dev/null || true

  # Create namespace
  kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

  # Generate memberlist secret (encryption)
  echo ">>> [2.1] Generating memberlist secret..."
  kubectl create secret generic metallb-memberlist \
    --namespace "$NAMESPACE" \
    --from-literal=secretkey="$(openssl rand -base64 128)" \
    --dry-run=client -o yaml | kubectl apply -f -

  echo ""
  echo ">>> [2.2] Installing MetalLB via Helm..."
  helm upgrade --install metallb metallb/metallb \
    --namespace "$NAMESPACE" \
    --set controller.replicas=2 \
    --set speaker.tolerations='[{"key":"node-role.kubernetes.io/control-plane","operator":"Exists","effect":"NoSchedule"}]' \
    --set speaker.nodeSelector.'topology\.kubernetes\.io/node-type'='worker' \
    --set prometheus.enabled=false \
    --atomic \
    --timeout 5m \
    --wait \
    --debug 2>&1 | tail -15

  echo ""
  echo ">>> [2.3] Waiting for MetalLB pods..."
  kubectl -n "$NAMESPACE" rollout status deployment/metallb-controller --timeout=120s 2>/dev/null || true
  kubectl -n "$NAMESPACE" rollout status daemonset/metallb-speaker --timeout=120s 2>/dev/null || true

  echo ""
  echo ">>> [2.4] MetalLB pods:"
  kubectl -n "$NAMESPACE" get pods -o wide
  echo ""
}

# ===================================================================
# 3. CONFIGURE IP POOL & L2 ADVERTISEMENT
# ===================================================================
#
# Senior DevOps: MetalLB dùng CRD (IPPool và L2Advertisement/BGPAdvertisement)
# từ version 0.14+. Không dùng ConfigMap cũ.

configure_pool() {
  echo ""
  echo "================================================================"
  echo "  PHASE 3: Configure IP Pool"
  echo "================================================================"

  # Parse IP range
  local range_start="${LB_IP_RANGE%%-*}"
  local range_end="${LB_IP_RANGE##*-}"

  echo ">>> [3.1] Creating IPPool (${LB_IP_RANGE})..."

  cat <<EOF | kubectl apply -f -
apiVersion: metallb.io/v1beta1
kind: IPPool
metadata:
  name: production-pool
  namespace: ${NAMESPACE}
spec:
  addresses:
  - ${LB_IP_RANGE}
  autoAssign: true
  avoidBuggyIPs: true
  protocol:
    layer2: {}
EOF

  echo "   ✅ IPPool created: ${LB_IP_RANGE}"

  if [ "$METALLB_MODE" = "l2" ]; then
    echo ""
    echo ">>> [3.2] Creating L2Advertisement..."
    cat <<EOF | kubectl apply -f -
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: l2-advertisement
  namespace: ${NAMESPACE}
spec:
  ipAddressPools:
  - production-pool
  # Senior: nếu muốn chỉ announce từ một số node cụ thể:
  # nodeSelectors:
  # - matchLabels:
  #     metallb.universe.tf/announce: "true"
EOF
    echo "   ✅ L2 Advertisement configured"
  fi

  if [ "$METALLB_MODE" = "bgp" ]; then
    echo ""
    echo ">>> [3.2] Creating BGP configuration..."
    cat <<EOF | kubectl apply -f -
apiVersion: metallb.io/v1beta2
kind: BGPPeer
metadata:
  name: upstream-router
  namespace: ${NAMESPACE}
spec:
  peerAddress: ${BGP_PEER_ADDR}
  peerASN: ${BGP_ASN_PEER}
  myASN: ${BGP_ASN_LOCAL}
  routerID: ${BGP_ROUTER_ID}
  peerPort: 179
  holdTime: 180s
---
apiVersion: metallb.io/v1beta1
kind: BGPAdvertisement
metadata:
  name: bgp-advertisement
  namespace: ${NAMESPACE}
spec:
  ipAddressPools:
  - production-pool
  peers:
  - upstream-router
  aggregationLength: 32
EOF
    echo "   ✅ BGP configured — peering with ${BGP_PEER_ADDR}"
  fi

  echo ""
}

# ===================================================================
# 4. VERIFY
# ===================================================================

verify() {
  echo ""
  echo "================================================================"
  echo "  PHASE 4: Verification"
  echo "================================================================"

  echo ">>> [4.1] MetalLB resources:"
  kubectl -n "$NAMESPACE" get ippools,l2advertisements 2>/dev/null || \
    kubectl -n "$NAMESPACE" get ippools,bgppeers,bgpadvertisements 2>/dev/null || true

  echo ""
  echo ">>> [4.2] Test LoadBalancer service..."
  # Tạo test service
  kubectl create deployment metallb-test --image=nginx --port=80 --dry-run=client -o yaml | kubectl apply -f -

  # Chờ deployment ready
  kubectl rollout status deployment/metallb-test --timeout=60s 2>/dev/null || true

  # Expose as LoadBalancer
  kubectl expose deployment metallb-test \
    --name=metallb-test-svc \
    --port=80 \
    --target-port=80 \
    --type=LoadBalancer \
    --dry-run=client -o yaml | kubectl apply -f -

  echo ""
  echo "   Waiting for LoadBalancer IP (might take 10-20s)..."
  local external_ip=""
  for i in $(seq 1 20); do
    external_ip=$(kubectl get svc metallb-test-svc -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
    if [ -n "$external_ip" ] && [ "$external_ip" != "null" ]; then
      break
    fi
    sleep 2
  done

  if [ -n "$external_ip" ] && [ "$external_ip" != "null" ]; then
    echo "   ✅ LoadBalancer IP: $external_ip"
    echo ""
    echo "   Testing HTTP response..."
    if curl -sI --connect-timeout 5 "http://${external_ip}" 2>/dev/null | head -5; then
      echo ""
      echo "   ✅ MetalLB working! Nginx responds on $external_ip"
    else
      echo "   ⚠️  HTTP test failed (nginx might not be ready yet)"
      echo "   Try: curl http://${external_ip}"
    fi
  else
    echo "   ⚠️  LoadBalancer IP still pending — check metallb-controller logs:"
    echo "   kubectl -n metallb-system logs deployment/metallb-controller"
  fi

  # Cleanup test resources
  echo ""
  echo ">>> [4.3] Cleaning up test resources..."
  kubectl delete svc metallb-test-svc --ignore-not-found=true
  kubectl delete deployment metallb-test --ignore-not-found=true

  echo ""
  echo "   ✅ Verification complete"
}

# ===================================================================
# MAIN
# ===================================================================

main() {
  configure_strict_arp
  deploy_metallb
  configure_pool
  verify

  echo ""
  echo "================================================================"
  echo "  ✅ STEP 7 COMPLETE — MetalLB ready"
  echo "================================================================"
  echo ""
  echo "📌 IP Pool: ${LB_IP_RANGE}"
  echo "   Các service type LoadBalancer sẽ nhận IP từ pool này."
  echo ""
  echo "📌 Ingress-Nginx (STEP 6) sẽ tự động nhận IP sau khi MetalLB ready."
  echo ""
  echo "📌 Kiểm tra lại:"
  echo "    kubectl -n ingress-nginx get svc"
  echo "    # EXTERNAL-IP sẽ hiển thị IP từ pool"
  echo ""
  echo "📌 Nếu cần thêm IP pool:"
  echo "    kubectl -n metallb-system edit ippool production-pool"
  echo ""
}

main
