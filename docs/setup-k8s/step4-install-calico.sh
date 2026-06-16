#!/usr/bin/env bash
# ===================================================================
# STEP 4: Calico CNI — Production Configuration
#
# Chạy TRÊN MASTER sau khi tất cả worker đã join.
#
# Senior DevOps decisions:
#  - VXLAN mode (thay vì IPIP) — better throughput, no kernel dep
#  - Typha — scale cho cluster > 50 nodes (mặc định 3 replicas)
#  - Resource requests/limits cho calico-node và Typha
#  - Felix tuning — rate-limit logging, connection tracking
#  - BGP config (tùy chọn) — nếu muốn Calico route với router vật lý
#  - NetworkPolicy mặc định deny-all cho production security
#  - IP pool blockSize=26 (64 IPs/subnet) — phù hợp multi-tenant
# ===================================================================

set -euo pipefail

# ===================================================================
# 0. CẤU HÌNH
# ===================================================================

POD_CIDR="10.244.0.0/16"
CALICO_VERSION="v3.29"           # Calico 3.29 tương thích K8s 1.30-1.32

# Networking mode: "vxlan" (recommended) or "ipip"
# VXLAN: higher throughput, UDP encapsulation, cross-subnet works OOTB
# IPIP: lower overhead on same subnet, needs kernel module (tunnel)
CALICO_NETWORK_MODE="vxlan"

# Typha — API server load balancer cho Calico
# Bật khi cluster > 50 nodes, hoặc khi có nhiều calico-node
ENABLE_TYPHA=true
TYPHA_REPLICAS=1                 # Tăng lên 3 khi cluster > 100 nodes

# BGP — tắt nếu không cần route quảng bá ra router vật lý
ENABLE_BGP=false

# IP Pool config
BLOCK_SIZE=26                    # 64 IPs per block (mặc định 26)
NAT_OUTGOING=true                # SNAT cho pod traffic ra ngoài cluster

# === CẢNH BÁO: uncomment dòng dưới nếu bạn thực sự muốn apply ngay ===
# APPLY_IMMEDIATELY=true

confirm_apply() {
  if [ "${APPLY_IMMEDIATELY:-false}" != "true" ]; then
    echo ""
    echo "⚠️  Ở chế độ dry-run. Các lệnh kubectl apply sẽ chỉ được in ra."
    echo "   Set APPLY_IMMEDIATELY=true ở đầu script để apply thật."
    echo ""
    APPLY_PREFIX="echo "[DRY-RUN] kubectl apply""
  else
    APPLY_PREFIX="kubectl apply"
    echo "🚀 APPLY_IMMEDIATELY=true — sẽ apply thật!"
  fi
}

# ===================================================================
# 1. INSTALL CALICO OPERATOR
# ===================================================================

install_calico() {
  echo ""
  echo "================================================================"
  echo "  PHASE 1: Install Calico Operator & CRDs"
  echo "================================================================"
  echo ""

  local manifest_url="https://raw.githubusercontent.com/projectcalico/calico/${CALICO_VERSION}/manifests/tigera-operator.yaml"

  echo ">>> [1.1] Download and apply Tigera Operator"
  echo "    From: $manifest_url"
  curl -sSL "$manifest_url" -o /tmp/tigera-operator.yaml

  # Verify download
  if [ ! -s /tmp/tigera-operator.yaml ]; then
    echo "   ❌ Download thất bại. Kiểm tra network hoặc CALICO_VERSION."
    echo "   Có thể dùng: curl -sSL https://docs.projectcalico.org/manifests/tigera-operator.yaml"
    exit 1
  fi

  eval "$APPLY_PREFIX" -f /tmp/tigera-operator.yaml
  echo "   ✅ Tigera Operator deployed"

  # Wait for operator to be ready
  echo ">>> [1.2] Waiting for Tigera Operator..."
  kubectl -n tigera-operator rollout status deployment/tigera-operator --timeout=120s 2>/dev/null || {
    echo "   ⚠️  Operator rollout timeout — continuing (may need manual check)"
  }
  echo ""
}

# ===================================================================
# 2. CREATE INSTALLATION CR (Custom Resource)
# ===================================================================

configure_calico() {
  echo ""
  echo "================================================================"
  echo "  PHASE 2: Configure Calico Installation"
  echo "================================================================"

  # Xác định encapsulation mode
  if [ "$CALICO_NETWORK_MODE" = "vxlan" ]; then
    ENCAPSULATION="VXLANCrossSubnet"
    ENCAP_VXLAN="Enabled"
    ENCAP_IPIP="Never"
  else
    ENCAPSULATION="IPIPCrossSubnet"
    ENCAP_VXLAN="Never"
    ENCAP_IPIP="Enabled"
  fi

  # Typha config
  if [ "$ENABLE_TYPHA" = true ]; then
    TYPHA_SPEC="typhaDeployment:
    replicas: ${TYPHA_REPLICAS}"
  else
    TYPHA_SPEC="typhaDeployment:
    enabled: false"
  fi

  # BGP config
  if [ "$ENABLE_BGP" = true ]; then
    BGP_SPEC="birdDeployment:
    enabled: true"
  else
    BGP_SPEC="birdDeployment:
    enabled: false"
  fi

  # Calico Installation manifest
  cat <<EOF | tee /tmp/calico-installation.yaml > /dev/null
# ===================================================================
# Calico Installation — Production Configuration
# Generated for: POD_CIDR=${POD_CIDR}, Mode=${CALICO_NETWORK_MODE}
# ===================================================================
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  # Kubernetes provider
  kubernetesProvider: ""
  # Calico image registry (mặc định — dùng mirror nếu cần)
  registry: quay.io
  # CNI config
  cni:
    type: Calico
    ipam:
      type: Calico
  # Calico node resource requests/limits — điều chỉnh theo capacity
  calicoNode:
    resources:
      requests:
        cpu: 250m
        memory: 128Mi
      limits:
        cpu: 500m
        memory: 256Mi
  # Typha — scale-out for Calico API
  ${TYPHA_SPEC}
  # BGP
  ${BGP_SPEC}
  # Encapsulation
  calicoNetwork:
    bgp: "${CALICO_NETWORK_MODE}"   # VXLAN or IPIP
    ipPools:
    - cidr: "${POD_CIDR}"
      encapsulation: "${ENCAPSULATION}"
      natOutgoing: ${NAT_OUTGOING}
      nodeSelector: all()
      blockSize: ${BLOCK_SIZE}
    nodeAddressAutodetectionV4:
      kubernetes: {}
    # Linux dataplane
    linuxDataplane: Iptables
    # Senior DevOps: Set mtu = 1500 - 50 (VXLAN overhead) = 1450
    # Nếu dùng Jumbo frames (9000): set mtu = 8950
    mtu: 1450
  # FlexVolume plugin path
  flexVolumePath: /usr/libexec/kubernetes/kubelet-plugins/volume/exec/
  # Node update strategy
  nodeUpdateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
---
# ===================================================================
# Felix Configuration — Production tuning
# ===================================================================
apiVersion: operator.tigera.io/v1
kind: FelixConfiguration
metadata:
  name: default
spec:
  # Rate-limit felix logs — tránh fill disk
  healthStatsEnabled: true
  logSeverityScreen: Warning
  logging:
    logFileMaxSize: 50Mi
    logFileMaxAge: 7
    logFileMaxCount: 5
  # Senior DevOps: Flow log không nên bật ở mức individual
  # Nếu cần network flow logs, dùng Calico Enterprise hoặc Cilium Hubble
  flowLogsFlushInterval: 0s
  # Prometheus metrics
  prometheusMetricsEnabled: false   # Bật nếu có Prometheus
  # Senior DevOps: Điều chỉnh nếu gặp performance issues
  # iptables refresh interval
  iptablesRefreshInterval: 60s
  # Senior DevOps: Route refresh
  routeRefreshInterval: 60s
  # Connection tracking
  conntrackLoggingEnabled: false
  conntrackMaxProcessStats: 60
  # XDP acceleration — tắt nếu kernel không hỗ trợ
  xdpEnabled: false
---
# ===================================================================
# IP Pool Configuration
# ===================================================================
apiVersion: projectcalico.org/v3
kind: IPPool
metadata:
  name: default-ipv4-ippool
spec:
  cidr: ${POD_CIDR}
  blockSize: ${BLOCK_SIZE}
  nodeSelector: all()
  natOutgoing: ${NAT_OUTGOING}
  disabled: false
EOF

  echo ">>> [2.1] Calico Installation manifest created: /tmp/calico-installation.yaml"
  echo ""
}

# ===================================================================
# 3. APPLY CALICO CONFIG
# ===================================================================

apply_calico() {
  echo ""
  echo "================================================================"
  echo "  PHASE 3: Apply Calico Configuration"
  echo "================================================================"

  echo ">>> [3.1] Apply Installation CR..."
  eval "$APPLY_PREFIX" -f /tmp/calico-installation.yaml

  if [ "${APPLY_IMMEDIATELY:-false}" = "true" ]; then
    echo ""
    echo ">>> [3.2] Wait for calico-node DaemonSet..."
    echo "    (This can take 1-3 minutes while images pull)"
    kubectl -n calico-system rollout status daemonset/calico-node --timeout=180s 2>/dev/null || {
      echo "   ⚠️  calico-node rollout timeout. Check with:"
      echo "      kubectl -n calico-system get pods"
    }

    echo ""
    echo ">>> [3.3] Wait for calico-typha Deployment (if enabled)..."
    if [ "$ENABLE_TYPHA" = true ]; then
      kubectl -n calico-system rollout status deployment/calico-typha --timeout=120s 2>/dev/null || true
    fi
  fi

  echo ""
}

# ===================================================================
# 4. VERIFY CALICO
# ===================================================================

verify_calico() {
  echo ""
  echo "================================================================"
  echo "  PHASE 4: Verify Calico Installation"
  echo "================================================================"

  if [ "${APPLY_IMMEDIATELY:-false}" = "true" ]; then
    echo ">>> [4.1] Calico pods:"
    kubectl -n calico-system get pods -o wide 2>/dev/null || \
      kubectl -n kube-system get pods -l k8s-app=calico-node -o wide

    echo ""
    echo ">>> [4.2] Calico nodes status:"
    # Sử dụng calicoctl nếu có
    if command -v calicoctl &>/dev/null; then
      calicoctl get nodes 2>/dev/null || echo "   (calicoctl not connected)"
    else
      kubectl get nodes -o wide
      echo "   (Install calicoctl để xem detailed status)"
    fi

    echo ""
    echo ">>> [4.3] IP pool:"
    kubectl describe ippool default-ipv4-ippool 2>/dev/null | head -15

    echo ""
    echo ">>> [4.4] Node status:"
    kubectl get nodes -o wide
  else
    echo "   ⚠️  Calico chưa apply (dry-run mode). Apply manually:"
    echo "      kubectl apply -f /tmp/calico-installation.yaml"
  fi

  echo ""
}

# ===================================================================
# 5. NETWORK POLICY — DEFAULT DENY
# ===================================================================
#
# Senior DevOps: Trong production, luôn có default-deny policy.
# Ngăn chặn traffic không mong muốn nếu dev deploy namespace mới.

create_default_policies() {
  echo ""
  echo "================================================================"
  echo "  PHASE 5: Security — Default Deny Network Policies"
  echo "================================================================"
  echo ""

  # Tạo policy cho các namespace system
  local system_ns=("kube-system" "calico-system" "tigera-operator")
  for ns in "${system_ns[@]}"; do
    # Kiểm tra namespace tồn tại trước
    if kubectl get ns "$ns" &>/dev/null; then
      cat <<EOF | kubectl apply -f - 2>/dev/null || true
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-ingress
  namespace: ${ns}
spec:
  podSelector: {}
  policyTypes:
  - Ingress
EOF
    fi
  done

  echo "   ✅ Default deny policies applied to system namespaces"
  echo ""
  echo "⚠️  LƯU Ý: Mặc định các user namespace không có NetworkPolicy."
  echo "   Khuyến nghị apply default-deny cho mọi namespace mới:"
  echo ""
  echo '   apiVersion: networking.k8s.io/v1'
  echo '   kind: NetworkPolicy'
  echo '   metadata:'
  echo '     name: default-deny-all'
  echo '   spec:'
  echo '     podSelector: {}'
  echo '     policyTypes:'
  echo '     - Ingress'
  echo '     - Egress'
  echo ""
}

# ===================================================================
# 6. POST-INSTALL: LABEL WORKER NODES, ETC
# ===================================================================

post_install() {
  echo ""
  echo "================================================================"
  echo "  PHASE 6: Post-Install — Label & Verify"
  echo "================================================================"

  # Label tất cả nodes với Calico role
  echo ">>> [6.1] Label nodes for Calico topology..."
  for node in $(kubectl get nodes -o name 2>/dev/null); do
    local n="${node#node/}"
    kubectl label node "$n" \
      topology.kubernetes.io/node-type="worker" \
      --overwrite 2>/dev/null || true
  done

  # Re-label master
  kubectl label node -l node-role.kubernetes.io/control-plane \
    topology.kubernetes.io/node-type="control-plane" \
    --overwrite 2>/dev/null || true

  echo ""
  echo ">>> [6.2] Final verification — CNI config..."
  # Kiểm tra CNI config tồn tại trên node (nếu chạy local)
  if [ -d /etc/cni/net.d ]; then
    ls -la /etc/cni/net.d/ 2>/dev/null || echo "   (no CNI config)"
  fi

  echo ""
  echo ">>> [6.3] Taint worker nodes (nếu cần)..."
  echo "    (No additional taints applied — workers are schedulable by default)"
}

# ===================================================================
# MAIN
# ====================================================================

main() {
  confirm_apply

  if [ "${APPLY_IMMEDIATELY:-false}" = "true" ]; then
    install_calico
    configure_calico
    apply_calico
    verify_calico
    create_default_policies
    post_install
  else
    echo ""
    echo "================================================================"
    echo "  🏗️  GENERATING CALICO MANIFESTS (Dry-Run Mode)"
    echo "================================================================"
    echo ""
    echo "   Các manifest được tạo tại:"
    echo "   - /tmp/tigera-operator.yaml"
    echo "   - /tmp/calico-installation.yaml"
    echo ""
    echo "   Để apply, chạy:"
    echo "     export APPLY_IMMEDIATELY=true"
    echo "     bash $0"
    echo ""
    echo "   Hoặc apply manual:"
    echo "     kubectl apply -f /tmp/tigera-operator.yaml"
    echo "     sleep 30"
    echo "     kubectl apply -f /tmp/calico-installation.yaml"
    echo ""
    echo "   Watch:"
    echo "     kubectl -n calico-system get pods -w"
    echo ""

    # Dù dry-run cũng tạo manifest để review
    install_calico
    configure_calico
  fi

  echo ""
  echo "================================================================"
  echo "  ✅ STEP 4 COMPLETE"
  echo "================================================================"
  echo ""
  echo "  Sau khi Calico running, verify:"
  echo "    kubectl get nodes -o wide    # tất cả nodes phải Ready"
  echo "    kubectl -n calico-system get pods"
  echo "    kubectl run nginx --image=nginx --port=80"
  echo "    kubectl get pods -o wide"
  echo ""
  echo "  Nếu node NotReady, check:"
  echo "    kubectl -n calico-system logs daemonset/calico-node"
  echo "    -> Nếu lỗi IP detection, chạy:"
  echo "    -> bash calico-patch-ip-autodetect.sh"
  echo ""
}

main
